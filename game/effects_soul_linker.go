package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

type soulLinkerParticles struct {
	origins [21]modelPoint3
	count   int
}

func initSoulLinkerProjectile(ctx client.Context, effect *worldEffect) {
	destination := effectWorldAnchor(ctx, *effect, effect.starts)
	origin := destination
	if _, ok, _ := actorForCombatID(ctx, effect.targetID); effect.targetID != 0 && ok {
		origin = effectWorldAnchor(ctx, worldEffect{actorID: effect.targetID}, effect.starts)
	}
	direction := sub3(destination, origin)
	direction.y = 0
	if distance := math.Hypot(direction.x, direction.z); distance > 0 {
		direction = mul3(direction, 1/distance)
	}
	// Estin/Estun keep their cast positions; Esma keeps the launch position
	// while homing toward its target.
	if effect.effectID == effectStin2 {
		origin.y += 2
		effect.esma = &esmaProjectile{}
	} else {
		origin.y += float64(10-hitEffectRandom(*effect, 1, 5)) / 5
	}
	effect.actorOrigin = origin
	effect.projectileDirection = direction
	effect.hasActorTransform = true
}

// dhxj PrimSTIN: three startup ticks, nine accelerating fade-in ticks,
// then faster acceleration while fading. Distances are converted to cells.
func estinHead(process int) (distance, alpha float64) {
	if process <= 0 {
		return 0, 0
	}
	rise := min(process-1, 9)
	fall := max(process-10, 0)
	distance = .8 * (1 + 1.02*(math.Pow(1.02, float64(rise))-1)/.02)
	distance += .8 * math.Pow(1.02, 9) * 1.04 * (math.Pow(1.04, float64(fall)) - 1) / .04
	return distance / 5, float64(max(25*rise-5*fall, 0)) / 255
}

func (m *WorldMode) drawEstinEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if now.Before(effect.starts) || frame >= 200 {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	process := frame + 1 - 3
	var vertices []render.Vertex3D
	var indices []uint16
	for trail := 0; trail < 4; trail++ {
		sample := process
		if trail > 0 {
			// PrimSTIN copies the head into a three-slot history every third tick.
			sample = min(process, 56)/3*3 - (trail-1)*3
		}
		distance, alpha := estinHead(sample)
		if alpha <= 0 {
			continue
		}
		alpha *= math.Pow(.8, float64(trail))
		center := add3(effect.actorOrigin, mul3(effect.projectileDirection, distance))
		var corners [4]modelPoint3
		for i := range corners {
			angle := degreesToRadians(float64(sample*30 + (i+1)*90))
			corners[i] = add3(center, modelPoint3{x: 4 * math.Cos(angle), z: 4 * math.Sin(angle)})
		}
		appendEffectQuad(&vertices, &indices, texture, corners, effectComponentTint(component, alpha))
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
	}
}

