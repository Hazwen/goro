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

func (m *WorldMode) applyClownSkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	source, target := notify.SourceID, notify.TargetID
	if isLocalActor(ctx, source) {
		source = localSkillTarget(ctx)
	}
	if isLocalActor(ctx, target) {
		target = localSkillTarget(ctx)
	}
	switch notify.SkillID {
	case db.SkillCGMoonlit:
		if _, ok, _ := actorForCombatID(ctx, source); ok {
			m.setActorStateEffect(ctx, source, effectMoonlit, true)
		}
	case db.SkillCGMarionette:
		if _, ok, _ := actorForCombatID(ctx, source); !ok {
			return
		}
		if _, ok, _ := actorForCombatID(ctx, target); !ok {
			return
		}
		m.removeMarionetteLinks(source)
		if m.addWorldEffectBetweenAt(ctx, effectMarionetteLink, source, target, now) {
			effect := &m.worldEffects[len(m.worldEffects)-1]
			effect.persistent, effect.expires = true, time.Time{}
		}
		m.setActorStateEffect(ctx, source, effectPinkBody, true)
		m.setActorStateEffect(ctx, target, effectPinkBody, true)
	}
}

func (m *WorldMode) removeMarionetteLinks(actorID uint32) {
	active := m.worldEffects[:0]
	for _, effect := range m.worldEffects {
		if effect.effectID != effectMarionetteLink || effect.actorID != actorID && effect.targetID != actorID {
			active = append(active, effect)
		}
	}
	m.worldEffects = active
}

// The initial shot uses ACT 88. Each additional hit restarts the normal
// weapon attack, as AM_WILL_BE_ATTACK does in the original client.
func (m *WorldMode) startArrowVulcanHits(ctx client.Context, action network.ActorActionNotify, hitAt time.Time) {
	if action.SkillID != db.SkillCGArrowvulcan || skillSplashAction(action) || actionVisualHitCount(action) < 2 {
		return
	}
	anim, ok := m.actorAnims[action.SourceID]
	source, sourceOK, _ := actorForCombatID(ctx, action.SourceID)
	if !ok || !sourceOK {
		return
	}
	anim.speed = m.actorActionFrameDelay(ctx, source, anim.actionFamily, anim.duration)
	anim.hasSpeed = anim.speed > 0
	count := min(9, actionVisualHitCount(action))
	next := readyFightAnimation(hitAt.Add(time.Duration(count) * multiHitDelay))
	for hit := count - 1; hit >= 1; hit-- {
		at := hitAt.Add(time.Duration(hit) * multiHitDelay)
		rest := actorAnimation{
			actionFamily: attackActionFamilyForActorWithResources(ctx.Resources, source),
			started:      at.Add(24 * time.Millisecond), duration: multiHitDelay - 24*time.Millisecond,
			play: true, hasPlay: true, speed: 24 * time.Millisecond, hasSpeed: true,
			frameOffset: 1, hasFrameOffset: true, next: next,
		}
		next = &actorAnimation{
			actionFamily: rest.actionFamily, started: at, duration: 24 * time.Millisecond,
			hasPlay: true, hasFixedMotion: true, fixedMotion: 4, next: &rest,
		}
	}
	anim.duration = next.started.Sub(anim.started)
	anim.next = next
	m.setCombatActorAction(ctx, action.SourceID, anim)
}

func arrowVulcanLine(effect worldEffect, index, frame int) (from, to modelPoint3, alpha int, radius float64) {
	salt := index * 7
	age := frame + 1 - index/4 - 30 - hitEffectRandom(effect, salt, 110)
	alpha = max(0, 8*clampInt(age, 0, 5)-4*max(0, age-5))
	angle := math.Atan2(effect.projectileDirection.z, effect.projectileDirection.x)
	s, c := math.Sincos(angle)
	back := .15 * float64(hitEffectRandom(effect, salt+1, 100))
	from = add3(effect.actorOrigin, modelPoint3{
		x: (.08*float64(hitEffectRandom(effect, salt+2, 100)) - 4 - back*c) / 5,
		z: (.08*float64(hitEffectRandom(effect, salt+3, 100)) - 4 - back*s) / 5,
		y: (4 + .06*float64(hitEffectRandom(effect, salt+4, 100))) / 5,
	})
	length := math.Hypot(effect.projectileDirection.x, effect.projectileDirection.z) + .06*float64(hitEffectRandom(effect, salt+5, 100))
	to = add3(from, modelPoint3{x: length * c, z: length * s})
	radius = (.02 + .008*float64(hitEffectRandom(effect, salt+6, 11))) / 5
	return
}

func tarotCardAlpha(frame int) int {
	n := frame + 1
	return max(0, min(220, 10*n)-5*max(0, n-199))
}

