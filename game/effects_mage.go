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

func (m *WorldMode) addMageProjectileEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) bool {
	count := actionVisualHitCount(action)
	frames := 52
	switch action.SkillID {
	case db.SkillMGFireball, db.SkillMGFrostdiver:
		count = 1
	case db.SkillMGSoulstrike:
		count = min(count, 5)
		frames = 6 + (count-1)*11 + 40
	case db.SkillMGColdbolt, db.SkillMGFirebolt:
		frames = 12 + (count-1)*10 + 70
		if action.SkillID == db.SkillMGFirebolt {
			frames = max(frames, 160)
		}
	default:
		return false
	}
	id := skillBeforeHitEffectIDs(action.SkillID)[0]
	if m.addWorldEffectBetweenAtDuration(ctx, id, action.TargetID, action.SourceID, starts, time.Duration(frames)*db.EffectFrameDuration) {
		effect := &m.worldEffects[len(m.worldEffects)-1]
		effect.hitCount = count
		if action.SkillID == db.SkillMGSoulstrike {
			for i := 1; i < count; i++ {
				m.scheduleSoundAtWorldEffect(starts.Add(time.Duration(6+11*i)*db.EffectFrameDuration), *effect, "effect\\ef_soulstrike.wav")
			}
		}
	}
	return true
}

func initMageProjectile(ctx client.Context, effect *worldEffect) {
	destination := effectWorldAnchor(ctx, *effect, effect.starts)
	if _, ok, _ := actorForCombatID(ctx, effect.targetID); effect.targetID != 0 && ok {
		effect.actorID, effect.targetID = effect.targetID, effect.actorID
		effect.x, effect.y, _ = effectAnchor(ctx, effect.actorID)
	}
	effect.actorOrigin = effectWorldAnchor(ctx, *effect, effect.starts)
	effect.projectileDirection = sub3(destination, effect.actorOrigin)
	effect.projectileDirection.y = 0
	effect.hasActorTransform = true
	if effect.effectID == effectFrostDiver {
		effect.duration = time.Duration(frostDiverTrailCount(*effect)-1+40) * db.EffectFrameDuration
		effect.expires = effect.starts.Add(effect.duration)
	}
}

func frostDiverTrailCount(effect worldEffect) int {
	distance := math.Hypot(effect.projectileDirection.x, effect.projectileDirection.z)
	// One spike per tick, starting five native units ahead, advancing two,
	// and stopping within 2.5 units of the destination.
	return 1 + max(0, int(math.Ceil((distance*5-7.5)/2)))
}