func (m *WorldMode) drawSoulLinkerParticles(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
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

func (m *WorldMode) drawSoulLinkerHitLines(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if now.Before(effect.starts) || frame >= 200 {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	// HitLine4(3) for Estun; HitLine3(3) for Esma.
	count, start, randomDelay, fixedDelay, baseRadius, heightBase := 16, 6, 31, 0, .05/5, .8
	if effect.effectID == effectHitLine6 {
		count, start, randomDelay, fixedDelay, baseRadius, heightBase = 20, 0, 41, 30, .07/5, .6
	}
	for i := 0; i < count; i++ {
		salt := i * 5
		age := frame - (start + i/4) - fixedDelay - hitEffectRandom(effect, salt, randomDelay) + 1
		if age <= 0 || age >= 34 {
			continue
		}
		angle := deterministicAngle(effect, salt+1)
		sin, cos := math.Sincos(angle)
		distance := float64(6+hitEffectRandom(effect, salt+2, 6)) / 5
		height := heightBase + .1*float64(hitEffectRandom(effect, salt+3, 7))
		alpha := float64(125 + hitEffectRandom(effect, salt+4, 121))
		point := func(tick int) modelPoint3 {
			phase := float64(clampInt(tick, 0, 30) * 6)
			travel := phase / 90 * distance
			return modelPoint3{x: x + cos*travel, y: z + 1 + distance*math.Sin(degreesToRadians(phase))*height, z: y + sin*travel}
		}
		for trail := 0; trail < 4; trail++ {
			// PrimHitLine3 updates history after advancing its head, so the
			// first history slot has the same position as the head.
			a, b := point(age-max(0, trail-1)), point(age-trail)
			if a == b {
				continue
			}
			for layer := 0; layer < 2; layer++ {
				radius := baseRadius
				brightness := uint8(255 - (5-trail)*30)
				tint := color.RGBA{brightness, brightness, brightness, uint8(alpha - float64(trail*10))}
				if layer == 1 {
					radius *= 4
					tint = color.RGBA{0, 0, 255, tint.A / 2}
				}
				offset := func(side int) modelPoint3 {
					s, c := math.Sincos(degreesToRadians(float64(side * 120)))
					return modelPoint3{x: -sin * radius * c, y: -radius * s, z: cos * radius * c}
				}
				for side := 0; side < 3; side++ {
					p, q := offset(side), offset(side+1)
					appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{add3(a, q), add3(b, q), add3(b, p), add3(a, p)}, tint)
				}
			}
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
	}
}

type esmaSample struct {
	position modelPoint3
	alpha    float64
	spin     float64
}

type esmaHead struct {
	samples [4]esmaSample
	heading float64
	fading  bool
}

type esmaProjectile struct {
	tick  int
	heads [2]esmaHead
}

// PrimSTIN2 turns the two heads in opposite directions at 60 Hz. Retaining
// their small histories avoids bending an old trail when the target moves.
func (p *esmaProjectile) advance(ctx client.Context, effect worldEffect, now time.Time) {
	end := min(int(now.Sub(effect.starts)/db.EffectFrameDuration)+1, 200)
	for p.tick < end {
		p.tick++
		at := effect.starts.Add(time.Duration(p.tick-1) * db.EffectFrameDuration)
		target := effectWorldAnchor(ctx, effect, at)
		_, targetExists, _ := actorForCombatID(ctx, effect.actorID)
		for side := range p.heads {
			head := &p.heads[side]
			turn := 1.0
			if side == 0 {
				turn = -1
			}
			if p.tick == 1 {
				for i := range head.samples {
					head.samples[i].position = effect.actorOrigin
				}
				head.heading = math.Atan2(target.z-effect.actorOrigin.z, target.x-effect.actorOrigin.x)*180/math.Pi - turn*90
			}
			if !targetExists || p.tick > 140 {
				head.fading = true
			}
			sample := &head.samples[0]
			if head.fading && sample.alpha <= 0 {
				continue
			}
			sample.spin = float64(p.tick * 30)
			if p.tick > 1 && p.tick <= 20 {
				sample.alpha = min(sample.alpha+15, 250)
			}
			if p.tick > 5 {
				if head.fading {
					for i := range head.samples {
						head.samples[i].alpha = max(head.samples[i].alpha-5, 0)
					}
					head.heading += turn
				} else {
					toward := math.Atan2(target.z-sample.position.z, target.x-sample.position.x) * 180 / math.Pi
					toward = math.Mod(toward+360, 360)
					if math.Abs(head.heading-toward) > 3 {
						head.heading += turn * 5
					}
				}
			}
			head.heading = math.Mod(head.heading+360, 360)
			sin, cos := math.Sincos(degreesToRadians(head.heading))
			sample.position.x += .4 * cos
			sample.position.z += .4 * sin
			if p.tick%3 == 0 {
				for i := 3; i > 0; i-- {
					head.samples[i] = head.samples[i-1]
					head.samples[i].alpha *= .8
				}
			}
		}
	}
}

func (m *WorldMode) drawEsmaEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	if now.Before(effect.starts) || effect.esma == nil {
		return
	}
	effect.esma.advance(ctx, effect, now)
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for _, head := range effect.esma.heads {
		for _, sample := range head.samples {
			if sample.alpha <= 0 {
				continue
			}
			var corners [4]modelPoint3
			for i := range corners {
				sin, cos := math.Sincos(degreesToRadians(sample.spin + float64((i+1)*90)))
				corners[i] = add3(sample.position, modelPoint3{x: 5 * cos, z: 5 * sin})
			}
			appendEffectQuad(&vertices, &indices, texture, corners, effectComponentTint(component, sample.alpha/255))
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
	}
}

func (m *WorldMode) drawEsmaHitTexture(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	age := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	alpha := 8 * min(age-40, 82-age)
	if now.Before(effect.starts) || alpha <= 0 {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	size := math.Sqrt2 * (10 + .7*float64(age-40)) / 5
	size *= 1 + .05*math.Sin(degreesToRadians(float64(age*2)))
	// RenderEffectTexture's hit branch falls through the white and grey passes.
	for _, tint := range [...]color.RGBA{{130, 130, 250, uint8(alpha)}, {255, 255, 255, uint8(alpha)}, {150, 150, 150, uint8(alpha)}, {50, 50, 50, uint8(alpha)}} {
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture, x, y, effect.actorOrigin.y+2, size, size, 0, tint, true)
	}
}

type soulLightParticle struct {
	position        modelPoint3
	angles          [4]int
	seed, age, kind int
	radius, speed   float64
}

// PrimSoulParticle and PrimPeongUp share the same wandering motion. Keep
// their history so neither frame rate nor camera movement resets a trail.
type soulLights struct {
	tick         int
	particles    []soulLightParticle
	glyphOrigins [2]modelPoint3
}

func soulLinkLightPosition(tick int, right modelPoint3) modelPoint3 {
	angle := min(tick*(tick+1)/60, 359)
	p := mul3(right, (-30+float64(angle)/6)/5)
	p.y = (40 + 20*math.Sin(degreesToRadians(float64(angle)))) / 5
	return p
}

func (p *soulLights) advance(ctx client.Context, effect worldEffect, right modelPoint3, now time.Time) {
	end := min(int(now.Sub(effect.starts)/db.EffectFrameDuration)+1, 300)
	dodge := effect.effectID == effectStormKick4
	for p.tick < end {
		p.tick++
		if !dodge && (p.tick == 2 || p.tick == 22) {
			at := effect.starts.Add(time.Duration(p.tick-1) * db.EffectFrameDuration)
			p.glyphOrigins[(p.tick-2)/20] = effectWorldAnchor(ctx, effect, at)
		}
		count := 0
		if dodge && p.tick == 1 {
			count = 120
		}
		if !dodge && p.tick < 180 {
			count = 6
		}
		for i := 0; i < count; i++ {
			seed := (p.tick*120 + i) * 16
			particle := soulLightParticle{seed: seed, kind: i % 3, speed: .07 / 5,
				position: soulLinkLightPosition(p.tick, right),
				radius:   (1 + float64(hitEffectRandom(effect, (p.tick*2+i/3)*16, 11))*.1) / 5}
			for j := range particle.angles {
				particle.angles[j] = hitEffectRandom(effect, seed+j, 360)
			}
			if dodge {
				angle := deterministicAngle(effect, seed+4)
				particle.position = modelPoint3{x: .7 * math.Cos(angle), z: .7 * math.Sin(angle), y: float64(hitEffectRandom(effect, seed+5, 8)) / 5}
				particle.age = -hitEffectRandom(effect, seed+6, 26)
				particle.radius = (2 + float64(hitEffectRandom(effect, seed+7, 11))*.05) / 5
				particle.speed = (.08 + float64(hitEffectRandom(effect, seed+8, 11))*.005) / 5
			}
			p.particles = append(p.particles, particle)
		}
		live := p.particles[:0]
		for _, particle := range p.particles {
			particle.age++
			if particle.age > 0 {
				if (dodge && particle.age >= 102) || (!dodge && particle.age >= 90) {
					continue
				}
				for axis := 0; axis < 4; axis += 2 {
					a, target := &particle.angles[axis], &particle.angles[axis+1]
					if *target > *a {
						*a = min(*a+3, 359)
						if *a > *target {
							*target = hitEffectRandom(effect, particle.seed+p.tick*4+axis, max(*a, 1))
						}
					} else {
						*a = max(*a-2, 0)
						if *a <= *target {
							*target = 359 - hitEffectRandom(effect, particle.seed+p.tick*4+axis, max(*a, 1))
						}
					}
				}
				x, z, y := particle.speed, particle.speed, particle.speed
				if dodge {
					x, z = .03/5, .03/5
				} else {
					switch particle.kind {
					case 1:
						x, z, y = -x, -z, -y
					case 2:
						x, z, y = x*1.2, -z*1.2, 0
					}
				}
				if particle.position.y > 0 {
					particle.position.x += x * math.Sin(degreesToRadians(float64(particle.angles[0])))
					particle.position.z += z * math.Sin(degreesToRadians(float64(particle.angles[2])))
				}
				particle.position.y += y
			}
			live = append(live, particle)
		}
		p.particles = live
	}
}

func (m *WorldMode) drawSoulLights(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	if now.Before(effect.starts) || effect.soulLights == nil {
		return
	}
	center := effectWorldAnchor(ctx, effect, now)
	if effect.effectID == effectStormKick4 {
		center = effect.actorOrigin
	}
	right, up, _, ok := projection.BillboardBasis(center.x, center.z, center.y)
	if !ok {
		return
	}
	effect.soulLights.advance(ctx, effect, right, now)
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for _, particle := range effect.soulLights.particles {
		if particle.age <= 0 {
			continue
		}
		a := max(20*min(particle.age, 9)-3*max(particle.age-30, 0), 0)
		c := color.RGBA{80, 80, 255, uint8(a)}
		if effect.effectID == effectStormKick4 {
			a = max(5*min(particle.age, 29)-2*max(particle.age-29, 0), 0)
			c = color.RGBA{105, 105, 225, uint8(a)}
		}
		half := particle.radius / math.Sqrt2
		position := add3(center, particle.position)
		if a == 0 || particle.position.y < half*up.y {
			continue
		}
		r, u := mul3(right, half), mul3(up, half)
		appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{
			add3(sub3(position, r), u), add3(add3(position, r), u), sub3(add3(position, r), u), sub3(sub3(position, r), u),
		}, c)
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
	}
}

func soulLinkGlyph(age, layer int) (radius, alpha float64) {
	if age <= 0 {
		return 0, 0
	}
	a := min(13*min(age, 19), 10+layer*23)
	if layer <= 8 {
		a -= 2 * max(age-19, 0)
	} else {
		a -= 5 * max(age-100, 0)
	}
	growth := min(max(age-100, 0), 43) + min(max(age-100, 0), 47)
	return (5*math.Pow(1.005, float64(growth)) + float64(10-layer)) / 5, float64(max(a, 0)) / 255
}

func (m *WorldMode) drawSoulLinkEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	if now.Before(effect.starts) || effect.soulLights == nil || len(component.textureFiles) != 8 {
		return
	}
	m.drawSoulLights(screen, ctx, projection, component, effect, now)
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	center := effectWorldAnchor(ctx, effect, now)
	right, _, _, ok := projection.BillboardBasis(center.x, center.z, center.y)
	if !ok {
		return
	}
	for i, file := range component.textureFiles {
		age := frame - i*20
		texture := m.effectFileTexture(ctx.Resources, file)
		position := add3(effect.soulLights.glyphOrigins[i/4], mul3(right, float64(-22+i*6+(i/4)*2)/5))
		position.y += 8
		for layer := 10; layer >= 1; layer-- {
			radius, alpha := soulLinkGlyph(age, layer)
			if alpha <= 0 {
				continue
			}
			size := math.Sqrt2 * radius
			drawTexturedEffectBillboardRotatedXY(screen, projection, texture, position.x, position.z, position.y, size, size, 0, color.RGBA{255, 255, 255, uint8(alpha * 255)}, false)
		}
	}
	tick := frame + 1
	a := max(20*min(tick, 10)-5*max(tick-170, 0), 0)
	if a > 0 {
		p := add3(center, soulLinkLightPosition(tick, right))
		drawTexturedEffectBillboardRotatedXY(screen, projection, m.effectFileTexture(ctx.Resources, component.textureFile), p.x, p.z, p.y, math.Sqrt2/2, math.Sqrt2/2, 0, color.RGBA{125, 125, 255, uint8(a)}, false)
	}
}

