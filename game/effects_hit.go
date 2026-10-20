package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) drawHitRingEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	distance, alpha := hitRingAnimation(component, now.Sub(effect.starts)-component.delay)
	if alpha <= 0 {
		return
	}
	texture := m.effectTexture(ctx.Resources, component.textureName)
	if texture == nil {
		return
	}
	if effect.hasActorTransform {
		x, y, z = effect.actorOrigin.x, effect.actorOrigin.z, effect.actorOrigin.y
	}
	// Hit1 captures 180 - master.roty; DirToDeg(dir) is 180 + dir*45.
	// Keep this orientation even if the target turns during the impact.
	yaw := -45 * float64(normalizeDirectionIndex(effect.actorDirection))
	right := rotateEffectCylinderVector(modelPoint3{x: 1}, component.angleX, yaw, 0)
	depth := rotateEffectCylinderVector(modelPoint3{z: 1}, component.angleX, yaw, 0)
	axis := rotateEffectCylinderVector(modelPoint3{y: 1}, component.angleX, yaw, 0)
	n := float64((now.Sub(effect.starts)-component.delay)/component.frameDelay + 1)
	motion := component.hitRing
	height := component.height + n*motion.HeightSpeed + motion.HeightAcceleration*n*(n+1)/2
	if motion.RandomImpactOffset {
		// SonicBlowHit divides integer random offsets before converting to world units.
		x += float64((1+hitEffectRandom(effect, 601, 10))/4) / 5
		z -= float64((1+hitEffectRandom(effect, 602, 20))/2) / 5
	}
	x += axis.x * distance
	y += axis.z * distance
	z += component.posZ + axis.y*distance
	// Prim3DCylinder advances U by 1/4 per face and uses RF_ALPHA.
	drawWorldCylinderBandWithOptions(screen, m.whitePixel, texture, x, y, z,
		component.bottomSize, component.topSize, height,
		effectComponentTint(component, alpha), component.circleSides, right, depth, axis,
		float64(component.circleSides)/4, render.BlendSourceOver)
}

func hitRingAnimation(component worldEffectComponent, elapsed time.Duration) (distance, alpha float64) {
	if elapsed < 0 || elapsed >= component.duration || component.frameDelay <= 0 || component.hitRing == nil {
		return 0, 0
	}
	frames := float64(component.duration / component.frameDelay)
	if frames < 2 {
		return 0, 0
	}
	// Sakexe Prim3DCylinder: acceleration is -speed/(2*duration).
	// Acceleration and movement run before the first draw. One native unit is .2 cells.
	n := float64(elapsed/component.frameDelay + 1)
	speed := component.hitRing.Speed
	acceleration := -speed / (2 * frames)
	distance = component.hitRing.InitialDistance + n*speed + acceleration*n*(n+1)/2
	// ProcessAlpha applies its first decrement on FadeOutFrame, before drawing.
	alpha = component.alphaMax * clampFloat((frames-n)/math.Max(1, frames-float64(component.hitRing.FadeOutFrame)), 0, 1)
	return distance, alpha
}

func hitEffectRandom(effect worldEffect, salt, count int) int {
	return min(count-1, int(deterministicUnit(effect, salt)*float64(count)))
}

type hitParticle struct {
	axis                modelPoint3
	radius, speed, size float64
	frames              int
}

func newHitParticle(effect worldEffect, component worldEffectComponent, index int) hitParticle {
	rand := func(salt, count int) int { return hitEffectRandom(effect, index*17+salt, count) }
	yaw := degreesToRadians(float64(-45*normalizeDirectionIndex(effect.actorDirection) - 40 + rand(1, 80)))
	pitch := degreesToRadians(float64(-50 + rand(2, 100)))
	// The original multiplies Y then X, then we flip its downward vertical axis.
	return hitParticle{
		axis:   modelPoint3{x: math.Sin(yaw), y: math.Cos(yaw) * math.Sin(pitch), z: math.Cos(yaw) * math.Cos(pitch)},
		radius: float64(4+rand(3, 4)) / 5,
		speed:  float64(component.hitParticles.SpeedMin+rand(4, component.hitParticles.SpeedRange)) / 500,
		size:   float64(60+rand(5, 100)) / 100,
		frames: 6 + rand(6, 24),
	}
}

