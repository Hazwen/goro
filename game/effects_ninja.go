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

func ninjaCastingSkill(id uint16) bool {
	return id == db.SkillNJHuuma || id == db.SkillNJBunsinjyutsu || id >= db.SkillNJKouenka && id <= db.SkillNJNen
}

func ninjaSkillActionSpec(id uint16) (skillActionSpec, bool) {
	first, last, duration := 0, 1, 500*time.Millisecond
	switch id {
	case db.SkillNJSyuriken, db.SkillNJKunai, db.SkillNJHuuma, db.SkillNJZenynage,
		db.SkillNJUtsusemi, db.SkillNJKouenka, db.SkillNJHyousyouraku,
		db.SkillNJHuujin, db.SkillNJRaigekisai, db.SkillNJKamaitachi, db.SkillNJNen:
		first, last, duration = 2, 5, time.Second
	case db.SkillNJTatamigaeshi, db.SkillNJKasumikiri, db.SkillNJShadowjump,
		db.SkillNJBunsinjyutsu, db.SkillNJKaensin, db.SkillNJBakuenryu, db.SkillNJHyousensou, db.SkillNJSuiton:
	case db.SkillNJIssen:
		return defaultSkillActionSpec, true
	case db.SkillNJKirikage:
		return newSkillActionSpec(skillActorActionAttack2, false, &idleSkillActionSpec), true
	default:
		return skillActionSpec{}, false
	}
	step := 6 * db.EffectFrameDuration
	next := skillActionSpec{defined: true, action: skillActorActionSkill, hasFrame: true,
		frame: last, duration: duration - time.Duration(last-first)*step}
	for frame := last - 1; frame >= first; frame-- {
		next = skillActionSpec{defined: true, action: skillActorActionSkill, hasFrame: true,
			frame: frame, duration: step, next: cloneSkillActionSpec(&next)}
	}
	return next, true
}

func (m *WorldMode) addNinjaBeforeHitEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) bool {
	if action.SkillID < db.SkillNJSyuriken || action.SkillID > db.SkillNJIssen {
		return false
	}
	for _, id := range skillBeforeHitEffectIDs(action.SkillID) {
		if !m.addWorldEffectBetweenAt(ctx, id, action.TargetID, action.SourceID, starts) {
			continue
		}
		effect := &m.worldEffects[len(m.worldEffects)-1]
		if id >= effectThrowItem7 && id <= effectThrowItem10 {
			m.scheduleSound(starts, "effect\\닌자_던지기.wav")
		}
		if id == effectStin4 {
			effect.hitCount = actionVisualHitCount(action)
			if effect.hitCount > 36 {
				effect.hitCount = 1
			}
		}
	}
	return true
}

// Each petal captures its target's position when it appears, then drifts on its own.
type ninjaPetals struct {
	tick      int
	particles [20]soulLightParticle
}

func (p *ninjaPetals) advance(ctx client.Context, effect worldEffect, now time.Time) {
	end := min(int(now.Sub(effect.starts)/db.EffectFrameDuration)+1, 110)
	for p.tick < end {
		if p.tick < len(p.particles) {
			seed := p.tick * 512
			particle := &p.particles[p.tick]
			particle.seed, particle.kind = seed, hitEffectRandom(effect, seed+4, 3)
			particle.speed = float64(2+hitEffectRandom(effect, seed+5, 3)) * .15 / 5
			particle.radius = float64(22+hitEffectRandom(effect, seed+6, 6)) / 70
			particle.position = effectWorldAnchor(ctx, effect, effect.starts.Add(time.Duration(p.tick)*db.EffectFrameDuration))
			s, c := math.Sincos(degreesToRadians(float64(hitEffectRandom(effect, seed+7, 360))))
			particle.position.x += 1.6 * c
			particle.position.z += 1.6 * s
			for i := range particle.angles {
				particle.angles[i] = hitEffectRandom(effect, seed+i, 360)
			}
		}
		p.tick++
		for i := 0; i < min(p.tick, len(p.particles)); i++ {
			particle := &p.particles[i]
			if particle.age >= 83 {
				continue
			}
			particle.age++
			advanceWanderingAngles(particle, effect, particle.age+20)
			particle.position.y += particle.speed
			particle.position.x += .03 * math.Sin(degreesToRadians(float64(particle.angles[0])))
			particle.position.z += .03 * math.Sin(degreesToRadians(float64(particle.angles[2])))
		}
	}
}

func (m *WorldMode) drawNinjaPetals(screen *render.Frame, ctx client.Context, projection sceneProjection, c worldEffectComponent, effect worldEffect, now time.Time) {
	if effect.ninjaPetals == nil {
		return
	}
	effect.ninjaPetals.advance(ctx, effect, now)
	for i := 0; i < min(effect.ninjaPetals.tick, 20); i++ {
		p := effect.ninjaPetals.particles[i]
		alpha := 250 - 3*p.age
		if alpha < 3 {
			continue
		}
		motion := 1 + (p.age-1)/4
		m.drawEffectParticleSpriteAction(screen, ctx, projection, c.spriteFile, p.position, p.kind, motion, p.radius, 0, color.RGBA{255, 255, 255, uint8(alpha)})
	}
}

type ninjaWindSample struct {
	position    modelPoint3
	alpha, spin int
}

type ninjaWindHead struct {
	samples        [4]ninjaWindSample
	heading, speed float64
	fading         bool
}

type ninjaWind struct {
	tick    int
	heads   [36]ninjaWindHead
	origins [5]modelPoint3
	born    int
}

