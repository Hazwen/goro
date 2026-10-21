package game

import (
	"image/color"
	"math"
	"slices"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) applyTaekwonSkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	id := notify.SourceID
	if _, ok, _ := actorForCombatID(ctx, id); !ok {
		return
	}
	if isLocalActor(ctx, id) {
		id = localSkillTarget(ctx)
	}
	switch notify.SkillID {
	case db.SkillTKRun:
		if m.hasActiveWorldEffect(effectStopRun, id, now) {
			m.addWorldEffectAt(ctx, effectStopEffect, id, now)
		} else if !m.hasActiveWorldEffect(effectRun, id, now) {
			m.addWorldEffectAt(ctx, effectRun, id, now)
		}
	case db.SkillTKHighjump:
		m.removeWorldEffect(effectJumpBody, notify.SourceID)
		m.removeWorldEffect(effectJumpBody, id)
		m.scheduledSounds = slices.DeleteFunc(m.scheduledSounds, func(sound scheduledSound) bool {
			return (sound.actorID == notify.SourceID || sound.actorID == id) && sound.at.After(now) && slices.Contains(sound.paths, "effect\\T_회피2.wav")
		})
	case db.SkillTKDodge:
		m.scheduleSoundVolume(now, 4.0/15, "effect\\T_낙법.wav")
	case db.SkillTKMission:
		m.scheduleSoundVolume(now, 4.0/10, "effect\\T_피링.wav")
	case db.SkillTKSevenwind:
		if notify.Amount < 1 || notify.Amount > 7 {
			return
		}
		m.scheduleSound(now, "effect\\T_바람방출.wav")
		for _, effectID := range []int{effectStormKick3, effectBeginAsura1 + int(notify.Amount) - 1} {
			m.removeWorldEffect(effectID, id)
			m.addWorldEffectAt(ctx, effectID, id, now)
		}
	}
}

func (m *WorldMode) drawTaekwonStorm(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	age := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	alpha := min(age, 40) - max(0, age-60)
	if alpha <= 0 || len(component.textureFiles) != 2 {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFiles[0])
	if texture == nil {
		return
	}
	rotation := hitEffectRandom(effect, 0, 360)
	for tick := 1; tick <= age; tick++ {
		rotation += 3 + hitEffectRandom(effect, tick, 3)
	}
	var vertices []render.Vertex3D
	var indices []uint16
	tint := color.RGBA{50, 50, 255, uint8(alpha)}
	if effect.effectID == effectStormKick3 {
		tint.R, tint.G = 130, 130
	}
	radius := 40.0
	for layer := 0; layer < 10; layer++ {
		if layer > 0 {
			radius -= 2 * (2.2 - .2*float64(layer))
		}
		var corners [4]modelPoint3
		for corner := range corners {
			s, c := math.Sincos(degreesToRadians(float64(rotation + layer*45 + corner*90)))
			point := modelPoint3{x: c * radius / 5, z: s * radius / 5, y: float64(50-layer*5) / 5}
			if corner < 2 {
				point.y += .2
			}
			corners[3-corner] = add3(effect.actorOrigin, point)
		}
		appendEffectQuad(&vertices, &indices, texture, corners, tint)
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(effect.effectID == effectStormKick, false))
	m.drawTaekwonWind(screen, ctx, component.textureFiles[1], effect, age)
}

