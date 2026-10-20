package game

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestBardPerformanceOwnership(t *testing.T) {
	ctx := archerEffectContext()
	for _, skill := range []uint16{db.SkillBDLullaby, db.SkillBDRichmankim, db.SkillBDEternalchaos, db.SkillBDDrumbattlefield, db.SkillBDRingnibelungen, db.SkillBDRokisweil, db.SkillBDIntoabyss, db.SkillBDSiegfried, db.SkillBaDissonance, db.SkillBaWhistle, db.SkillBaAssassincross, db.SkillBaPoembragi, db.SkillBaAppleidun} {
		for _, ground := range []bool{false, true} {
			m := &WorldMode{}
			if ground {
				m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: skill, SourceID: 2000000})
			} else {
				m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 2000000, Result: 1})
			}
			sounds := 1
			if skill == db.SkillBaDissonance || skill == db.SkillBaWhistle {
				sounds = 0
			}
			if len(m.worldEffects) != 0 || len(m.scheduledSounds) != sounds || !m.actorAnims[2000000].loop {
				t.Fatalf("skill %d ground=%t: activation must only start performance and its one sound", skill, ground)
			}
			m.stopPerformance(ctx, network.StatusEffectChange{StatusID: db.StatusBdplaying, ActorID: 100})
			if len(m.actorAnims) != 0 {
				t.Fatal("performance end must clear both local actor aliases")
			}
		}
	}
	m := &WorldMode{}
	for unit := uint16(158); unit <= 170; unit++ {
		entry := network.SkillUnitEntry{ID: uint32(unit), UnitID: unit, X: 15, Y: 20, Visible: true}
		m.applySkillUnitEntry(ctx, entry)
		started := m.worldEffects[len(m.worldEffects)-1].starts
		m.applySkillUnitEntry(ctx, entry)
		if !m.worldEffects[len(m.worldEffects)-1].starts.Equal(started) {
			t.Fatal("ground-unit refresh restarted the song animation")
		}
	}
	if len(m.worldEffects) != 13 || len(m.scheduledSounds) != 0 {
		t.Fatal("each ground unit must own one silent visual")
	}
	if !m.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 169, Value: 166}) || m.worldEffects[len(m.worldEffects)-1].effectID != effectBottomDissonanceGround {
		t.Fatal("overlapping songs must change to Dissonance")
	}
	for unit := uint32(158); unit <= 170; unit++ {
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: unit})
	}
	if len(m.worldEffects) != 0 {
		t.Fatal("song symbols survived removal of their ground units")
	}
}

func TestBardNativeAnimationFrames(t *testing.T) {
	ctx := archerEffectContext()
	start := time.Unix(100, 0)
	action := res.ACTAction{Animations: make([]res.ACTAnimation, 6)}
	for _, tc := range []struct {
		skill  uint16
		frames []int
	}{{db.SkillBaWhistle, []int{1, 2, 3, 1, 2, 3}}, {db.SkillBDRokisweil, []int{1, 2, 3, 2, 1, 2}}, {db.SkillBaFrostjoke, []int{4, 5, 4, 5, 4, 5}}, {db.SkillBaPangvoice, []int{4, 5, 4, 5, 4, 5}}} {
		m := &WorldMode{}
		m.startSkillActionAnimation(ctx, 2000000, ctx.World.Player, skillAction(tc.skill), start, time.Second)
		for i, frame := range tc.frames {
			now := start.Add(time.Duration(i)*10*db.EffectFrameDuration + time.Millisecond)
			anim, _ := m.actorAnimation(2000000, now)
			state := spriteState{}
			applyActorAnimationToSpriteState(&state, anim, true)
			if got := bodyMotionForState(action, state, start, now); got != frame {
				t.Fatalf("skill %d tick %d: motion %d, want %d", tc.skill, i, got, frame)
			}
		}
		if tc.skill == db.SkillBaFrostjoke || tc.skill == db.SkillBaPangvoice {
			anim, _ := m.actorAnimation(2000000, start.Add(1800*time.Millisecond))
			if anim.actionFamily != spriteActionIdle {
				t.Fatal("shout animation must stop after 1800ms")
			}
		} else {
			m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillBDAdaptation, SourceID: 2000000, TargetID: 2000000, Result: 1})
			if m.actorAnims[2000000].actionFamily != spriteActionIdle {
				t.Fatal("Adaptation must return to standing")
			}
		}
	}
	m := &WorldMode{}
	m.startSkillActionAnimation(ctx, 2000000, ctx.World.Player, attackSkillActionSpec, start, time.Second)
	m.stopPerformance(ctx, network.StatusEffectChange{StatusID: db.StatusBdplaying})
	if len(m.actorAnims) != 2 {
		t.Fatal("ending a song cancelled an unrelated attack")
	}
}

