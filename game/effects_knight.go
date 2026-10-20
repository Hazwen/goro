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

func knightSplashAction(action network.ActorActionNotify) bool {
	if action.Action != 5 { // ACTION_SPLASH: secondary victims do not replay the skill.
		return false
	}
	switch action.SkillID {
	case db.SkillKNPierce, db.SkillKNBrandishspear, db.SkillKNSpearstab, db.SkillKNSpearboomerang, db.SkillKNBowlingbash, db.SkillKNChargeatk:
		return true
	default:
		return false
	}
}

func (m *WorldMode) drawBowlingBashEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	origin := effect.actorOrigin
	if texture := m.effectFileTexture(ctx.Resources, component.textureFile); texture != nil {
		for _, birth := range []int{0, 5} {
			age, life := frame-birth, 20-birth
			if age < 0 || age >= life {
				continue
			}
			n := float64(age + 1)
			growth := (.5*n - .03*n*(n+1)/2) / 5
			alpha := float64(240-7*birth) * math.Min(1, float64(life-age-1)/float64(life/2))
			yaw := -45*float64(normalizeDirectionIndex(effect.actorDirection)) + float64(20*birth)
			right := rotateEffectCylinderVector(modelPoint3{x: 1}, 0, yaw, 0)
			depth := rotateEffectCylinderVector(modelPoint3{z: 1}, 0, yaw, 0)
			drawWorldCylinderBandWithOptions(screen, m.whitePixel, texture, origin.x, origin.z, origin.y,
				.6+growth, 1.6+growth, .7, color.RGBA{255, 255, 255, uint8(alpha)},
				10, right, depth, modelPoint3{y: 1}, 2.5, render.BlendSourceOver)
		}
	}
	if frame < 0 || frame >= 50 {
		return
	}
	if texture := m.effectFileTexture(ctx.Resources, component.textureFiles[0]); texture != nil {
		n := float64(min(frame+1, 20))
		radius := (8 + .7*n - (.7/30)*n*(n+1)/2 + .08*float64(max(0, frame+1-20))) / 5
		alpha := uint8(45 * math.Min(1, float64(49-frame)/15))
		vertices, indices := groundEffectRingMesh(texture, origin, radius-1, radius, alpha)
		screen.DrawTriangles3DOwned(vertices, indices, texture, &render.DrawTrianglesOptions{Filter: render.FilterLinear, Address: render.AddressRepeat})
	}
}

func initSpearBoomerang(ctx client.Context, effect *worldEffect) {
	destination := effectWorldAnchor(ctx, *effect, effect.starts)
	origin := destination
	if _, ok, _ := actorForCombatID(ctx, effect.targetID); effect.targetID != 0 && ok {
		origin = effectWorldAnchor(ctx, worldEffect{actorID: effect.targetID}, effect.starts)
		effect.actorID, effect.targetID = effect.targetID, effect.actorID
		effect.x, effect.y, _ = effectAnchor(ctx, effect.actorID)
	}
	effect.projectileDirection = sub3(destination, origin)
	effect.projectileDirection.y = 0
	origin.y += 3 // Fifteen native units above the caster.
	effect.actorOrigin = origin
	effect.hasActorTransform = true
}

func spearBoomerangPosition(effect worldEffect, age int) (modelPoint3, bool) {
	if age < 0 || age > 15 {
		return modelPoint3{}, false
	}
	delta := effect.projectileDirection
	fraction := float64(age+1) / 15
	// Prim3DParticle stops when it comes within three native units of the target.
	if math.Hypot(delta.x, delta.z)*(1-fraction) <= .6 {
		return modelPoint3{}, false
	}
	return add3(effect.actorOrigin, mul3(delta, fraction)), true
}

func (m *WorldMode) drawSpearBoomerangEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	view := m.effectSpriteView(ctx.Resources, component.spriteFile)
	if view == nil || len(view.act.Actions) == 0 || len(view.act.Actions[0].Animations) == 0 {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	delta := effect.projectileDirection
	angle := math.Atan2(delta.z, delta.x) - math.Pi/2 + degreesToRadians(projection.cameraYaw)
	options := spriteBillboardTriangleDrawOptions()
	for i, alpha := range []uint8{254, 180, 130, 80} {
		age := frame - i*4
		position, visible := spearBoomerangPosition(effect, age)
		if !visible {
			continue
		}
		motion := (age + 1) % len(view.act.Actions[0].Animations)
		key := singleSpriteBillboardKey{motion: motion}
		billboard, ok := view.billboards[key]
		if !ok {
			billboard, ok = composeSingleSpriteBillboard(view, view.act.Actions[0].Animations[motion])
			if !ok {
				continue
			}
			view.billboards[key] = billboard
		}
		drawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard, position.x, position.z, position.y,
			effectPixelRatio, angle, float64(alpha)/255, 1, color.RGBA{255, 255, 255, 255}, options)
	}
}
