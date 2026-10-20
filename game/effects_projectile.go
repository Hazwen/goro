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
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	process := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
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
