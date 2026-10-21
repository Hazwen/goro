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

func gunslingerSkillActionSpec(id uint16) (skillActionSpec, bool) {
	action, first, last, duration := skillActorActionSkill, 0, 3, time.Second
	switch id {
	case db.SkillGSFling, db.SkillGSMadnesscancel, db.SkillGSAdjustment, db.SkillGSIncreasing, db.SkillGSDesperado, db.SkillGSGatlingfever:
	case db.SkillGSGlittering:
		first, last, duration = 4, 5, 500*time.Millisecond
	case db.SkillGSCracker, db.SkillGSGrounddrift:
		action, last, duration = skillActorActionAttack1, 2, 500*time.Millisecond
	case db.SkillGSTripleaction, db.SkillGSBullseye, db.SkillGSMagicalbullet, db.SkillGSTracking, db.SkillGSDisarm,
		db.SkillGSPiercingshot, db.SkillGSRapidshower, db.SkillGSDust, db.SkillGSFullbuster, db.SkillGSSpreadattack:
		return attackSkillActionSpec, true
	default:
		return skillActionSpec{}, false
	}
	step := 6 * db.EffectFrameDuration
	next := skillActionSpec{defined: true, action: action, hasFrame: true, frame: last, duration: duration - time.Duration(last-first)*step}
	for frame := last - 1; frame >= first; frame-- {
		next = skillActionSpec{defined: true, action: action, hasFrame: true, frame: frame, duration: step, next: cloneSkillActionSpec(&next)}
	}
	next.randomReverse = first == 0 && last == 3
	return next, true
}

func gunslingerCastAction(id uint16, now time.Time) (skillActionSpec, bool) {
	spec := skillActionSpec{defined: true, action: skillActorActionSkill, hasFrame: true, frame: 1 + int(now.UnixNano()&1), duration: 500 * time.Millisecond}
	switch id {
	case db.SkillGSBullseye, db.SkillGSGatlingfever, db.SkillGSTracking, db.SkillGSMadnesscancel,
		db.SkillGSAdjustment, db.SkillGSDisarm, db.SkillGSPiercingshot, db.SkillGSRapidshower:
	case db.SkillGSDust, db.SkillGSFullbuster, db.SkillGSSpreadattack, db.SkillGSGrounddrift:
		spec.action, spec.frame = skillActorActionAttack1, 0
	default:
		return skillActionSpec{}, false
	}
	return spec, true
}

func (m *WorldMode) addGunslingerBeforeHitEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) bool {
	if action.SkillID < db.SkillGSGlittering || action.SkillID > db.SkillGSGrounddrift {
		return false
	}
	for _, id := range skillBeforeHitEffectIDs(action.SkillID) {
		m.addWorldEffectBetweenAt(ctx, id, action.TargetID, action.SourceID, starts)
	}
	return true
}

func (m *WorldMode) initGunslingerShot(ctx client.Context, effect *worldEffect) {
	destination := effectWorldAnchor(ctx, *effect, effect.starts)
	if _, ok, _ := actorForCombatID(ctx, effect.targetID); effect.targetID != 0 && ok {
		effect.actorID, effect.targetID = effect.targetID, effect.actorID
		effect.x, effect.y, _ = effectAnchor(ctx, effect.actorID)
	}
	effect.actorOrigin = effectWorldAnchor(ctx, *effect, effect.starts)
	effect.projectileDirection = sub3(destination, effect.actorOrigin)
	effect.hasActorTransform = true
	if effect.effectID == effectChemical4 {
		return
	}
	yaw := cameraYawForMap(ctx)
	if !cameraRotationLockedForMap(ctx) {
		yaw += m.camera.yawOffset
	}
	effect.spriteFrameOverride = directedSpriteAction(mul3(effect.projectileDirection, -1), yaw)
	if effect.effectID == effectSpreadAttack {
		effect.spriteFrameOverride *= 2
	}
	effect.hasSpriteFrame = true
}

type gunslingerCoins struct {
	tick   int
	angles [5][7]int
}

func newGunslingerCoins(effect worldEffect) *gunslingerCoins {
	p := &gunslingerCoins{}
	for i := range p.angles {
		for j := range p.angles[i] {
			p.angles[i][j] = hitEffectRandom(effect, i*7+j, 360)
		}
	}
	return p
}

func (p *gunslingerCoins) advance(effect worldEffect, now time.Time) {
	end := max(0, int(now.Sub(effect.starts)/db.EffectFrameDuration)+1)
	// Catch up cosmetic wandering for at most two seconds after a long pause.
	p.tick = max(p.tick, end-120)
	for p.tick < end {
		for i := range p.angles {
			a := &p.angles[i]
			for j := 0; j < 6; j += 2 {
				seed := (p.tick*5+i)*7 + j
				if a[j+1] > a[j] {
					a[j] = min(a[j]+1, 359)
					if a[j] > a[j+1] {
						a[j+1] = hitEffectRandom(effect, seed, max(a[j], 1))
					}
				} else {
					a[j] = max(a[j]-1, 0)
					if a[j] <= a[j+1] {
						a[j+1] = 359 - hitEffectRandom(effect, seed, max(a[j], 1))
					}
				}
			}
			a[6] = (a[6] + 5 + hitEffectRandom(effect, (p.tick*5+i)*7+6, 5)) % 360
		}
		p.tick++
	}
}

func (m *WorldMode) drawGunslingerCoins(screen *render.Frame, ctx client.Context, projection sceneProjection, c worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, c.textureFile)
	p := effect.gunslingerCoins
	if texture == nil || p == nil || now.Before(effect.starts) {
		return
	}
	p.advance(effect, now)
	count := min(effect.hitCount, 5)
	for i := 0; i < count; i++ {
		a := p.angles[i]
		angle := degreesToRadians(float64(i*360/count + p.tick/2))
		cx := x + 1.4*math.Sin(angle) + .1*math.Sin(degreesToRadians(float64(a[0])))
		cy := y + 1.4*math.Cos(angle) + .1*math.Sin(degreesToRadians(float64(a[2])))
		cz := z + 3 - .1*math.Sin(degreesToRadians(float64(a[4])))
		radius := .2
		if i < effect.hitCount-5 {
			radius = .4
		}
		pulse := radius * .05 * math.Sin(degreesToRadians(float64(a[6])))
		alpha := uint8(min(p.tick, 200))
		outer, inner := math.Sqrt2*(radius+pulse), math.Sqrt2*(radius*.5+pulse)
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture, cx, cy, cz, outer, outer, math.Pi/4, color.RGBA{68, 42, 30, alpha}, true)
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture, cx, cy, cz, inner, inner, math.Pi/4, color.RGBA{255, 255, 255, alpha}, true)
	}
}
