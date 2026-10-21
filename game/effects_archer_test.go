package game

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/world"
)

func archerEffectContext() client.Context {
	w := world.New()
	w.Player = world.Actor{ID: 100, Job: db.JobArcher, Weapon: db.WeaponBow, X: 10, Y: 20}
	w.Actors[300] = world.Actor{ID: 300, Job: 1002, X: 20, Y: 20}
	return client.Context{World: w, Session: &session.Session{
		AccountID: 2000000, CharID: 100,
		Selected: session.Character{ID: 100, Job: db.JobArcher, Weapon: db.WeaponBow},
	}}
}

func TestArcherConcentrationAndUtility(t *testing.T) {
	ctx := archerEffectContext()
	m := &WorldMode{}
	notify := network.SkillNoDamageNotify{SkillID: db.SkillACConcentration, SourceID: 2000000, TargetID: 300}
	m.applySkillNoDamageNotify(ctx, notify)
	if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
		t.Fatal("failed Concentration must not produce effects or sound")
	}
	notify.Result = 1
	m.applySkillNoDamageNotify(ctx, notify)
	if len(m.worldEffects) != 1 || len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\ac_concentration.wav" {
		t.Fatal("Concentration needs only its STR and native sound")
	}
	if e := m.worldEffects[0]; e.effectID != effectConcentration || e.actorID != notify.SourceID {
		t.Fatalf("Concentration effect must belong to the caster: %+v", e)
	}
	for _, skill := range []uint16{db.SkillACOwl, db.SkillACVulture, db.SkillACMakingarrow} {
		m := &WorldMode{}
		notify.SkillID = skill
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatalf("utility skill %d must not invent a visual effect", skill)
		}
	}
}

func TestArcherConcentrationCast(t *testing.T) {
	ctx := archerEffectContext()
	m := &WorldMode{}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{
		SkillID: db.SkillACConcentration, SourceID: 2000000, TargetID: 2000000, DelayTime: 4000, Property: 3,
	})
	if len(m.worldEffects) != 2 || m.worldEffects[1].effectID != effectIncAgiDex || m.worldEffects[1].duration != 4*time.Second || len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\ef_incagidex.wav" {
		t.Fatal("Concentration must use its AGI/DEX casting label and sound, regardless of element")
	}
	label := agilityLabel(false, 119, 240)
	if math.Abs(label.y+125.55) > .001 || label.alpha != 200 {
		t.Fatal("Concentration's label must follow the cast duration instead of ending after 100 ticks")
	}
	ctx.Resources = realDataManager(t)
	frame := render.NewFrame(800, 600)
	effect := m.worldEffects[1]
	spec, _ := worldEffectSpecForID(effectIncAgiDex)
	projection := newSceneProjectionForTarget(800, 600, 10, 20, 0)
	m.drawAgilityEffect(frame, ctx, projection, spec.components[0], effect, 10.5, 20.5, 0, effect.starts.Add(120*db.EffectFrameDuration))
	if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() == 0 {
		t.Fatal("Concentration's rising streaks must remain visible during a longer cast")
	}
}

