package game

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

func TestDancerPerformanceOwnership(t *testing.T) {
	ctx := archerEffectContext()
	for _, skill := range []uint16{db.SkillDCUglydance, db.SkillDCHumming, db.SkillDCDontforgetme, db.SkillDCFortunekiss, db.SkillDCServiceforyou} {
		for _, ground := range []bool{false, true} {
			m := &WorldMode{}
			if ground {
				m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: skill, SourceID: 2000000})
			} else {
				m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 2000000, Result: 1})
			}
			sounds := 1
			if skill == db.SkillDCUglydance {
				sounds = 0
			}
			if len(m.worldEffects) != 0 || len(m.scheduledSounds) != sounds || !m.actorAnims[2000000].pingPong {
				t.Fatalf("dance %d ground=%t must start only its pose and sound", skill, ground)
			}
			m.stopPerformance(ctx, network.StatusEffectChange{StatusID: db.StatusBdplaying, ActorID: 100})
			if len(m.actorAnims) != 0 {
				t.Fatal("dance continued after status end")
			}
		}
	}
	m := &WorldMode{}
	for unit := uint16(171); unit <= 175; unit++ {
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: uint32(unit), UnitID: unit, X: 15, Y: 20, Visible: true})
	}
	if len(m.worldEffects) != 5 || len(m.scheduledSounds) != 0 {
		t.Fatal("five dance units must create five silent visuals")
	}
	if !m.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 172, Value: 171}) || m.worldEffects[len(m.worldEffects)-1].effectID != effectBottomUglyDanceGround {
		t.Fatal("overlapping dances must change to Ugly Dance")
	}
	for unit := uint32(171); unit <= 175; unit++ {
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: unit})
	}
	if len(m.worldEffects) != 0 {
		t.Fatal("dance visual survived ground unit removal")
	}
}

func TestDancerScreamAndSlingingArrow(t *testing.T) {
	ctx := archerEffectContext()
	m := &WorldMode{}
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillDCScream, SourceID: 2000000, TargetID: 300, Result: 1})
	anim := m.actorAnims[2000000]
	if !anim.hasFixedMotion || anim.fixedMotion != 0 || anim.actionFamily != spriteActionPCSkill || anim.duration != 1200*time.Millisecond {
		t.Fatal("Scream must hold skill frame zero for 1200ms")
	}
	if len(m.worldEffects) != 1 || m.worldEffects[0].actorID != 2000000 || len(m.scheduledSounds) != 0 {
		t.Fatal("Scream's silent chat effect must belong to the caster")
	}
	if _, ok := m.actorAnimation(2000000, anim.started.Add(1200*time.Millisecond)); ok {
		t.Fatal("Scream pose outlived its native duration")
	}
	m = &WorldMode{playerView: humanoidSoundView(spriteActionPCAttack3, 4, 2, []string{"atk"})}
	a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillDCThrowarrow, SourceSpeed: 500, HitCount: 1, Damage: 100}
	m.applyActorActionNotify(ctx, a)
	anim = m.actorAnims[a.SourceID]
	if len(m.worldEffects) != 2 || anim.actionFamily != spriteActionPCAttack3 {
		t.Fatal("Slinging Arrow needs ACT 88, one arrow, and one impact")
	}
	arrow, hit := m.worldEffects[0], m.worldEffects[1]
	if arrow.effectID != effectArrowShot || hit.effectID != effectHit1 || arrow.starts.Sub(anim.started) != 250*time.Millisecond || hit.starts.Sub(arrow.starts) != referenceBowArrowFlightDuration {
		t.Fatal("Slinging Arrow must use the ACT marker and native arrow flight time")
	}
	a.Action = 5
	m.applyActorActionNotify(ctx, a)
	if len(m.worldEffects) != 2 {
		t.Fatal("splash notification replayed Slinging Arrow")
	}
}

func TestDancerWinkAndClientAssets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	projection := newSceneProjectionForTarget(800, 600, 15, 20, 0)
	for unit := uint16(171); unit <= 175; unit++ {
		m := &WorldMode{}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: unit, X: 15, Y: 20, Visible: true})
		effect := m.worldEffects[0]
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, effect.starts.Add(20*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		billboards := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if commands.Len()+billboards.Len() == 0 {
			t.Fatalf("dance unit %d drew nothing", unit)
		}
		if unit == 173 || unit == 175 {
			vertices := commands.Index(0).FieldByName("Vertices")
			count, red, green := 16, 155.0/255, 1.0
			if unit == 175 {
				count, red, green = 8, 1, 150.0/255
			}
			if vertices.Len() != count || math.Abs(vertices.Index(0).FieldByName("ColorR").Float()-red) > .001 || math.Abs(vertices.Index(0).FieldByName("ColorG").Float()-green) > .001 {
				t.Fatalf("dance unit %d has incorrect ribbon count or color", unit)
			}
		}
	}
	for skill, spec := range db.SkillEffects {
		if isDancerPerformance(skill) || skill == db.SkillDCWinkcharm {
			for _, sound := range spec.SFX {
				if _, err := ctx.Resources.ReadFile("data/wav/" + sound); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	m := &WorldMode{}
	n := network.SkillNoDamageNotify{SkillID: db.SkillDCWinkcharm, SourceID: 2000000, TargetID: 300, Result: 1}
	m.applySkillNoDamageNotify(ctx, n)
	if len(m.worldEffects) != 2 || len(m.scheduledSounds) != 1 {
		t.Fatal("Wink needs its kiss emote, directional sprite, and one sound")
	}
	if e := m.worldEffects[0]; e.effectID != effectEmotion || e.actorID != n.SourceID || e.spriteFrameOverride != 30 {
		t.Fatal("Wink must display the kiss emote on the caster")
	}
	e := m.worldEffects[1]
	if e.effectID != effectWink || e.actorID != n.SourceID || e.targetID != n.TargetID || e.actorOrigin.x != 20.5 || e.projectileDirection.x != -10 {
		t.Fatal("Wink must capture the target position and caster-to-target orientation")
	}
	frame := render.NewFrame(800, 600)
	m.drawWorldEffects(frame, ctx, projection, e.starts.Add(6*db.EffectFrameDuration), false)
	if reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() == 0 {
		t.Fatal("Wink sprite did not draw")
	}
	m.removeWorldEffectsForActor(n.SourceID)
	if len(m.worldEffects) != 0 {
		t.Fatal("Wink must disappear when its caster leaves view")
	}
	m = &WorldMode{}
	start := time.Now()
	m.startSkillSourceCastAnimation(ctx, n.SourceID, n.SkillID, time.Second, start)
	m.addSkillCastEffects(ctx, n.SkillID, 0, n.SourceID, n.TargetID, 0, 0, time.Second, start, "test")
	if anim := m.actorAnims[n.SourceID]; !anim.loop || !anim.pingPong || anim.actionFamily != spriteActionPCSkill {
		t.Fatal("Wink must dance during its cast")
	}
	if len(m.worldEffects) != 2 {
		t.Fatal("Wink cast needs only the normal circle and lock-on, without an aura")
	}
}
