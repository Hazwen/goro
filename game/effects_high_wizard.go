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

func (m *WorldMode) applyHighWizardSkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	switch notify.SkillID {
	case db.SkillHWSouldrain:
		m.addWorldEffectBetweenAt(ctx, effectEnergyDrain2, notify.SourceID, notify.TargetID, now)
	case db.SkillHWMagicpower:
		id := notify.TargetID
		if isLocalActor(ctx, id) {
			id = localSkillTarget(ctx)
		}
		if _, ok, _ := actorForCombatID(ctx, id); ok {
			m.removeWorldEffect(effectLightBlade, id)
			m.setActorStateEffect(ctx, id, effectLightBlade, true)
			m.scheduleSound(now, "effect\\마법력 증폭.wav")
		}
	}
}

func initSoulDrain(ctx client.Context, effect *worldEffect) {
	caster := effectWorldAnchor(ctx, *effect, effect.starts)
	effect.actorOrigin = caster
	if _, ok, _ := actorForCombatID(ctx, effect.targetID); effect.targetID != 0 && ok {
		effect.actorOrigin = effectWorldAnchor(ctx, worldEffect{actorID: effect.targetID}, effect.starts)
		effect.actorOrigin.y = caster.y
	}
	effect.projectileDirection = sub3(caster, effect.actorOrigin)
	effect.hasActorTransform = true
}

func soulDrainPosition(effect worldEffect, tick int, roll float64) modelPoint3 {
	if tick == 0 {
		return add3(effect.actorOrigin, effect.projectileDirection)
	}
	n := float64(tick)
	direction := normalize3(effect.projectileDirection)
	forward := sub3(mul3(effect.projectileDirection, n/60), mul3(direction, (2*n-n*(n+1)/30)/5))
	arc := (2*n - n*(n-1)/30) / 5
	s, c := math.Sincos(degreesToRadians(roll))
	p := add3(effect.actorOrigin, forward)
	p.y += 2 + arc*c
	p.x -= direction.z * arc * s
	p.z += direction.x * arc * s
	return p
}

func (m *WorldMode) drawSoulDrain(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if now.Before(effect.starts) || frame >= 60 {
		return
	}
	for i := 0; i < 3; i++ {
		tint := color.RGBA{uint8(160 + hitEffectRandom(effect, i*3, 81)), uint8(160 + hitEffectRandom(effect, i*3+1, 81)), 255, 254}
		size := 1.5 + .1*float64(hitEffectRandom(effect, i*3+2, 6))
		for trail := 0; trail < 7; trail++ {
			tint.A = uint8(254 * (1 - float64(trail)/7))
			p := soulDrainPosition(effect, max(0, frame+1-trail), float64(-90+i*90))
			m.drawEffectParticleSprite(screen, ctx, projection, component.spriteFile, p, frame/4+1, size*(1-float64(trail)/14), 0, tint)
		}
	}
}

// Prim3DQuadHorn2 alternates three descending and three ascending ticks.
// Each six-tick cycle has zero net travel; after tick 550 it retracts.
func gravitationTravel(tick int) float64 {
	if tick <= 0 {
		return 0
	}
	if tick == 1 {
		return 1.01
	}
	remainder := (min(tick, 550) - 2) % 6
	return 2.03 + [...]float64{0, -1.2, -2.4, -3.6, -2.41, -1.21}[remainder] - 1.2*float64(max(0, tick-550))
}

func (m *WorldMode) drawGravitation(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, origin modelPoint3, now time.Time) {
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	if n <= 0 || n >= 2000 {
		return
	}
	origin.y -= 2
	alpha := uint8(min(20, 2*(2000-n)))
	for group, file := range [...]string{component.textureFile, component.textureFiles[0]} {
		texture := m.effectFileTexture(ctx.Resources, file)
		if texture == nil {
			continue
		}
		var vertices []render.Vertex3D
		var indices []uint16
		for i := group * 2; i < group*2+2; i++ {
			size := float64(300+hitEffectRandom(effect, i*3, 50)) / 100
			yaw := degreesToRadians(float64(hitEffectRandom(effect, i*3+1, 360)))
			pitch := degreesToRadians(float64(60 + hitEffectRandom(effect, i*3+2, 40)))
			appendEffectSpike(&vertices, &indices, texture, origin, size, 18, pitch, yaw, gravitationTravel(n), color.RGBA{255, 255, 255, alpha})
		}
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(false, false))
	}
}
