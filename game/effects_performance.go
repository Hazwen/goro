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

func (m *WorldMode) applyPerformanceGroundSkill(ctx client.Context, notify network.GroundSkillNotify, now time.Time) bool {
	if !isBardPerformance(notify.SkillID) && !isDancerPerformance(notify.SkillID) && notify.SkillID != db.SkillCGHermode && notify.SkillID != db.SkillCGMoonlit {
		return false
	}
	if actor, ok, _ := actorForCombatID(ctx, notify.SourceID); ok {
		m.startSkillActionAnimation(ctx, notify.SourceID, actor, skillAction(notify.SkillID), now, defaultAttackAnimationDuration)
	}
	for _, sound := range db.SkillEffects[notify.SkillID].SFX {
		m.scheduleSoundAtActor(now, notify.SourceID, sound)
	}
	if notify.SkillID == db.SkillCGMoonlit {
		m.applyClownGypsySkill(ctx, network.SkillNoDamageNotify{SkillID: notify.SkillID, SourceID: notify.SourceID}, now)
	}
	// Skill-unit packets own each symbol and its lifetime.
	return true
}

func (m *WorldMode) stopPerformance(ctx client.Context, change network.StatusEffectChange) {
	if change.StatusID != db.StatusBdplaying || change.Active {
		return
	}
	id := change.ActorID
	if id == 0 {
		id = localSkillTarget(ctx)
	}
	ids := []uint32{id}
	if isLocalActor(ctx, id) {
		ids = []uint32{ctx.Session.AccountID, ctx.Session.CharID}
	}
	for _, id := range ids {
		if anim, ok := m.actorAnims[id]; ok && anim.loop && anim.actionFamily == spriteActionPCSkill && anim.hasFrameOffset && anim.frameOffset == 1 && anim.length == 3 {
			m.clearActorAction(ctx, id)
		}
	}
}

func (m *WorldMode) drawSongMusic(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, origin modelPoint3, now time.Time) {
	file := component.textureFile
	if len(component.textureFiles) > 0 {
		file = component.textureFiles[hitEffectRandom(effect, 0, len(component.textureFiles))]
	}
	texture := m.effectFileTexture(ctx.Resources, file)
	if texture == nil {
		return
	}
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	variant := component.animation
	radius, count := 3.0, 1
	tint := color.RGBA{130, 130, 250, 200}
	switch variant {
	case 1:
		radius = 5
	case 2:
		count = 4
	case 4:
		radius = 15
	case 3:
		tint = color.RGBA{200, 200, 100, 200}
	case 5, 6, 7:
		tint = color.RGBA{255, 255, 255, 200}
	}
	if variant == 6 {
		radius = 8
	}
	for i := 0; i < count; i++ {
		age := n
		if i > 0 {
			if n < 6*i {
				continue
			}
			age = 6*(n/6) - 1 - 6*(i-1)
		}
		p := origin
		p.y += (6 - 3*math.Sin(degreesToRadians(float64(hitEffectRandom(effect, 1, 360)+age)))) / 5
		size := math.Sqrt2 * (radius + .5*float64(i)) / 5 * (1 + .05*math.Sin(degreesToRadians(float64(5*age))))
		if variant == 4 {
			right, _, _, _ := projection.BillboardBasis(p.x, p.z, p.y)
			p = add3(p, mul3(right, .8))
		}
		angle := 0.0
		if variant == 7 {
			angle = degreesToRadians(float64(10 * age))
		}
		tint.A = uint8(200 - 50*i)
		drawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, p.x, p.z, p.y, size, size, angle, tint, texturedEffectBillboardDrawOptions(variant != 5 && variant != 6, false))
	}
}

func songOutParticle(n, initial int) (alpha int, height float64) {
	firstEnd := max(1, 6*(10-initial))
	if n < firstEnd {
		fade := max(0, n+initial-10)
		return 25*min(n, 10-initial) - 5*fade, .2 * float64(fade) / 5
	}
	phase := (n - firstEnd) % 60
	fade := max(0, phase-10)
	return 25*min(phase, 10) - 5*fade, .2 * float64(1+fade) / 5
}

