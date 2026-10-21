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

func (m *WorldMode) applyProfessorSkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	switch notify.SkillID {
	case db.SkillPFSoulchange:
		m.addWorldEffectBetweenAt(ctx, effectLineLink2, notify.SourceID, notify.TargetID, now)
	case db.SkillPFDoublecasting:
		if _, ok, _ := actorForCombatID(ctx, notify.TargetID); ok {
			m.scheduleSound(now, "effect\\마법력 증폭.wav")
		}
	}
}

func (m *WorldMode) drawHPConversion(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if now.Before(effect.starts) || frame >= 64 {
		return
	}
	// EnergyDrain3 emits three PP_3DPARTICLESPLINE souls on each of five ticks.
	for i := 0; i < 15; i++ {
		age := frame - i/3
		if age < 0 || age >= 60 {
			continue
		}
		direction := normalize3(modelPoint3{x: float64(hitEffectRandom(effect, i*5, 7) - 3), z: float64(hitEffectRandom(effect, i*5+1, 7) - 3)})
		if direction == (modelPoint3{}) {
			direction.z = 1
		}
		tint := color.RGBA{uint8(160 + hitEffectRandom(effect, i*5+2, 81)), 255, uint8(160 + hitEffectRandom(effect, i*5+3, 81)), 254}
		size := 1 + .1*float64(hitEffectRandom(effect, i*5+4, 6))
		s, c := math.Sincos(degreesToRadians(float64(-90 + i%3*90)))
		for trail := 0; trail < 7; trail++ {
			n := float64(max(0, age+1-trail))
			p := effect.actorOrigin
			if n > 0 {
				forward := (2*n - n*(n+1)/30) / 5
				arc := (2*n - n*(n-1)/30) / 5
				p = add3(p, mul3(direction, forward))
				p.y += 2 + arc*c
				p.x += direction.z * arc * s
				p.z -= direction.x * arc * s
			}
			tint.A = uint8(254 * (1 - float64(trail)/7))
			m.drawEffectParticleSprite(screen, ctx, projection, component.spriteFile, p, age/4+1, size*(1-float64(trail)/14), 0, tint)
		}
	}
}

func (m *WorldMode) drawSoulChangeLink(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	alpha := 6*min(n, 19) - 3*max(0, n-40)
	if n <= 0 || alpha <= 0 {
		return
	}
	if _, ok, _ := actorForCombatID(ctx, effect.targetID); !ok {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	from := effectWorldAnchor(ctx, effect, now)
	target := effect
	target.actorID = effect.targetID
	to := effectWorldAnchor(ctx, target, now)
	from.y, to.y = from.y+1.6, to.y+1.6
	if n < 15 {
		to = add3(from, mul3(sub3(to, from), math.Sin(degreesToRadians(float64(n*6)))))
	}
	var vertices []render.Vertex3D
	var indices []uint16
	appendMonkTrail(&vertices, &indices, texture, from, to, .2/5, 10, color.RGBA{0, 100, 150, uint8(alpha)})
	appendMonkTrail(&vertices, &indices, texture, from, to, .4/5, 10, color.RGBA{255, 50, 0, uint8(alpha * 2)})
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
}