func TestArcherArrowShowerGroundAndDamagePackets(t *testing.T) {
	ctx := archerEffectContext()
	family := attackActionFamilyForActorWithResources(nil, ctx.World.Player)
	m := &WorldMode{playerView: humanoidSoundView(family, 4, 2, []string{"atk"})}
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: db.SkillACShower, X: 20, Y: 20})
	if len(m.worldEffects) != 1 || m.worldEffects[0].arrowTargets == nil {
		t.Fatal("Arrow Shower must fire on empty ground")
	}
	ground := m.worldEffects[0]
	if ground.starts.Sub(m.actorAnims[2000000].started) != arrowShowerReleaseDelay {
		t.Fatal("ground arrows must wait 18 native 24 ms ticks")
	}
	if len(m.scheduledSounds) != 1 || !sameStringSet(m.scheduledSounds[0].paths, db.WeaponAttackSounds(db.WeaponBow)) {
		t.Fatal("Arrow Shower must play the bow release even on empty ground")
	}
	if target := ground.arrowTargets[0]; target.x != 10.5 || target.z != 20.5 {
		t.Fatalf("the ground packet's first arrow must follow the native caster endpoint: %+v", target)
	}
	for _, target := range ground.arrowTargets[1:] {
		if math.Abs(target.x-20.5) > 1 || math.Abs(target.z-20.5) > 1 || target.x == 20.5 && target.z == 20.5 {
			t.Fatalf("arrow outside the eight neighbouring cells: %+v", target)
		}
	}
	// The original explicitly processes Arrow Shower's ACTION_SPLASH as a
	// full skill, including arrows and impacts for each affected target.
	m.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID: 2000000, TargetID: 300, SkillID: db.SkillACShower,
		Action: 5, Damage: 100, HitCount: 1, SourceSpeed: 500,
	})
	if len(m.worldEffects) != 5 {
		t.Fatalf("expected ground spread, bow arrow, victim spread, and both impacts; got %d", len(m.worldEffects))
	}
	spread := m.worldEffects[2]
	if spread.arrowTargets == nil || spread.arrowTargets[0].x != 20.5 || spread.arrowTargets[0].z != 20.5 {
		t.Fatal("damage notification must aim the spread's first arrow at its victim")
	}
	for i, id := range []int{effectHit1, effectBashHit} {
		if e := m.worldEffects[3+i]; e.effectID != id || e.actorID != 300 || !e.starts.Equal(m.worldEffects[1].expires) {
			t.Fatalf("wrong Arrow Shower impact: %+v", e)
		}
	}
}

func TestArcherArrowShowerReplacesPendingReleaseSound(t *testing.T) {
	ctx := archerEffectContext()
	family := attackActionFamilyForActorWithResources(nil, ctx.World.Player)
	m := &WorldMode{playerView: humanoidSoundView(family, 4, 2, []string{"atk"})}
	ground := network.GroundSkillNotify{SourceID: 2000000, SkillID: db.SkillACShower, X: 20, Y: 20}
	m.applyGroundSkillNotify(ctx, ground)
	// Other casters and independent effects must survive replacement.
	otherRelease := time.Now().Add(time.Second)
	m.scheduleSkillActionSoundAtActor(otherRelease, 400, db.SkillACShower, db.WeaponAttackSounds(db.WeaponBow)...)
	m.scheduleSoundAtActor(otherRelease, ground.SourceID, "effect\\ac_concentration.wav")
	for _, id := range []uint32{300, 301, 302} {
		ctx.World.Actors[id] = world.Actor{ID: id, Job: 1002, X: 20, Y: 20}
		m.applyActorActionNotify(ctx, network.ActorActionNotify{
			SourceID: ground.SourceID, TargetID: id, SkillID: ground.SkillID,
			Action: 5, Damage: 100, HitCount: 1, SourceSpeed: 200,
		})
	}
	releases, hits, others := 0, 0, 0
	for _, sound := range m.scheduledSounds {
		switch {
		case sound.actorID == ground.SourceID && sameStringSet(sound.paths, db.WeaponAttackSounds(db.WeaponBow)):
			releases++
			if want := m.actorAnims[ground.SourceID].started.Add(100 * time.Millisecond); !sound.at.Equal(want) {
				t.Fatalf("release at %s, want latest animation's marker at %s", sound.at, want)
			}
		case sound.actorID >= 300 && sound.actorID <= 302:
			hits++
		case sound.at.Equal(otherRelease):
			others++
		}
	}
	// Each victim keeps the normal hit sound and EF_HIT2's sound.
	if releases != 1 || hits != 6 || others != 2 {
		t.Fatalf("sounds: releases=%d hits=%d unrelated=%d, want 1, 6, 2", releases, hits, others)
	}
	// Once the release has played, a new cast must still produce its sound.
	m.playDueScheduledSounds(ctx, otherRelease.Add(time.Second))
	m.applyGroundSkillNotify(ctx, ground)
	if len(m.scheduledSounds) != 1 || !sameStringSet(m.scheduledSounds[0].paths, db.WeaponAttackSounds(db.WeaponBow)) {
		t.Fatal("a later Arrow Shower must have its own release sound")
	}
}