func (m *WorldMode) drawClownEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	n := frame + 1
	origin := effectWorldAnchor(ctx, effect, now)
	var vertices []render.Vertex3D
	var indices []uint16
	switch effect.effectID {
	case effectTripleAttack3:
		for layer := 0; layer < 2; layer++ {
			vertices, indices = nil, nil
			for i := 0; i < 36; i++ {
				from, to, alpha, radius := arrowVulcanLine(effect, i, frame)
				if alpha > 0 {
					appendMonkTrail(&vertices, &indices, texture, from, to, radius*float64(layer+1), 5, color.RGBA{255, 89, 182, uint8(alpha)})
				}
			}
			if len(indices) > 0 {
				screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(layer == 1, false))
			}
		}
		return
	case effectMoonlit:
		origin.y += 2.8
		for i := 0; i < 20; i++ {
			salt := i * 4
			halfWidth := (4 + .2*float64(hitEffectRandom(effect, salt, 11))) / 5
			angle := hitEffectRandom(effect, salt+1, 360) + n*(3+hitEffectRandom(effect, salt+2, 4))
			arc := 180 + hitEffectRandom(effect, salt+3, 91)
			step := arc / 20
			tilt := degreesToRadians(float64(i%4*45 + i/4*9))
			s, c := math.Sincos(tilt)
			edge := modelPoint3{x: -halfWidth * s, y: -halfWidth * c}
			point := func(a int) modelPoint3 {
				sin, cos := math.Sincos(degreesToRadians(float64(a)))
				return add3(origin, modelPoint3{x: 2.8 * cos * c, y: -2.8 * cos * s, z: 2.8 * sin})
			}
			for j := 0; j < arc/step; j++ {
				from, to := point(angle+j*step), point(angle+(j+1)*step)
				start := len(vertices)
				appendEffectQuad(&vertices, &indices, texture,
					[4]modelPoint3{sub3(from, edge), sub3(to, edge), add3(to, edge), add3(from, edge)}, color.RGBA{255, 89, 182, 100})
				for k, u := range []int{min(j, 19), min(j+1, 20), min(j+1, 20), min(j, 19)} {
					vertices[start+k].SrcX = float32(u) / 21 * float32(texture.Bounds().Dx())
				}
			}
		}
	case effectMarionetteLink:
		if _, ok, _ := actorForCombatID(ctx, effect.targetID); !ok {
			return
		}
		target := effect
		target.actorID = effect.targetID
		to := effectWorldAnchor(ctx, target, now)
		origin.y, to.y = origin.y+1.6, to.y+1.6
		if n < 15 {
			to = add3(origin, mul3(sub3(to, origin), math.Sin(degreesToRadians(float64(n*6)))))
		}
		phase := degreesToRadians(float64(n % 360))
		halfStep := math.Pi / 360
		radius := (.7 + .005*math.Sin(phase/2)*math.Sin(phase/2+halfStep)/math.Sin(halfStep)) / 5
		for layer := 1; layer <= 2; layer++ {
			appendMonkTrail(&vertices, &indices, texture, origin, to, radius*float64(layer), 10, color.RGBA{255, 89, 182, uint8(min(80, n) * layer)})
		}
	case effectBottomHermode:
		yaw := degreesToRadians(float64(hitEffectRandom(effect, 0, 360) + n))
		phase := degreesToRadians(float64(hitEffectRandom(effect, 1, 360) + n))
		radius := (1.44 + .16*math.Sin(phase)) / 5
		var bottom, top [4]modelPoint3
		for i := range bottom {
			s, c := math.Sincos(yaw + float64(i)*math.Pi/2)
			bottom[i] = add3(origin, modelPoint3{x: radius * c, y: (12 - 2*math.Sin(yaw)) / 5, z: radius * s})
			top[i] = add3(bottom[i], modelPoint3{y: 1.6 / 5})
		}
		grey := uint8(hitEffectRandom(effect, 2, 31))
		alpha := uint8(min(140, 2*n))
		appendEffectQuad(&vertices, &indices, texture, bottom, color.RGBA{grey, grey, 250, alpha})
		appendEffectQuad(&vertices, &indices, texture, top, color.RGBA{grey, grey, 250, alpha})
		for i, blue := range []uint8{220, 190, 130, 160} {
			j := (i + 1) % 4
			appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{top[i], top[j], bottom[j], bottom[i]}, color.RGBA{grey, grey, blue, alpha})
		}
	default: // The server selects and positions the Tarot card through ZC_NOTIFY_EFFECT.
		alpha := tarotCardAlpha(frame)
		if alpha == 0 {
			return
		}
		size := 1.6 * (1 + .05*math.Sin(degreesToRadians(float64(2*n))))
		origin.y = effect.actorOrigin.y + 6
		drawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, origin.x, origin.z, origin.y,
			size, math.Sqrt(3)*size, 0, color.RGBA{255, 255, 255, uint8(alpha)}, texturedEffectBillboardDrawOptions(false, false))
		return
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
	}
}