func (m *WorldMode) drawTaekwonWind(screen *render.Frame, ctx client.Context, file string, effect worldEffect, age int) {
	alpha := min(age, 11) - max(0, age-50)
	texture := m.effectFileTexture(ctx.Resources, file)
	if alpha <= 0 || texture == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	span := min(30+age*3, 180)
	step := max(1, span/21)
	for ring := 0; ring < 8; ring++ {
		i := ring % 4
		height, radius := float64(48-i*5), float64(14-i*2)
		if ring >= 4 {
			height, radius = float64(28-i*5), float64(6-i)
		}
		radius += .1 * float64(max(0, age-12))
		pitch := degreesToRadians(float64(80 + hitEffectRandom(effect, 200+ring, 21)))
		point := func(angle int, top bool) modelPoint3 {
			s, c := math.Sincos(degreesToRadians(float64(angle + i*90 + age*5)))
			r, y := radius, height
			if top {
				r += math.Cos(pitch)
				y += math.Sin(pitch)
			}
			return add3(effect.actorOrigin, modelPoint3{x: c * r / 5, z: s * r / 5, y: y / 5})
		}
		for angle, section := step, 0; angle < span; angle, section = angle+step, section+1 {
			start := len(vertices)
			appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{
				point(angle-step, true), point(angle, true), point(angle, false), point(angle-step, false),
			}, color.RGBA{255, 255, 255, uint8(alpha)})
			w := float32(texture.Bounds().Dx())
			for j := 0; j < 4; j++ {
				u := float32(min(section, 19)) / 21
				if j == 1 || j == 2 {
					u += 1.0 / 21
				}
				vertices[start+j].SrcX = u * w
			}
		}
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
}

func (m *WorldMode) drawTaekwonStop(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	age := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	alpha := 5*min(age, 5) - 2*max(0, age-9)
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if alpha <= 0 || texture == nil {
		return
	}
	origin := effectWorldAnchor(ctx, effect, now)
	var vertices []render.Vertex3D
	var indices []uint16
	for ray := 0; ray < 8; ray++ {
		for layer := -1; layer <= 1; layer++ {
			s, c := math.Sincos(degreesToRadians(float64(ray*45 + layer)))
			bottom := add3(origin, modelPoint3{x: c * float64(1+age) / 5, z: s * float64(1+age) / 5})
			tip := add3(origin, modelPoint3{x: c * float64(3+age*2) / 5, z: s * float64(3+age*2) / 5})
			height := math.Min(2, .2*float64(age)) / 5
			low, high, a := height*.2, height, alpha
			if layer != 0 {
				low, high, a = height*.1, height*.4, alpha/3
			}
			appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{
				add3(bottom, modelPoint3{y: low}), add3(tip, modelPoint3{y: high}), tip, bottom,
			}, color.RGBA{255, 255, 255, uint8(a)})
		}
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
}

func taekwonSkillActionSpec(skillID uint16) (skillActionSpec, bool) {
	pose := func(action skillActorAction, frame int, duration time.Duration) skillActionSpec {
		return skillActionSpec{defined: true, action: action, hasFrame: true, frame: frame, duration: duration}
	}
	switch skillID {
	case db.SkillTKRun:
		return idleSkillActionSpec, true
	case db.SkillTKReadystorm:
		return pose(skillActorActionSkill, 0, 2*time.Second), true
	case db.SkillTKStormkick:
		return pose(skillActorActionSkill, 1, 2*time.Second), true
	case db.SkillTKReadydown:
		return pose(skillActorActionSkill, 2, 2*time.Second), true
	case db.SkillTKReadyturn:
		return pose(skillActorActionSkill, 3, 2*time.Second), true
	case db.SkillTKReadycounter:
		return pose(skillActorActionSkill, 4, 2*time.Second), true
	case db.SkillTKDodge:
		return pose(skillActorActionPickup, 1, 2*time.Second), true
	case db.SkillTKJumpkick:
		// EF_JUMPKICK replaces the three-second pose at tick five with a 25 ms hold.
		return pose(skillActorActionSkill, 5, 5*db.EffectFrameDuration+25*time.Millisecond), true
	case db.SkillTKDownkick:
		return taekwonKickSequence(skillActorActionAttack2, [5]int{3, 5, 7, 8, 2}, 5), true
	case db.SkillTKTurnkick:
		return taekwonKickSequence(skillActorActionAttack3, [5]int{4, 5, 6, 7, 2}, 5), true
	case db.SkillTKCounter:
		return taekwonKickSequence(skillActorActionAttack1, [5]int{2, 3, 4, 4, 4}, 4), true
	case db.SkillTKHighjump:
		return taekwonJumpSequence(false), true
	default:
		return skillActionSpec{}, false
	}
}

