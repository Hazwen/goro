package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func initDirectedEffect(ctx client.Context, effect *worldEffect) {
	destination := effectWorldAnchor(ctx, *effect, effect.starts)
	origin := destination
	if _, ok, _ := actorForCombatID(ctx, effect.targetID); effect.targetID != 0 && ok {
		origin = effectWorldAnchor(ctx, worldEffect{actorID: effect.targetID}, effect.starts)
		// The skill mapping passes victim, caster; these effects belong to the caster.
		effect.actorID, effect.targetID = effect.targetID, effect.actorID
		effect.x, effect.y, _ = effectAnchor(ctx, effect.actorID)
	}
	direction := sub3(destination, origin)
	if effect.effectID == effectGrimtooth {
		direction.y = 0
		// The original stops emitting immediately for a target within 2.5 native units.
		if distance := math.Hypot(direction.x, direction.z); distance > .5 {
			direction = mul3(direction, .4/distance)
		} else {
			direction = modelPoint3{}
		}
	} else {
		// ThrowItem keeps the entire displacement to calculate its flight time.
		origin.y += 1.6
	}
	effect.actorOrigin = origin
	effect.projectileDirection = direction
	effect.hasActorTransform = true
}

func throwItemAt(effect worldEffect, process int) (position modelPoint3, angle float64, alpha int) {
	if process <= 0 {
		return modelPoint3{}, 0, 0
	}
	delta := effect.projectileDirection
	distance := math.Hypot(delta.x, delta.z)
	speed, height, rotation, spin := .6, .2, 330, 45
	if effect.effectID == effectThrowItem3 {
		speed, height, rotation, spin = .4, 2, 0, 10
	}
	travel := int(distance/speed) + 1
	if isAlchemistBottle(effect.effectID) {
		speed, height, rotation, spin = distance/25, 6, 0, 10
		travel = 26
	}
	stop := int(float64(travel) * 1.05)
	n := min(process, stop)
	position = effect.actorOrigin
	if distance > 0 {
		position = add3(position, mul3(delta, speed*float64(n)/distance))
	}
	phase := min(180, int(180*float64(n)/float64(travel)))
	position.y += math.Sin(degreesToRadians(float64(phase))) * height
	return position, degreesToRadians(float64((rotation + process*spin) % 360)), max(0, 255-20*max(0, process-stop))
}

func (m *WorldMode) drawThrowItemEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTextureWithBlackKey(ctx.Resources, component.textureFile, !isAlchemistBottle(effect.effectID))
	if texture == nil {
		return
	}
	process := int((now.Sub(effect.starts)-component.delay)/db.EffectFrameDuration) + 1
	if now.Before(effect.starts.Add(component.delay)) {
		return
	}
	if effect.effectID == effectThrowItem4 {
		effect.itemLevel = 4
	}
	for trail := 0; trail < 4; trail++ {
		step := process
		if trail > 0 {
			// PrimThrowItem shifts the three afterimages before every odd update.
			step = (process-1)/2*2 - (trail-1)*2
		}
		position, angle, alpha := throwItemAt(effect, step)
		if trail > 0 {
			alpha = max(0, alpha-150-(trail-1)*25)
		}
		if alpha == 0 {
			continue
		}
		size := .8 * math.Sqrt2
		width, additive := size, false
		tint := color.RGBA{255, 200, 255, uint8(alpha)}
		if effect.effectID == effectThrowItem3 {
			size = .4 * math.Sqrt2
			width = size
			tint.G = 255
		}
		if isAlchemistBottle(effect.effectID) {
			size = .6 * math.Sqrt2
			width = -size // RenderCloud reverses the bottle's texture corners.
			tint = color.RGBA{135, 255, 135, uint8(alpha)}
			if effect.effectID != effectThrowItem {
				colors := [...]color.RGBA{{255, 135, 135, 255}, {255, 205, 135, 255}, {255, 255, 135, 255}, {255, 255, 255, 255}, {135, 135, 255, 255}, {255, 255, 255, 255}}
				if effect.itemLevel < 1 || effect.itemLevel > len(colors) {
					continue
				}
				tint = colors[effect.itemLevel-1]
				tint.A = uint8(alpha)
				additive = effect.itemLevel == 6
			}
		}
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture,
			position.x, position.z, position.y, width, size, angle-3*math.Pi/4,
			tint, additive)
	}
}

// Shield Boomerang and Monk Tanji share the native spinning-projectile primitive.
type shieldImage struct {
	position modelPoint3
	angle    float64
	alpha    int
}

type shieldFlight struct {
	images         [4]shieldImage
	tick, phase    int
	heading, slope float64
	distance       float64
	oneWay         bool
	deflection     float64
}

func initShieldBoomerang(ctx client.Context, effect *worldEffect) {
	destination := effectWorldAnchor(ctx, *effect, effect.starts)
	if _, ok, _ := actorForCombatID(ctx, effect.targetID); effect.targetID != 0 && ok {
		effect.actorID, effect.targetID = effect.targetID, effect.actorID
		effect.x, effect.y, _ = effectAnchor(ctx, effect.actorID)
	}
	origin := effectWorldAnchor(ctx, *effect, effect.starts)
	delta := sub3(destination, origin)
	distance := math.Hypot(delta.x, delta.z)
	origin.y += 1.6
	effect.shield = &shieldFlight{
		images:  [4]shieldImage{{position: origin, alpha: 255, angle: -3 * math.Pi / 4}},
		heading: math.Atan2(delta.z, delta.x), distance: distance,
	}
	if distance > 0 {
		effect.shield.slope = .4 * delta.y / distance
	}
	// TANJI uses the same native primitive, with a delayed, one-way flight.
	if effect.effectID == effectTanji {
		effect.shield.tick = -15
		effect.shield.oneWay = true
		effect.shield.deflection = degreesToRadians(float64(hitEffectRandom(*effect, 0, 45)))
		if hitEffectRandom(*effect, 1, 2) == 0 {
			effect.shield.deflection *= -1
		}
		effect.shield.images[0].alpha = 5
		effect.shield.images[0].angle = math.Pi / 4
	}
}

