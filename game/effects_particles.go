package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) drawEffectParticleSprite(screen *render.Frame, ctx client.Context, projection sceneProjection, file string, position modelPoint3, motion int, scale, angle float64, tint color.RGBA) {
	m.drawEffectParticleSpriteAction(screen, ctx, projection, file, position, 0, motion, scale, angle, tint)
}

func (m *WorldMode) drawEffectParticleSpriteAction(screen *render.Frame, ctx client.Context, projection sceneProjection, file string, position modelPoint3, action, motion int, scale, angle float64, tint color.RGBA) {
	if tint.A == 0 || scale <= 0 {
		return
	}
	view := m.effectSpriteView(ctx.Resources, file)
	if view == nil || action < 0 || action >= len(view.act.Actions) || len(view.act.Actions[action].Animations) == 0 {
		return
	}
	motion %= len(view.act.Actions[action].Animations)
	key := singleSpriteBillboardKey{actionIndex: action, motion: motion}
	billboard, ok := view.billboards[key]
	if !ok {
		billboard, ok = composeSingleSpriteBillboard(view, view.act.Actions[action].Animations[motion])
		if !ok {
			return
		}
		view.billboards[key] = billboard
	}
	alpha := float64(tint.A) / 255
	tint.A = 255
	drawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard, position.x, position.z, position.y,
		scale*effectPixelRatio, angle, alpha, 1, tint, spriteBillboardTriangleDrawOptions())
}

type effectParticle struct {
	position           modelPoint3
	size, angle, alpha float64
}

func risingParticleAt(effect worldEffect, index, age int) effectParticle {
	life, fade, initial := 40, 32, 0.0
	speed := float64(30+hitEffectRandom(effect, index*4+1, 30)) / 500
	size := float64(40+hitEffectRandom(effect, index*4+2, 50)) / 100
	yaw := degreesToRadians(float64(hitEffectRandom(effect, index*4+3, 360)))
	radius := float64(2+hitEffectRandom(effect, index*4+4, 6)) / 5
	if effect.effectID == effectEDP {
		speed = float64(30+hitEffectRandom(effect, index*4+1, 50)) / 500
		size = float64(20+hitEffectRandom(effect, index*4+2, 20)) / 100
	}
	if effect.effectID == effectVenomDust2 {
		life, fade, initial = 10, 8, 70
		speed = float64(20+hitEffectRandom(effect, index*4+1, 20)) / 400
		size, radius = .8, 0
	}
	if age < 0 || age >= life {
		return effectParticle{}
	}
	n := float64(age + 1)
	alpha := math.Min(254, initial+25.4*n)
	if age >= fade {
		alpha = math.Min(254, initial+25.4*float64(fade)) * float64(life-age-1) / float64(life-fade)
	}
	return effectParticle{
		position: modelPoint3{x: math.Sin(yaw) * radius, y: n*speed - speed/(2*float64(life))*n*(n+1)/2, z: math.Cos(yaw) * radius},
		size:     size, angle: degreesToRadians(n * 3), alpha: alpha / 255,
	}
}

func (m *WorldMode) drawRisingParticlesEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	view := m.effectSpriteView(ctx.Resources, component.spriteFile)
	if view == nil || len(view.act.Actions) == 0 || len(view.act.Actions[0].Animations) == 0 {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	life, last := 40, min(frame/5, 42)
	interval := 5
	origin := effect.actorOrigin
	if effect.effectID == effectVenomDust2 {
		life, last = 10, frame/5
	} else if effect.effectID == effectDetoxication {
		// ProcessAlpha puts particles that would outlive the emitter straight
		// into their final fade, starting from zero alpha.
		last = min(frame/5, 12)
	} else if effect.effectID == effectEDP {
		interval, last = 3, min(frame/3, 26)
		origin = effectWorldAnchor(ctx, effect, now)
	}
	options := triangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
	for i := max(0, (frame-life+interval)/interval); i <= last; i++ {
		age := frame - i*interval
		particle := risingParticleAt(effect, i, age)
		if particle.alpha <= 0 {
			continue
		}
		motion := (age/4 + 1) % len(view.act.Actions[0].Animations)
		key := singleSpriteBillboardKey{motion: motion}
		billboard, ok := view.billboards[key]
		if !ok {
			billboard, ok = composeSingleSpriteBillboard(view, view.act.Actions[0].Animations[motion])
			if !ok {
				continue
			}
			view.billboards[key] = billboard
		}
		position := add3(origin, particle.position)
		drawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard,
			position.x, position.z, position.y, particle.size*effectPixelRatio, -particle.angle, particle.alpha, 1, color.RGBA{255, 255, 255, 255}, options)
	}
}