// Native SetForceAnimation2 plays five selected ACT motions, then holds the last
// one until its timeout. The existing animation chain also permits the landing
// pose to finish in a different ACT action.
func taekwonKickSequence(action skillActorAction, frames [5]int, ticks int) skillActionSpec {
	step := time.Duration(ticks) * db.EffectFrameDuration
	next := skillActionSpec{
		defined: true, action: action, hasFrame: true, frame: frames[4],
		duration: 500*time.Millisecond - 4*step,
	}
	for i := 3; i >= 0; i-- {
		next = skillActionSpec{
			defined: true, action: action, hasFrame: true, frame: frames[i],
			duration: step, next: cloneSkillActionSpec(&next),
		}
	}
	return next
}

func taekwonJumpSequence(casting bool) skillActionSpec {
	step := 10 * db.EffectFrameDuration
	action, frames := skillActorActionPickup, [5]int{1, 1, 1, 1, 5}
	finalAction := skillActorActionWalk
	if casting {
		action, frames = skillActorActionAttack1, [5]int{0, 0, 0, 1, 1}
		finalAction = action
	}
	next := skillActionSpec{
		defined: true, action: finalAction, hasFrame: true, frame: frames[4],
		duration: 3*time.Second - 4*step,
	}
	for i := 3; i >= 0; i-- {
		next = skillActionSpec{
			defined: true, action: action, hasFrame: true, frame: frames[i],
			duration: step, next: cloneSkillActionSpec(&next),
		}
	}
	return next
}

type taekwonJumpParticles struct {
	tick   int
	origin modelPoint3
	lights [30]wanderingEffectParticle
}

func (p *taekwonJumpParticles) advance(ctx client.Context, effect worldEffect, now time.Time) {
	end := min(int(now.Sub(effect.starts)/db.EffectFrameDuration)+1, 180)
	for p.tick < end {
		p.tick++
		if p.tick == 36 {
			p.origin = effectWorldAnchor(ctx, effect, effect.starts.Add(35*db.EffectFrameDuration))
			for i := range p.lights {
				particle := &p.lights[i]
				particle.seed = i * 10
				angle := deterministicAngle(effect, particle.seed+4)
				particle.position = modelPoint3{x: .5 * math.Cos(angle), z: .5 * math.Sin(angle), y: float64(hitEffectRandom(effect, particle.seed+5, 8)) / 5}
				particle.age = -hitEffectRandom(effect, particle.seed+6, 26)
				particle.radius = (3 + .1*float64(hitEffectRandom(effect, particle.seed+7, 11))) / 5
				particle.speed = (.1 + .01*float64(hitEffectRandom(effect, particle.seed+8, 16))) / 5
				for j := range particle.angles {
					particle.angles[j] = hitEffectRandom(effect, particle.seed+j, 360)
				}
			}
		}
		for i := range p.lights {
			particle := &p.lights[i]
			particle.age++
			if particle.age <= 0 || particle.age >= 102 {
				continue
			}
			advanceWanderingAngles(particle, effect, p.tick)
			if particle.position.y > 0 {
				particle.position.x += .03 / 5 * math.Sin(degreesToRadians(float64(particle.angles[0])))
				particle.position.z += .03 / 5 * math.Sin(degreesToRadians(float64(particle.angles[2])))
			}
			particle.position.y += particle.speed
		}
	}
}

