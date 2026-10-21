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

// Full Strip and Disarm share the equipment-particle burst.
func (m *WorldMode) drawEquipmentBurst(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if now.Before(effect.starts) || frame > 108 {
		return
	}
	for kind, path := range component.textureFiles {
		texture := m.effectFileTexture(ctx.Resources, path)
		if texture == nil {
			continue
		}
		count, size := 16, float64(5+kind*4)*math.Sqrt2/5
		if effect.effectID == effectRGCoin3 {
			count, size = 4, math.Sqrt2
		}
		for i := 0; i < count; i++ {
			seed := (kind*16 + i) * 4
			n := frame + 1 - 18 - hitEffectRandom(effect, seed, 30)
			alpha := min(n, 10)*25 - max(0, n-25)*7
			if n <= 0 || alpha <= 0 {
				continue
			}
			pitch := degreesToRadians(float64(hitEffectRandom(effect, seed+1, 360)))
			yaw := degreesToRadians(float64(hitEffectRandom(effect, seed+2, 360)))
			center := add3(effect.actorOrigin, effectParticlePoint(float64(12+n), 0, pitch, yaw))
			roll := degreesToRadians(float64(hitEffectRandom(effect, seed+3, 360) + 3*(frame+1)))
			drawTexturedEffectBillboardRotatedXY(screen, projection, texture, center.x, center.z, center.y, size, size, roll, color.RGBA{250, 100, 100, uint8(alpha)}, true)
		}
	}
}

type wanderingEffectParticle struct {
	position        modelPoint3
	angles          [4]int
	seed, age, kind int
	radius, speed   float64
}

func advanceWanderingAngles(particle *wanderingEffectParticle, effect worldEffect, tick int) {
	for axis := 0; axis < 4; axis += 2 {
		a, target := &particle.angles[axis], &particle.angles[axis+1]
		if *target > *a {
			*a = min(*a+3, 359)
			if *a > *target {
				*target = hitEffectRandom(effect, particle.seed+tick*4+axis, max(*a, 1))
			}
		} else {
			*a = max(*a-2, 0)
			if *a <= *target {
				*target = 359 - hitEffectRandom(effect, particle.seed+tick*4+axis, max(*a, 1))
			}
		}
	}
}

type risingBuffParticles struct {
	origins [21]modelPoint3
	count   int
}

func (m *WorldMode) drawRisingBuffParticles(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if now.Before(effect.starts) || len(component.textureFiles) != 2 {
		return
	}
	textures := [2]*render.Image{
		m.effectFileTexture(ctx.Resources, component.textureFiles[0]),
		m.effectFileTexture(ctx.Resources, component.textureFiles[1]),
	}
	start, end, step, life := 0, 80, 4, 55
	jitter, radius, radiusStep := 11, 2.0, .1
	if effect.effectID == effectSmaReady {
		start, end = 40, 120
	} else if effect.effectID == effectStin3 {
		end, step, life = 199, 1, 50
		jitter, radius, radiusStep = 9, 1.2, .08
	}
	if effect.effectID != effectStin3 {
		if effect.buffParticles == nil {
			return
		}
		// Keep each emission's position when the actor replaces its movement path.
		p := effect.buffParticles
		for p.count < len(p.origins) && start+p.count*step <= min(frame, end) {
			at := effect.starts.Add(time.Duration(start+p.count*step) * db.EffectFrameDuration)
			p.origins[p.count] = effectWorldAnchor(ctx, effect, at)
			p.count++
		}
	}
	// ParticleTime and SMA both emit four independently rising particles.
	first := max(start, (frame-life+step)/step*step)
	for emitted := first; emitted <= min(frame, end); emitted += step {
		age := frame - emitted + 1
		alpha := float64(max(15*(min(age, 10)-1)-3*max(age-10, 0), 0)) / 255
		if alpha <= 0 {
			continue
		}
		var center modelPoint3
		if effect.effectID == effectStin3 {
			center = add3(effect.actorOrigin, mul3(effect.projectileDirection, .8*float64(max(emitted-2, 0))/5))
		} else {
			center = effect.buffParticles.origins[(emitted-start)/step]
		}
		for i := 0; i < 4; i++ {
			salt := (emitted*4 + i) * 5
			position := add3(center, modelPoint3{
				x: float64(hitEffectRandom(effect, salt+2, jitter)-jitter/2) / 5,
				z: float64(hitEffectRandom(effect, salt+3, jitter)-jitter/2) / 5,
				y: (6 + float64(age)*(.1+.01*float64(hitEffectRandom(effect, salt+4, 20)))) / 5,
			})
			size := math.Sqrt2 * (radius + radiusStep*float64(hitEffectRandom(effect, salt+5, 10))) / 5
			index := 0
			if age == 35-i*7 || age == 36-i*7 {
				index = 1
			}
			drawTexturedEffectBillboardRotatedXY(screen, projection, textures[index],
				position.x, position.z, position.y, size, size, degreesToRadians(float64(-5*age+45)), effectComponentTint(component, alpha), true)
		}
	}
}