func TestArcherArrowShowerSpreadAroundMovingTarget(t *testing.T) {
	ctx := archerEffectContext()
	now := time.Unix(100, 0)
	target := ctx.World.Actors[300]
	target.Moving = true
	target.FromX, target.FromY, target.ToX, target.ToY = 20, 20, 21, 19
	target.MoveStarted, target.MoveDuration = now.Add(-750*time.Millisecond), time.Second
	ctx.World.Actors[300] = target
	m := &WorldMode{}
	m.addSkillEffect(ctx, network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillACShower}, now, now)
	arrows := m.worldEffects[0].arrowTargets
	if arrows[0].x != 21.25 || arrows[0].z != 19.75 {
		t.Fatalf("center arrow must aim at the interpolated position: %+v", arrows[0])
	}
	var centerX, centerZ float64
	for _, arrow := range arrows[1:] {
		centerX += arrow.x / 8
		centerZ += arrow.z / 8
	}
	if centerX != 21.5 || centerZ != 19.5 {
		t.Fatalf("spread center = %g,%g, want center of containing cell 21,19", centerX, centerZ)
	}
}

func TestArcherArrowFlightAndSplash(t *testing.T) {
	origin, target := modelPoint3{}, modelPoint3{x: 4}
	if p, ok := arrowShowerPosition(origin, target, 48*time.Millisecond); !ok || math.Abs(p.x-2) > 1e-9 {
		t.Fatalf("nearby arrow must travel at the native minimum speed: %+v, %t", p, ok)
	}
	for _, elapsed := range []time.Duration{-1, 96 * time.Millisecond} {
		if _, ok := arrowShowerPosition(origin, target, elapsed); ok {
			t.Fatal("arrow visible before release or after arrival")
		}
	}
	ctx := archerEffectContext()
	for _, skill := range []uint16{db.SkillACDouble, db.SkillACChargearrow} {
		m := &WorldMode{}
		m.applyActorActionNotify(ctx, network.ActorActionNotify{
			SourceID: 2000000, TargetID: 300, SkillID: skill, Action: 5, Damage: 100,
		})
		if len(m.worldEffects) != 0 {
			t.Fatalf("skill %d secondary splash must not replay the skill: %+v", skill, m.worldEffects)
		}
	}
	for _, damage := range []int32{0, 100} {
		m := &WorldMode{}
		m.applyActorActionNotify(ctx, network.ActorActionNotify{
			SourceID: 2000000, TargetID: 300, SkillID: db.SkillACChargearrow,
			Action: network.ActorActionSkill, Damage: damage,
		})
		want := 1
		if damage > 0 {
			want++
		}
		if len(m.worldEffects) != want || m.worldEffects[0].effectID != effectBashBegin || m.actorAnims[2000000].actionFamily != spriteActionPCSkill {
			t.Fatal("Arrow Repel must use the native skill pose and flash, with no bow projectile")
		}
	}
}

func TestArcherEffectsWithClientAssets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	m := &WorldMode{}
	start := time.Unix(100, 0)
	m.addArrowShower(ctx, 2000000, 20, 20, false, start)
	effect := m.worldEffects[0]
	spec, _ := worldEffectSpecForID(effectArrowShower)
	projection := newSceneProjectionForTarget(800, 600, 15, 20, 0)
	for _, tc := range []struct {
		delay time.Duration
		count int
	}{{-1, 0}, {48 * time.Millisecond, 9}, {192 * time.Millisecond, 0}} {
		frame := render.NewFrame(800, 600)
		m.drawArrowShower(frame, ctx, projection, spec.components[0], effect, effect.starts.Add(tc.delay))
		if got := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); got != tc.count {
			t.Fatalf("delay %s: drew %d arrows, want %d", tc.delay, got, tc.count)
		}
	}
	for _, id := range []int{effectIncAgiDex, effectConcentration} {
		if !m.addWorldEffectAt(ctx, id, 2000000, start) {
			t.Fatalf("could not add effect %d", id)
		}
	}
	for _, sound := range m.scheduledSounds {
		for _, path := range sound.paths {
			if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
				t.Error(err)
			}
		}
	}
	frame := render.NewFrame(800, 600)
	m.drawWorldEffects(frame, ctx, projection, start.Add(30*db.EffectFrameDuration), false)
	if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() == 0 {
		t.Fatal("Concentration's rising streaks did not render")
	}
}
