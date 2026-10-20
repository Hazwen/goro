package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
)

const arrowShowerReleaseDelay = 18 * 24 * time.Millisecond

func (m *WorldMode) addArrowShower(ctx client.Context, sourceID uint32, x, y float64, ground bool, now time.Time) {
	if !m.addWorldEffectAt(ctx, effectArrowShower, sourceID, now.Add(arrowShowerReleaseDelay)) {
		return
	}
	effect := &m.worldEffects[len(m.worldEffects)-1]
	effect.actorOrigin = effectWorldAnchor(ctx, *effect, now)
	effect.hasActorTransform = true
	effect.arrowTargets = new([9]modelPoint3)
	// LaunchSkillSpecialEffect aims at the victim and its eight neighbouring
	// cells. ZC_NOTIFY_GROUNDSKILL instead aims the first arrow at the caster.
	for i, offset := range [9][2]int{{0, 0}, {1, 0}, {1, 1}, {1, -1}, {-1, 0}, {-1, 1}, {-1, -1}, {0, 1}, {0, -1}} {
		tx, ty := x, y
		if i > 0 {
			tx, ty = math.Floor(cellCenter(x))+float64(offset[0]), math.Floor(cellCenter(y))+float64(offset[1])
		}
		target := modelPoint3{x: cellCenter(tx), y: terrainHeightAt(ctx.World, tx, ty) + .07, z: cellCenter(ty)}
		if i == 0 && ground {
			target = effect.actorOrigin
		}
		// CArrowEffect's positional constructor and first update each raise
		// the endpoint by ten native units; the source is raised only once.
		target.y += 4
		effect.arrowTargets[i] = target
	}
	effect.actorOrigin.y += 2
}

func arrowShowerPosition(origin, target modelPoint3, elapsed time.Duration) (modelPoint3, bool) {
	delta := sub3(target, origin)
	distance := math.Sqrt(dot3(delta, delta))
	flight := float64(referenceBowArrowFlightDuration) * math.Min(1, distance/8)
	if elapsed < 0 || flight <= 0 || float64(elapsed) >= flight {
		return modelPoint3{}, false
	}
	return add3(origin, mul3(delta, float64(elapsed)/flight)), true
}

func (m *WorldMode) drawArrowShower(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	if effect.arrowTargets == nil || now.Before(effect.starts) {
		return
	}
	view := m.effectSpriteView(ctx.Resources, component.spriteFile)
	if view == nil || len(view.act.Actions) == 0 || len(view.act.Actions[0].Animations) == 0 {
		return
	}
	key := singleSpriteBillboardKey{ignoreLayerAngles: true}
	billboard, ok := view.billboards[key]
	if !ok {
		billboard, ok = composeSingleSpriteBillboardWithOptions(view, view.act.Actions[0].Animations[0], true)
		if !ok {
			return
		}
		view.billboards[key] = billboard
	}
	for _, target := range effect.arrowTargets {
		position, visible := arrowShowerPosition(effect.actorOrigin, target, now.Sub(effect.starts))
		if !visible {
			continue
		}
		delta := sub3(target, effect.actorOrigin)
		angle := math.Atan2(delta.z, delta.x) + math.Pi/2 - degreesToRadians(projection.cameraYaw)
		drawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard, position.x, position.z, position.y,
			effectPixelRatio, angle, 1, 1, color.RGBA{255, 255, 255, 255}, spriteBillboardTriangleDrawOptions())
	}
}