func (flight *shieldFlight) advance(tick int, caster modelPoint3) {
	for flight.tick < min(tick, 499) {
		flight.tick++
		if flight.tick <= 0 {
			continue
		}
		current := &flight.images[0]
		if flight.oneWay && flight.tick <= 10 {
			current.alpha = min(255, current.alpha+25)
		}
		if flight.oneWay || flight.tick%3 == 0 {
			flight.images[3] = flight.images[2]
			flight.images[3].alpha = max(0, flight.images[3].alpha-25)
			flight.images[2] = flight.images[1]
			flight.images[2].alpha = max(0, flight.images[2].alpha-25)
			flight.images[1] = *current
			flight.images[1].alpha = max(0, current.alpha-150)
		}
		current.angle -= math.Pi / 12
		switch flight.phase {
		case 0:
			if .4*float64(flight.tick) > flight.distance {
				flight.phase = 1
				if flight.oneWay {
					flight.phase = 2
					flight.heading += flight.deflection
				}
			}
			current.position.y += flight.slope
		case 1:
			want := math.Atan2(caster.z-current.position.z, caster.x-current.position.x)
			want = math.Mod(want+2*math.Pi, 2*math.Pi)
			flight.heading = math.Mod(flight.heading+2*math.Pi, 2*math.Pi)
			if math.Abs(want-flight.heading) > degreesToRadians(4) {
				flight.heading = math.Mod(flight.heading+degreesToRadians(5), 2*math.Pi)
			} else {
				flight.phase = 2
			}
		case 2:
			current.alpha = max(0, current.alpha-12)
			current.position.y -= flight.slope
		}
		speed := .4
		if flight.oneWay && flight.phase == 2 {
			speed *= 3
		}
		current.position.x += speed * math.Cos(flight.heading)
		current.position.z += speed * math.Sin(flight.heading)
	}
}

func drawShieldImage(screen *render.Frame, projection sceneProjection, texture *render.Image, particle shieldImage, size float64, additive bool) {
	if particle.alpha <= 0 {
		return
	}
	p := particle.position
	drawTexturedEffectBillboardRotatedXY(screen, projection, texture, p.x, p.z, p.y,
		size, size, particle.angle, color.RGBA{255, 255, 255, uint8(min(255, particle.alpha))}, additive)
}

func (m *WorldMode) drawShieldBoomerangEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil || effect.shield == nil {
		return
	}
	tick := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	size := 7 * math.Sqrt2 / 5
	if effect.shield.oneWay {
		tick -= 15
		size = effect.size
		if size == 0 {
			size = 2 * math.Sqrt2 / 5
		}
	}
	effect.shield.advance(tick, effectWorldAnchor(ctx, effect, now))
	for _, particle := range effect.shield.images {
		drawShieldImage(screen, projection, texture, particle, size, effect.shield.oneWay)
	}
}

func (m *WorldMode) drawTaekwonDash(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	age := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	alpha := 12*min(age, 10) - 4*max(0, age-20)
	piercing := effect.effectID == effectChemical4
	if piercing {
		alpha = 20*min(age, 10) - 2*max(0, age-20)
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if alpha <= 0 || texture == nil {
		return
	}
	angle := math.Atan2(effect.projectileDirection.z, effect.projectileDirection.x)
	direction := modelPoint3{x: math.Cos(angle), z: math.Sin(angle)}
	side := modelPoint3{x: math.Sin(angle), z: -math.Cos(angle)}
	var vertices []render.Vertex3D
	var indices []uint16
	for i := 0; i < 32; i++ {
		seed := i * 6
		offset := float64(15 + hitEffectRandom(effect, seed, 11))
		if i/4%2 == 0 {
			offset = -offset
		}
		width := .3 + .1*float64(hitEffectRandom(effect, seed+1, 4))
		length := float64(20 + hitEffectRandom(effect, seed+2, 31))
		speed := 1.5 + .1*float64(hitEffectRandom(effect, seed+3, 21))
		travel := float64(hitEffectRandom(effect, seed+4, 200)) + float64(age)*speed
		if piercing {
			width += .1
			length += 10
			travel += .025 * float64(age*(age+1))
		}
		travel = math.Mod(travel, 200) - 100
		point := func(along, across float64) modelPoint3 {
			p := add3(effect.actorOrigin, add3(mul3(direction, along/5), mul3(side, across/5)))
			p.y += 1.2
			return p
		}
		shade := uint8(225 - 20*hitEffectRandom(effect, seed+5, 3))
		tint := color.RGBA{shade, shade, 255, uint8(alpha)}
		if piercing {
			tint.G, tint.B = 255, shade
		}
		appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{
			point(travel+length, offset+width), point(travel, offset+width),
			point(travel, offset-width), point(travel+length, offset-width),
		}, tint)
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(false, false))
}