func TestBardMusicalStrikeTiming(t *testing.T) {
	ctx := archerEffectContext()
	m := &WorldMode{playerView: humanoidSoundView(spriteActionPCAttack3, 4, 2, []string{"atk"})}
	a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillBaMusicalstrike, SourceSpeed: 500, HitCount: 1, Damage: 100}
	m.applyActorActionNotify(ctx, a)
	anim := m.actorAnims[a.SourceID]
	if anim.actionFamily != spriteActionPCAttack3 {
		t.Fatal("Musical Strike must use ACT 88")
	}
	if len(m.worldEffects) != 2 {
		t.Fatalf("Musical Strike needs one arrow and one impact, got %v", m.worldEffects)
	}
	arrow, hit := m.worldEffects[0], m.worldEffects[1]
	if arrow.effectID != effectArrowShot || hit.effectID != effectHit1 || arrow.starts.Sub(anim.started) != 250*time.Millisecond || hit.starts.Sub(arrow.starts) != referenceBowArrowFlightDuration {
		t.Fatal("arrow must launch at the ACT marker and hit eight ticks later")
	}
	a.Action = 5
	m.applyActorActionNotify(ctx, a)
	if len(m.worldEffects) != 2 {
		t.Fatal("splash notification replayed Musical Strike")
	}
}

func TestBardPangVoiceFacing(t *testing.T) {
	for _, tc := range []struct {
		dir    modelPoint3
		yaw    float64
		action int
	}{
		{modelPoint3{x: 1}, 0, 2}, {modelPoint3{z: 1}, 0, 1},
		{modelPoint3{x: -1}, 0, 0}, {modelPoint3{z: -1}, 0, 3},
		{modelPoint3{x: 1}, 90, 3}, {modelPoint3{x: 1}, -90, 1},
	} {
		if got := directedSpriteAction(tc.dir, tc.yaw); got != tc.action {
			t.Fatalf("direction %+v yaw %v: action %d, want %d", tc.dir, tc.yaw, got, tc.action)
		}
	}
}

func TestBardPangVoiceAndSongAssets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 15, 20, 0)
	for unit := uint16(158); unit <= 170; unit++ {
		m := &WorldMode{}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: unit, X: 15, Y: 20, Visible: true})
		effect := m.worldEffects[0]
		spec, _ := worldEffectSpecForID(effect.effectID)
		for _, file := range spec.components[0].textureFiles {
			if m.effectFileTexture(ctx.Resources, file) == nil {
				t.Fatalf("missing texture %s", file)
			}
		}
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, effect.starts.Add(20*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len()+reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() == 0 {
			t.Fatalf("song unit %d drew nothing", unit)
		}
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if unit == 160 || unit == 165 {
			vertices := commands.Index(0).FieldByName("Vertices")
			for i := 0; i < vertices.Len(); i++ {
				v := vertices.Index(i)
				radius := math.Hypot(v.FieldByName("X").Float()-15.5, v.FieldByName("Z").Float()-20.5)
				if radius < .399 || radius > .801 {
					t.Fatal("song light must rise vertically at its native radius")
				}
			}
		}
		if unit == 166 || unit == 168 {
			vertices := commands.Index(0).FieldByName("Vertices")
			if vertices.Index(0).FieldByName("Y").Float() <= vertices.Index(3).FieldByName("Y").Float() {
				t.Fatal("song ribbon texture is vertically inverted")
			}
		}
	}
	for skill, spec := range db.SkillEffects {
		if isBardPerformance(skill) {
			for _, sound := range spec.SFX {
				if _, err := ctx.Resources.ReadFile("data/wav/" + sound); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	m := &WorldMode{}
	m.addWorldEffectBetweenAt(ctx, effectFVoice, 300, 2000000, start)
	effect := m.worldEffects[0]
	if !effect.hasSpriteFrame || effect.actorOrigin.x != 20.5 || effect.projectileDirection.x != -10 {
		t.Fatal("Pang Voice must capture its target and caster direction")
	}
	frame := render.NewFrame(800, 600)
	m.drawWorldEffects(frame, ctx, projection, start.Add(6*db.EffectFrameDuration), false)
	if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len()+reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() == 0 {
		t.Fatal("Pang Voice drew nothing")
	}
	for initial := 0; initial <= 10; initial++ {
		process, alpha, height := initial, 0, 0.0
		for n := 1; n < 200; n++ {
			process++
			if process <= 10 {
				alpha += 25
			} else {
				alpha -= 5
				if alpha <= 0 {
					alpha, process, height = 0, 0, 0
				}
				height += .04
			}
			a, h := songOutParticle(n, initial)
			if a != alpha || math.Abs(h-height) > 1e-9 {
				t.Fatalf("Roki's veil diverged at initial %d tick %d", initial, n)
			}
		}
	}
}