func (m *WorldMode) drawKaizelEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	// EF_KAIZEL launches JobLvUp50_2(0) and (45) at tick 15.
	age := int(now.Sub(effect.starts)/db.EffectFrameDuration) - 15 + 1
	alpha := 5*min(age, 7) - 2*max(age-9, 0)
	if now.Before(effect.starts) || age <= 0 || alpha <= 0 {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	inner, outer := float64(1+age)/5, float64(3+3*age)/5
	height := (1.5 + .4*float64(min(age, 7))) / 5
	var vertices []render.Vertex3D
	var indices []uint16
	for blade := 0; blade < 8; blade++ {
		for slice := 0; slice < 3; slice++ {
			sin, cos := math.Sincos(degreesToRadians(float64(blade*45 - 1 + slice)))
			low, high, a := height*.1, height*.4, alpha/3
			if slice == 1 {
				low, high, a = height*.2, height, alpha
			}
			appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{
				{x: x + cos*inner, y: z + low, z: y + sin*inner},
				{x: x + cos*outer, y: z + high, z: y + sin*outer},
				{x: x + cos*outer, y: z, z: y + sin*outer},
				{x: x + cos*inner, y: z, z: y + sin*inner},
			}, color.RGBA{255, 255, 255, uint8(a)})
		}
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
}