func (m *WorldMode) drawMageProjectile(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	if now.Before(effect.starts) {
		return
	}
	if effect.effectID == effectFrostDiver {
		m.drawFrostDiverSpikes(screen, ctx, component, effect, effect.actorOrigin, now)
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	delta := effect.projectileDirection
	if effect.effectID == effectFireBall {
		angle := math.Atan2(delta.z, delta.x) - math.Pi/2 + degreesToRadians(projection.cameraYaw)
		for i, alpha := range []uint8{255, 180, 130, 80} {
			age := frame - 20 - i*4
			if age < 0 || age >= 20 {
				continue
			}
			fraction := float64(age+1) / 20
			if math.Hypot(delta.x, delta.z)*(1-fraction) <= .6 {
				continue
			}
			p := add3(effect.actorOrigin, mul3(delta, fraction))
			p.y += 3
			tint := color.RGBA{255, 255, 255, alpha}
			if i > 0 {
				tint.R, tint.G, tint.B = 246, 199, 76
			}
			m.drawMageSprite(screen, ctx, projection, component.spriteFile, p, age+1, 1.3, angle, tint)
		}
		return
	}
	count := max(1, effect.hitCount)
	for i := 0; i < count; i++ {
		age := frame - 6 - i*11
		if age < 0 || age >= 40 {
			continue
		}
		roll := 1.0
		if count > 1 {
			roll = -90 + float64((i+1)*180)/float64(count-1)
		}
		for segment := 11; segment >= 0; segment-- {
			p := soulStrikePosition(effect, max(0, age+1-segment), roll)
			scale := 2.75 * (1 - float64(segment)/24)
			alpha := uint8(255 * (1 - float64(segment)/12))
			m.drawMageSprite(screen, ctx, projection, component.spriteFile, p, age/4+1, scale, 0, color.RGBA{255, 255, 255, alpha})
		}
	}
}

func soulStrikePosition(effect worldEffect, tick int, roll float64) modelPoint3 {
	if tick == 0 {
		return effect.actorOrigin
	}
	n := float64(tick)
	direction := normalize3(effect.projectileDirection)
	// Prim3DParticleSpline first retreats, then accelerates forward. Its
	// transverse arc is rotated around the flight axis for each soul.
	forward := sub3(mul3(effect.projectileDirection, n/40), mul3(direction, (3.5*n-.175*n*(n+1)/2)/5))
	arc := (2*n - .1*n*(n-1)/2) / 5
	s, c := math.Sincos(degreesToRadians(roll))
	p := add3(effect.actorOrigin, forward)
	p.y += 2 + arc*c
	p.x -= direction.z * arc * s
	p.z += direction.x * arc * s
	return p
}

func (m *WorldMode) drawMageSprite(screen *render.Frame, ctx client.Context, projection sceneProjection, file string, position modelPoint3, motion int, scale, angle float64, tint color.RGBA) {
	if tint.A == 0 || scale <= 0 {
		return
	}
	view := m.effectSpriteView(ctx.Resources, file)
	if view == nil || len(view.act.Actions) == 0 || len(view.act.Actions[0].Animations) == 0 {
		return
	}
	motion %= len(view.act.Actions[0].Animations)
	key := singleSpriteBillboardKey{motion: motion}
	billboard, ok := view.billboards[key]
	if !ok {
		billboard, ok = composeSingleSpriteBillboard(view, view.act.Actions[0].Animations[motion])
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

func (m *WorldMode) drawFrostDiverSpikes(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, origin modelPoint3, now time.Time) {
	if now.Before(effect.starts) {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	count := 8
	direction := normalize3(effect.projectileDirection)
	trail := effect.effectID == effectFrostDiver
	if trail {
		count = frostDiverTrailCount(effect)
	}
	vertices := make([]render.Vertex3D, 0, 8*12)
	indices := make([]uint16, 0, 8*12)
	first := 0
	if trail {
		first, count = max(0, frame-39), min(count, frame+1)
	}
	for i := first; i < count; i++ {
		age, center := frame, origin
		if trail {
			age -= i
			center = add3(center, mul3(direction, float64(5+2*i)/5))
		}
		if age < 0 || age >= 40 {
			continue
		}
		salt := i * 6
		radius := float64(1+hitEffectRandom(effect, salt, 4)) / 10
		size := float64(100+hitEffectRandom(effect, salt+1, 250)) / 100
		height := float64(200+hitEffectRandom(effect, salt+2, 100)) / 10
		pitch := float64(80 + hitEffectRandom(effect, salt+3, 20))
		alpha := uint8(200)
		if trail {
			radius = float64(5+hitEffectRandom(effect, salt, 10)) / 10
			size = float64(60+hitEffectRandom(effect, salt+1, 40)) / 100
			height = float64(150+hitEffectRandom(effect, salt+2, 30)) / 10
			pitch = float64(75 + hitEffectRandom(effect, salt+3, 30))
			alpha = 255
		}
		placement := degreesToRadians(float64(hitEffectRandom(effect, salt+4, 360)))
		yaw := degreesToRadians(float64(hitEffectRandom(effect, salt+5, 360)))
		center.x += radius * math.Sin(placement) / 5
		center.z -= radius * math.Cos(placement) / 5
		center.y -= 6
		n := float64(min(age+1, 21))
		if age >= 30 {
			alpha = uint8(float64(alpha) * float64(39-age) / 10)
		}
		appendEffectSpike(&vertices, &indices, texture, center, size, height, degreesToRadians(pitch), yaw,
			3*n-.15*n*(n+1)/2, color.RGBA{255, 255, 255, alpha})
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(false, false))
	}
}

func mageBoltFlight(ctx client.Context, effect worldEffect, origin modelPoint3, index int) (start, direction modelPoint3, landing int) {
	start = modelPoint3{x: float64(25+hitEffectRandom(effect, index*4, 10)) / 5, y: 12, z: float64(15+hitEffectRandom(effect, index*4+1, 10)) / 5}
	end := modelPoint3{x: float64(hitEffectRandom(effect, index*4+2, 4)-2) / 5, z: float64(hitEffectRandom(effect, index*4+3, 4)-2) / 5}
	direction = normalize3(sub3(end, start))
	for landing = 0; landing < 70; landing++ {
		p := add3(origin, add3(start, mul3(direction, .55*float64(landing+1))))
		if p.y < terrainHeightAtRenderPoint(ctx.World, p.x, p.z) {
			break
		}
	}
	return
}

func (m *WorldMode) drawMageBolt(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, origin modelPoint3, now time.Time) {
	if now.Before(effect.starts) {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	fire := effect.effectID == effectFireBolt || effect.effectID == effectFireArrow
	count := max(1, effect.hitCount)
	for i := max(0, (frame-82)/10); i < min(count, (frame-12)/10+1); i++ {
		age := frame - 12 - i*10
		if age < 0 || age >= 70 {
			continue
		}
		start, direction, landing := mageBoltFlight(ctx, effect, origin, i)
		center := add3(origin, add3(start, mul3(direction, .55*float64(min(age, landing)+1))))
		alpha := 255.0
		if age > landing {
			alpha *= clampFloat(float64(landing+25-age)/20, 0, 1)
		}
		texture := m.effectFileTexture(ctx.Resources, component.textureFiles[(age+1)%len(component.textureFiles)])
		if texture != nil && alpha > 0 {
			width, length := 3.8/5, 11.5/5
			if fire {
				width, length = 3.5/5, 14.0/5
			}
			along := mul3(direction, length)
			right := mul3(normalize3(cross3(direction, modelPoint3{y: 1})), width)
			up := mul3(normalize3(cross3(direction, right)), width)
			vertices := make([]render.Vertex3D, 0, 8)
			indices := make([]uint16, 0, 12)
			for _, side := range [2]modelPoint3{right, up} {
				appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{
					add3(sub3(center, along), side), add3(add3(center, along), side),
					sub3(add3(center, along), side), sub3(sub3(center, along), side),
				}, color.RGBA{255, 255, 255, uint8(alpha)})
			}
			screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
		}
		ringAge := age - landing - 1
		if ringAge >= 0 && ringAge < 30 {
			texture := m.effectFileTexture(ctx.Resources, component.textureFile)
			if texture == nil {
				continue
			}
			n := float64(ringAge + 1)
			radius := (1.2*n - (2.4/70)*n*(n+1)/2) / 5
			vertices, indices := groundEffectRingMesh(texture, origin, max(0, radius-2), radius, effectParticleAlpha(ringAge, 30, 10, 20, 254))
			screen.DrawTriangles3DOwned(vertices, indices, texture, triangleDrawOptions(render.FilterLinear, render.AddressRepeat))
		}
	}
	if fire {
		m.drawFireBoltEmbers(screen, ctx, projection, component, effect, origin, frame)
	}
}

func (m *WorldMode) drawFireBoltEmbers(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, origin modelPoint3, frame int) {
	for birth := max(4, (frame-28)/4*4); birth <= min(frame, 156); birth += 4 {
		age, salt := frame-birth, 1000+birth*5
		life := 6 + hitEffectRandom(effect, salt, 24)
		if age < 0 || age >= life {
			continue
		}
		position, pitch, yaw := origin, 0.0, 0.0
		if birth > 12 {
			index := min((birth-13)/10, max(1, effect.hitCount)-1)
			start, direction, landing := mageBoltFlight(ctx, effect, origin, index)
			tick := min(birth-12-index*10, landing+1)
			position = add3(origin, add3(start, mul3(direction, .55*float64(tick))))
			pitch, yaw = math.Asin(-direction.y), math.Atan2(-direction.x, -direction.z)
		}
		pitch += degreesToRadians(float64(hitEffectRandom(effect, salt+1, 20) - 10))
		yaw += degreesToRadians(float64(hitEffectRandom(effect, salt+2, 40) - 20))
		speed := float64(60+hitEffectRandom(effect, salt+3, 90)) / 100
		n := float64(age + 1)
		distance := (speed*n - (speed/float64(life)/1.5)*n*(n+1)/2) / 5
		delta := rotateModelPointY(rotateModelPointX(modelPoint3{z: distance}, pitch), yaw)
		delta.y = -delta.y
		position = add3(position, delta)
		scale := float64(20+hitEffectRandom(effect, salt+4, 30)) / 100 * (1 - n/float64(life))
		alpha := uint8(254 * max(0, 1-n/float64(min(life, 160-birth))))
		m.drawMageSprite(screen, ctx, projection, component.spriteFile, position, age/4+1, scale, 0, color.RGBA{255, 255, 255, alpha})
	}
}

func (m *WorldMode) drawSightEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, origin modelPoint3, now time.Time) {
	if now.Before(effect.starts) {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	for birth := max(0, (frame-19+1)/2*2); birth <= frame; birth += 2 {
		age := frame - birth
		n := float64(age + 1)
		angle := degreesToRadians(float64(birth) * -5)
		p := add3(origin, modelPoint3{x: 3 * math.Sin(angle), z: -3 * math.Cos(angle)})
		size, alpha, motion := 2.5-.1*n, 150-3*n, age/3+1
		if effect.effectID == effectSightBlaster {
			alpha = max(0, 50-3*n)
		}
		p.y += 4
		if component.shadowTexture {
			size, alpha, motion = 1.5-.06*n, 120-n, age/2+1
			p.y = terrainHeightAtRenderPoint(ctx.World, p.x, p.z) + actorShadowTerrainLift
		}
		m.drawMageSprite(screen, ctx, projection, component.spriteFile, p, motion, size, 0,
			color.RGBA{255, 255, 255, uint8(alpha)})
	}
}