func (p *ninjaWind) advance(ctx client.Context, effect worldEffect, now time.Time) {
	end := min(int(now.Sub(effect.starts)/db.EffectFrameDuration)+1, 200)
	for p.tick < end {
		p.tick++
		at := effect.starts.Add(time.Duration(p.tick-1) * db.EffectFrameDuration)
		target := effectWorldAnchor(ctx, worldEffect{actorID: effect.targetID}, at)
		_, exists, _ := actorForCombatID(ctx, effect.targetID)
		for i := 0; i < max(1, min(effect.hitCount, 36)); i++ {
			h := &p.heads[i]
			turn := 1.0
			if i%2 == 0 {
				turn = -1
			}
			if p.tick == 1 {
				for j := range h.samples {
					h.samples[j].position = effect.actorOrigin
					h.samples[j].position.y += 2
				}
				h.heading = math.Atan2(target.z-effect.actorOrigin.z, target.x-effect.actorOrigin.x)*180/math.Pi - turn*float64(25+hitEffectRandom(effect, i*8, 21))
				h.speed = float64(10+hitEffectRandom(effect, i*8+1, 21)) / 50
			}
			if !exists || p.tick > 20 {
				h.fading = true
			}
			s := &h.samples[0]
			if h.fading && s.alpha <= 0 {
				continue
			}
			s.spin = p.tick * 30
			if p.tick > 1 && p.tick <= 20 {
				s.alpha = min(s.alpha+8, 120)
			}
			if p.tick > 5 {
				if !h.fading {
					toward := math.Mod(math.Atan2(target.z-s.position.z, target.x-s.position.x)*180/math.Pi+360, 360)
					if math.Abs(h.heading-toward) <= 3 {
						h.fading = true
					} else {
						h.heading += 5 * turn
					}
				} else {
					for j := range h.samples {
						h.samples[j].alpha = max(0, h.samples[j].alpha-5)
					}
					h.heading += turn
				}
			}
			h.heading = math.Mod(h.heading+360, 360)
			sin, cos := math.Sincos(degreesToRadians(h.heading))
			s.position.x += cos * h.speed
			s.position.z += sin * h.speed
			if p.tick%3 == 0 {
				for j := 3; j > 0; j-- {
					h.samples[j] = h.samples[j-1]
					h.samples[j].alpha = h.samples[j].alpha * 8 / 10
				}
			}
		}
	}
}

func (m *WorldMode) drawNinjaWind(screen *render.Frame, ctx client.Context, c worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, c.textureFile)
	if texture == nil || effect.ninjaWind == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	quad := func(position modelPoint3, spin, alpha int, radius, heading float64, vertical bool) {
		if alpha <= 0 {
			return
		}
		var corners [4]modelPoint3
		for i := range corners {
			sin, cos := math.Sincos(degreesToRadians(float64(spin + (i+1)*90)))
			offset := modelPoint3{x: cos * radius, z: sin * radius}
			if vertical {
				offset = modelPoint3{x: cos * radius * math.Cos(heading), z: cos * radius * math.Sin(heading), y: -sin * radius}
			}
			corners[3-i] = add3(position, offset)
		}
		appendEffectQuad(&vertices, &indices, texture, corners, color.RGBA{80, 255, 80, uint8(alpha)})
	}
	if effect.effectID == effectStin4 {
		effect.ninjaWind.advance(ctx, effect, now)
		for i := 0; i < max(1, min(effect.hitCount, 36)); i++ {
			for j, s := range effect.ninjaWind.heads[i].samples {
				quad(s.position, s.spin, s.alpha, float64(20+hitEffectRandom(effect, i*8+2+j, 11))/10, 0, false)
			}
		}
	} else {
		tick := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
		for i := 0; i < 5; i++ {
			process := tick - i - 3
			if process <= 0 {
				continue
			}
			for effect.ninjaWind.born <= i {
				birth := effect.ninjaWind.born
				origin := effect.actorOrigin
				if effect.actorID != 0 {
					origin = effectWorldAnchor(ctx, worldEffect{actorID: effect.actorID}, effect.starts.Add(time.Duration(birth)*db.EffectFrameDuration))
				}
				effect.ninjaWind.origins[birth] = origin
				effect.ninjaWind.born++
			}
			origin := effect.ninjaWind.origins[i]
			direction := sub3(add3(effect.actorOrigin, effect.projectileDirection), origin)
			direction.y = 0
			if length := math.Hypot(direction.x, direction.z); length > 0 {
				direction = mul3(direction, 1/length)
			}
			heading := math.Atan2(direction.z, direction.x)
			switch i {
			case 1:
				origin.x++
			case 2:
				origin.x--
			case 3:
				origin.z++
			case 4:
				origin.z--
			}
			origin.y += float64(10-hitEffectRandom(effect, 30+i, 5)) / 5

			for j := 0; j < 4; j++ {
				sample := process
				if j > 0 {
					sample = min(process, 29)/3*3 - (j-1)*3
				}
				if sample <= 0 {
					continue
				}
				rise, fall := min(sample-1, 9), max(0, sample-10)
				alpha := max(0, 6*rise-3*fall)
				for n := 0; n < j; n++ {
					alpha = alpha * 8 / 10
				}
				distance := 1.5 * (1 + 1.02*(math.Pow(1.02, float64(rise))-1)/.02 + math.Pow(1.02, 9)*1.04*(math.Pow(1.04, float64(fall))-1)/.04) / 5
				position := add3(origin, mul3(direction, distance))
				angle := heading
				if j > 0 {
					angle = degreesToRadians(float64(hitEffectRandom(effect, 40+i*4+j, 360)))
				}
				quad(position, sample*30, alpha, 4, angle, true)
			}
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
	}
}