func (m *WorldMode) drawSongGround(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, origin modelPoint3, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	var vertices []render.Vertex3D
	var indices []uint16
	variant := component.animation
	additive := true
	switch component.funcAdapter {
	case effectFuncSongVertical:
		count := 4
		if variant == 4 {
			count = 2
		}
		for i := 0; i < count; i++ {
			salt := i * 3
			from, to := origin, origin
			rate := 2
			tint := color.RGBA{255, 255, 255, 15}
			if variant == 1 || variant == 2 {
				x, z := float64(hitEffectRandom(effect, salt, 7)-3)/5, -.6
				if i >= 2 {
					x, z = z, x
				}
				from.x -= x
				from.z -= z
				to.x += x
				to.z += z
				rate, additive = 4, false
				tint = color.RGBA{255, 155, 155, 50}
				if variant == 2 {
					rate = 2
					tint = color.RGBA{155, 255, 155, 50}
				}
			} else if variant == 4 {
				if i == 0 {
					from.x -= 1.2
					to.x += 1.2
				} else {
					from.z += 1.2
					to.z -= 1.2
				}
				tint = color.RGBA{255, 150, 150, 100}
			} else {
				to.x += float64(hitEffectRandom(effect, salt, 40)-20) / 5
				to.z += float64(hitEffectRandom(effect, salt+1, 40)-20) / 5
			}
			height := (20 + 10*math.Sin(degreesToRadians(float64(hitEffectRandom(effect, salt+2, 360)+rate*n)))) / 5
			up := modelPoint3{y: height}
			appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{add3(from, up), add3(to, up), to, from}, tint)
		}
	case effectFuncSongLight:
		radius := (3 + math.Sin(degreesToRadians(float64(hitEffectRandom(effect, 0, 360)+n)))) / 5
		height := float64(25+hitEffectRandom(effect, 1, 10)) / 5 * (1 + .3*math.Sin(degreesToRadians(float64(n))))
		tint := color.RGBA{100, 100, 255, uint8(50 - hitEffectRandom(effect, 2, 10))}
		if variant == 8 {
			tint.R, tint.G, tint.B = 255, 255, 170
		}
		appendEffectBand(&vertices, &indices, texture, origin, radius,
			degreesToRadians(float64(hitEffectRandom(effect, 3, 360)+3*n)), degreesToRadians(315), 0, 21, tint,
			func(j int) float64 { return height * math.Sin(degreesToRadians(float64(min(j, 20)*9))) })
		for j := 0; j < 21; j++ {
			for k, u := range []int{min(j, 19), min(j+1, 20), min(j+1, 20), min(j, 19)} {
				vertices[4*j+k].SrcX = float32(u) / 21 * float32(texture.Bounds().Dx())
			}
		}
	case effectFuncSongOut:
		for i := 0; i < 2; i++ {
			alpha, height := songOutParticle(n, hitEffectRandom(effect, i*2, 11))
			p := origin
			p.y += height
			size := math.Sqrt2 * float64(5+hitEffectRandom(effect, i*2+1, 6)) / 5 * (1 + .05*math.Sin(degreesToRadians(float64(5*n))))
			drawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, p.x, p.z, p.y, size, size, 0, color.RGBA{130, 130, 250, uint8(alpha)}, texturedEffectBillboardDrawOptions(true, false))
		}
		return
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(additive, false))
	}
}

func danceSkillActionSpec() skillActionSpec {
	spec := newSkillActionSpec(skillActorActionSkill, true, nil)
	spec.hasFrame = true
	spec.frame = 1
	spec.length = 3
	spec.speed = 10 * db.EffectFrameDuration
	spec.pingPong = true
	return spec
}