func (m *WorldMode) drawTaekwonJump(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	if effect.taekwonJump == nil || len(component.textureFiles) != 4 {
		return
	}
	p := effect.taekwonJump
	p.advance(ctx, effect, now)
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if frame < 35 {
		return
	}
	_, up, _, ok := projection.BillboardBasis(p.origin.x, p.origin.z, p.origin.y)
	if !ok {
		return
	}
	for i, file := range component.textureFiles {
		texture := m.effectFileTexture(ctx.Resources, file)
		if texture == nil {
			continue
		}
		if i == 0 {
			for _, light := range p.lights {
				alpha := max(0, 5*clampInt(light.age, 0, 29)-2*max(0, light.age-29))
				if alpha == 0 || light.position.y < light.radius/math.Sqrt2*up.y {
					continue
				}
				pos := add3(p.origin, light.position)
				size := math.Sqrt2 * light.radius
				drawTexturedEffectBillboardRotatedXY(screen, projection, texture, pos.x, pos.z, pos.y, size, size, 0, color.RGBA{255, 255, 255, uint8(alpha)}, true)
			}
			continue
		}
		for particle := 0; particle < 64; particle++ {
			if [...]int{1, 2, 3, 2}[particle/4%4] != i {
				continue
			}
			seed := 1000 + particle*4
			age := frame - 34 - hitEffectRandom(effect, seed, 11)
			alpha := max(0, 2*clampInt(age, 0, 25)-max(0, age-70))
			if alpha == 0 {
				continue
			}
			angle := degreesToRadians(float64(particle/16*90 + hitEffectRandom(effect, seed+1, 90)))
			radius := (4 + .03*float64(min(age, 90))) / 5
			pos := add3(p.origin, modelPoint3{x: radius * math.Cos(angle), z: radius * math.Sin(angle), y: (4 + 1.5*math.Sin(degreesToRadians(float64(min(age, 90)*2)))) / 5})
			size := math.Sqrt2 * (2.5 + .2*float64(hitEffectRandom(effect, seed+2, 11))) / 5
			drawTexturedEffectBillboardRotatedXY(screen, projection, texture, pos.x, pos.z, pos.y, size, size, 0, color.RGBA{255, 255, 255, uint8(alpha)}, false)
		}
	}
}

type taekwonRunTrail struct {
	tick     int
	position modelPoint3
}

func (m *WorldMode) advanceTaekwonFootprints(ctx client.Context, now time.Time) {
	var footprints []worldEffect
	for _, effect := range m.worldEffects {
		p := effect.taekwonRun
		if p == nil || effect.expired(now) || now.Before(effect.starts) {
			continue
		}
		end := int(now.Sub(effect.starts) / db.EffectFrameDuration)
		p.tick = max(p.tick, (end-340)/7*7)
		for p.tick+7 <= end {
			p.tick += 7
			at := effect.starts.Add(time.Duration(p.tick) * db.EffectFrameDuration)
			position := effectWorldAnchor(ctx, effect, at)
			delta := sub3(position, p.position)
			p.position = position
			if p.tick <= 7 || delta.x == 0 && delta.z == 0 {
				continue
			}
			id := effectFoot3
			if p.tick/7%2 == 0 {
				id = effectFoot4
			}
			footprints = append(footprints, worldEffect{
				effectID: id, actorOrigin: position, projectileDirection: delta, hasActorTransform: true,
				starts: at, expires: at.Add(340 * db.EffectFrameDuration), duration: 340 * db.EffectFrameDuration,
			})
		}
	}
	m.worldEffects = append(m.worldEffects, footprints...)
}

func (m *WorldMode) drawTaekwonFoot(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	age := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	alpha := 150 - (age+1)/2
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if age <= 0 || alpha <= 0 || texture == nil {
		return
	}
	angle := math.Atan2(effect.projectileDirection.z, effect.projectileDirection.x) + 3*math.Pi/4
	var corners [4]modelPoint3
	for i := range corners {
		s, c := math.Sincos(angle + float64(i)*math.Pi/2)
		corners[3-i] = add3(effect.actorOrigin, modelPoint3{x: c * 1.08, z: s * 1.08, y: .04})
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for i := 0; i < 2; i++ {
		appendEffectQuad(&vertices, &indices, texture, corners, color.RGBA{255, 255, 255, uint8(alpha)})
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
}