func (p hitParticle) segment(frame, trail int, height float64) (position modelPoint3, size, alpha float64) {
	n := float64(frame + 1)
	life := clampFloat(1-n/float64(p.frames), 0, 1)
	size = p.size * life * (1 - float64(trail)/6)
	alpha = (254.0 / 255) * life * (1 - float64(trail)/3)
	// ShiftSegment stores two earlier positions, but fades/scales all three
	// using the current tick. Initial history is at the actor's feet.
	step := math.Max(0, n-float64(trail))
	if step > 0 {
		distance := p.radius + step*p.speed - p.speed/(2*float64(p.frames))*step*(step+1)/2
		position = add3(modelPoint3{y: height}, mul3(p.axis, distance))
	}
	return position, size, alpha
}

func (m *WorldMode) drawHitParticlesEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, componentIndex int, x, y, z float64, now time.Time) {
	elapsed := now.Sub(effect.starts) - component.delay
	if elapsed < 0 || elapsed >= component.duration || component.frameDelay <= 0 || component.hitParticles == nil {
		return
	}
	view := m.effectSpriteView(ctx.Resources, component.spriteFile)
	if view == nil || len(view.act.Actions) == 0 || len(view.act.Actions[0].Animations) == 0 {
		return
	}
	frame := int(elapsed / component.frameDelay)
	// AnimMotion advances immediately, then every four primitive ticks.
	motion := (frame/4 + 1) % len(view.act.Actions[0].Animations)
	key := singleSpriteBillboardKey{motion: motion}
	billboard, ok := view.billboards[key]
	if !ok {
		billboard, ok = composeSingleSpriteBillboard(view, view.act.Actions[0].Animations[motion])
		if !ok {
			return
		}
		view.billboards[key] = billboard
	}
	options := triangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
	// Hit3/Hit4 keep their CRagEffect origin at the impact point. The primitive
	// trails record earlier particle positions, independent of later actor movement.
	if effect.hasActorTransform {
		x, y, z = effect.actorOrigin.x, effect.actorOrigin.z, effect.actorOrigin.y
	}
	for i := 0; i < component.duplicate; i++ {
		particle := newHitParticle(effect, component, componentIndex*101+i)
		for trail := 0; trail < 3; trail++ {
			position, size, alpha := particle.segment(frame, trail, component.posZ)
			if alpha <= 0 || size <= 0 {
				continue
			}
			drawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard,
				x+position.x, y+position.z, z+position.y, size*effectPixelRatio, 0, alpha, 1,
				effectComponentTint(component, 1), options)
		}
	}
}

func initImpactParticles(ctx client.Context, effect *worldEffect) {
	destination := effectWorldAnchor(ctx, *effect, effect.starts)
	origin := destination
	if _, ok, _ := actorForCombatID(ctx, effect.targetID); effect.targetID != 0 && ok {
		origin = effectWorldAnchor(ctx, worldEffect{actorID: effect.targetID}, effect.starts)
		effect.actorID, effect.targetID = effect.targetID, effect.actorID
		effect.x, effect.y, _ = effectAnchor(ctx, effect.actorID)
	}
	effect.actorOrigin = destination
	effect.actorOrigin.y += 1.6
	effect.projectileDirection = sub3(destination, origin)
	effect.hasActorTransform = true
}

func impactParticle(effect worldEffect, index, frame int) (position modelPoint3, size, angle float64, alpha int) {
	salt := index * 5
	age := frame + 1 - 40 + hitEffectRandom(effect, salt, 10)
	alpha = max(0, 15*clampInt(age-1, 0, 9)-3*max(0, age-10))
	if alpha == 0 {
		return
	}
	direction := math.Atan2(effect.projectileDirection.z, effect.projectileDirection.x)
	direction += degreesToRadians(float64(hitEffectRandom(effect, salt+1, 60) - 30))
	speed := (.75 + .01*float64(hitEffectRandom(effect, salt+2, 50))) / 5
	position = add3(effect.actorOrigin, modelPoint3{
		x: math.Cos(direction) * speed * float64(age),
		z: math.Sin(direction) * speed * float64(age),
		y: (1 - .01*float64(hitEffectRandom(effect, salt+3, 200))) * float64(age) / 5,
	})
	size = (.1 + .05*float64(hitEffectRandom(effect, salt+4, 16))) / 5 * math.Sqrt2
	angle = degreesToRadians(float64(-5*age + 45))
	return
}

func (m *WorldMode) drawImpactParticles(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	for i := 0; i < component.duplicate; i++ {
		position, size, angle, alpha := impactParticle(effect, i, frame)
		if alpha > 0 {
			drawTexturedEffectBillboardRotatedXY(screen, projection, texture,
				position.x, position.z, position.y, size, size, angle, color.RGBA{130, 130, 250, uint8(alpha)}, true)
		}
	}
}
