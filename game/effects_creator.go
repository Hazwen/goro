package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) applyCreatorGroundSkill(ctx client.Context, notify network.GroundSkillNotify, now time.Time) bool {
	if notify.SkillID != db.SkillCRSlimpitcher && notify.SkillID != db.SkillCRCultivation {
		return false
	}
	m.faceSkillSource(ctx, notify.SourceID, 0, int(notify.X), int(notify.Y))
	if actor, ok, _ := actorForCombatID(ctx, notify.SourceID); ok {
		m.startSkillActionAnimation(ctx, notify.SourceID, actor, skillAction(notify.SkillID), now, defaultAttackAnimationDuration)
	}
	m.applySkillNameBubble(ctx, notify.SourceID, notify.SkillID, now)
	if notify.SkillID == db.SkillCRCultivation {
		m.addWorldEffectAt(ctx, effectBeginSpell6, notify.SourceID, now)
		return true
	}
	effectID := effectSlim
	if notify.Level >= 10 {
		effectID = effectSlim3
	} else if notify.Level >= 6 {
		effectID = effectSlim2
	}
	m.addWorldEffectAtCellLifetime(ctx, effectID, 0, int(notify.X), int(notify.Y), now, 0, false)
	// The native sound is centered at a fixed distance of 150, with a 40-unit minimum.
	m.scheduleSoundVolume(now, 40.0/150, "assulter_attack.wav")
	return true
}

func (m *WorldMode) drawSlimPitcher(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	origin := effectWorldAnchor(ctx, effect, now)
	variant := effect.effectID - effectSlim
	colors := [...]color.RGBA{{255, 135, 135, 255}, {255, 255, 135, 255}, {255, 255, 255, 255}}
	if texture := m.effectFileTextureWithBlackKey(ctx.Resources, component.textureFile, false); texture != nil {
		for trail := 0; trail < 4; trail++ {
			process := n - trail
			if process <= 0 {
				continue
			}
			alpha := min(process, 10)*20 - max(0, process-32)*5
			if trail > 0 {
				alpha -= 100 + 25*(trail-1)
			}
			if alpha <= 0 {
				continue
			}
			p := origin
			p.y += 16 - .6*float64(min(process, 24))
			angle := deterministicAngle(effect, 0) - degreesToRadians(float64(15*min(process, 24)))
			tint := colors[variant]
			tint.A = uint8(alpha)
			size := 8 * math.Sqrt2 / 5
			drawTexturedEffectBillboardRotatedXY(screen, projection, texture, p.x, p.z, p.y, -size, size, angle-3*math.Pi/4, tint, false)
		}
	}
	age := n - 16
	alpha := min(age, 10)*2 - max(0, age-10)
	texture := m.effectFileTexture(ctx.Resources, component.textureFiles[0])
	if age <= 0 || alpha <= 0 || texture == nil {
		return
	}
	origin.y++
	radius := 2.7 * math.Pow(1.07, float64(age))
	tint := [...]color.RGBA{{255, 0, 0, uint8(alpha)}, {255, 255, 0, uint8(alpha)}, {255, 255, 255, uint8(alpha)}}[variant]
	var vertices []render.Vertex3D
	var indices []uint16
	layers := 1
	if variant == 0 {
		layers = 2
	}
	for i := 0; i < layers; i++ {
		appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{
			add3(origin, modelPoint3{x: radius}), add3(origin, modelPoint3{z: radius}),
			add3(origin, modelPoint3{x: -radius}), add3(origin, modelPoint3{z: -radius}),
		}, tint)
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
}

func (m *WorldMode) drawAcidDemon(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	alpha := min(80, n*5) - 15*max(0, n-39)
	if n <= 0 || alpha <= 0 {
		return
	}
	texture := m.effectFileTextureWithBlackKey(ctx.Resources, component.textureFile, false)
	if texture == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for layer, r := range [...]struct{ radius, height, rise, rotation float64 }{
		{2.4, 25, 70, 0}, {2.7, 22, 57, 90}, {3, 19, 45, 180}, {2.2, 30, 90, 0},
	} {
		// Render3DCasting divides 315 degrees by 20 using integer arithmetic.
		growth, step := 1.005, 15
		if layer == 3 {
			growth, step = 1.025, 18
		}
		for tick := 0; tick < n; tick++ {
			if layer == 3 {
				r.rotation += float64(2 + hitEffectRandom(effect, tick*4+layer, 2))
			} else {
				r.rotation += float64(4 + hitEffectRandom(effect, tick*4+layer, 3))
			}
		}
		r.height *= math.Pow(growth, float64(n)) * math.Sin(degreesToRadians(float64(n)))
		riseSin, riseCos := math.Sincos(degreesToRadians(r.rise))
		base := uint16(len(vertices))
		for segment := 0; segment <= 20; segment++ {
			sin, cos := math.Sincos(degreesToRadians(r.rotation + float64(segment*step)))
			height := r.height * math.Sin(degreesToRadians(float64(segment*9))) / 5
			bottom := add3(effect.actorOrigin, modelPoint3{x: cos * r.radius / 5, z: sin * r.radius / 5})
			top := add3(bottom, modelPoint3{x: cos * riseCos * height, y: riseSin * height, z: sin * riseCos * height})
			u := float32(segment) / 21
			tint := color.RGBA{255, 50, 255, uint8(alpha)}
			w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
			vertices = append(vertices, texturedSurfaceVertex3D(bottom, texturePoint{u: u, v: 1}, tint, w, h), texturedSurfaceVertex3D(top, texturePoint{u: u}, tint, w, h))
			if segment < 20 {
				b := base + uint16(segment*2)
				indices = append(indices, b, b+1, b+2, b+1, b+3, b+2)
			}
		}
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(false, false))
}
