package game

import (
	"image/color"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
)

func TestStarGladiatorWarmthStatusAndSkillPackets(t *testing.T) {
	for _, status := range []uint16{db.StatusSgWarm, db.StatusSgSunWarm, db.StatusSgMoonWarm, db.StatusSgStarWarm} {
		ctx := archerEffectContext()
		ctx.World.Player.Job = db.JobStar
		m := &WorldMode{}
		m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: status, Active: true})
		if len(m.worldEffects) != 2 || len(m.scheduledSounds) != 0 {
			t.Fatal("Warmth status needs persistent lightning and body glow, without recasting particles or sound")
		}
		for _, e := range m.worldEffects {
			if !e.persistent || !e.expires.IsZero() || e.actorID != 2000000 {
				t.Fatalf("Warmth status effect lifetime or local alias: %+v", e)
			}
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 428, SourceID: 100, TargetID: 100, Result: 1})
		if len(m.worldEffects) != 3 || m.worldEffects[2].effectID != effectHated2 || m.worldEffects[2].buffParticles == nil || len(m.scheduledSounds) != 1 || m.scheduledSounds[0].at.Sub(m.worldEffects[2].starts) != 10*db.EffectFrameDuration {
			t.Fatal("Warmth release must restart its state effects, add red particles, and delay the sound ten ticks")
		}
		anim := m.actorAnims[100]
		if anim.actionFamily != spriteActionPCSkill || anim.fixedMotion != 3 || !anim.hasFixedMotion || anim.duration != time.Second {
			t.Fatalf("Warmth release pose: %+v", anim)
		}
		m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: status, Active: false})
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectHated2 {
			t.Fatal("status removal must stop lightning and red glow while letting release particles finish")
		}
	}
}

func TestStarGladiatorNoDamageOwnershipAndSounds(t *testing.T) {
	for _, tc := range []struct {
		skill  uint16
		ids    []int
		actor  uint32
		sounds int
	}{
		{427, nil, 0, 0},
		{431, []int{488, 543}, 2000000, 1},
		{432, []int{488, 543}, 2000000, 1},
		{433, []int{488, 543}, 2000000, 1},
		{434, []int{543}, 300, 1},
		{444, []int{effectQuake}, 2000000, 1},
	} {
		ctx := archerEffectContext()
		ctx.World.Player.Job = db.JobStar
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Result: 1}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.worldEffects) != len(tc.ids) || len(m.scheduledSounds) != tc.sounds {
			t.Fatalf("skill %d: effects=%+v sounds=%+v", tc.skill, m.worldEffects, m.scheduledSounds)
		}
		for i, e := range m.worldEffects {
			if e.effectID != tc.ids[i] || e.actorID != tc.actor {
				t.Fatalf("skill %d effect ownership: %+v", tc.skill, e)
			}
		}
		if tc.skill >= 431 && tc.skill <= 433 {
			anim := m.actorAnims[100]
			if anim.fixedMotion != 2 || !anim.hasFixedMotion || anim.duration != time.Second {
				t.Fatal("Comfort must hold skill motion two for one second")
			}
			if m.scheduledSounds[0].at.Sub(m.worldEffects[0].starts) != 5*db.EffectFrameDuration {
				t.Fatal("Comfort's sound belongs to effect tick five")
			}
		}
		m.worldEffects, m.scheduledSounds = nil, nil
		m.addSkillEffect(ctx, network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}, time.Now(), time.Now())
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatal("damage notifications must not replay Star Gladiator's no-damage effects")
		}
	}
}

func TestStarDevilReconcilesLevelStatusAndMapEntry(t *testing.T) {
	ctx := archerEffectContext()
	m := &WorldMode{}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusDevil1, Active: true})
	if len(m.worldEffects) != 0 {
		t.Fatal("Demon must wait for its learned level")
	}
	ctx.Session.Skills.List = []session.Skill{{ID: db.SkillSGDevil, Level: 10}}
	m.syncStarDevil(ctx)
	m.syncStarDevil(ctx)
	if len(m.worldEffects) != 2 || m.worldEffects[0].effectID != 484 || !m.worldEffects[0].persistent || len(m.scheduledSounds) != 1 {
		t.Fatal("Demon level ten must install a single persistent mask and introduction")
	}
	next := &WorldMode{}
	next.syncCurrentActorEffectStateEffects(ctx)
	if len(next.worldEffects) != 2 || next.worldEffects[0].effectID != 484 {
		t.Fatal("map entry did not restore Demon")
	}
	ctx.Session.Skills.List[0].Level = 1
	m.syncStarDevil(ctx)
	for _, e := range m.worldEffects {
		if e.effectID == 484 {
			t.Fatal("relearned Demon kept the old radius")
		}
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusDevil1, Active: false})
	if len(m.worldEffects) != 0 {
		t.Fatal("Demon removal left its mask or rings")
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusDevil1, ActorID: 300, Active: true})
	if len(m.worldEffects) != 0 {
		t.Fatal("another actor's Demon blinded the player")
	}
}

func TestStarFusionFlightUsesServerOptionAndElapsedTicks(t *testing.T) {
	ctx := archerEffectContext()
	actor := ctx.World.Player
	actor.Job, actor.EffectState = db.JobStar, db.EffectStateFlying
	if visualJobForEffectState(db.JobStar2, 0) != db.JobStar || visualJobForEffectState(db.JobStar2B, 0) != db.JobStarB {
		t.Fatal("server flying class IDs must normalize when the option ends")
	}
	if actorVisualJob(actor) != db.JobStar2 {
		t.Fatal("Fusion did not select its flying ACT")
	}
	if visualJobForEffectState(db.JobStarB, db.EffectStateFlying) != db.JobStar2B {
		t.Fatal("baby Fusion visual job")
	}
	start := time.Unix(100, 0)
	for _, step := range []int{1, 15, 300} {
		m := &WorldMode{}
		m.starFusionHeight(actor, start)
		for tick := step; tick <= 300; tick += step {
			m.starFusionHeight(actor, start.Add(time.Duration(tick)*db.EffectFrameDuration))
		}
		if m.starFlights[actor.ID].height != 20 {
			t.Fatal("Fusion flight height depends on render rate")
		}
		actor.EffectState = 0
		if actorVisualJob(actor) != db.JobStar {
			t.Fatal("cleared Fusion retained its flying job")
		}
		m.starFusionHeight(actor, start.Add(320*db.EffectFrameDuration))
		if len(m.starFlights) != 0 {
			t.Fatal("Fusion did not land and release its render state")
		}
		actor.EffectState = db.EffectStateFlying
	}
	m := &WorldMode{}
	actor.Sitting = true
	m.starFusionHeight(actor, start)
	m.starFusionHeight(actor, start.Add(1500*db.EffectFrameDuration))
	if m.starFlights[actor.ID].height != 100 {
		t.Fatal("sitting Fusion must rise higher")
	}
	actor.Opt3State = db.Opt3Warm
	m.starFusionHeight(actor, start.Add(2700*db.EffectFrameDuration))
	if m.starFlights[actor.ID].height != 20 {
		t.Fatal("Warmth must keep Fusion at its ordinary height")
	}
}

func TestStarGladiatorNativeAssetsAndGeometry(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	start := time.Now()
	projection := newSceneProjectionForTarget(800, 600, 10, 20, 0)
	for _, tc := range []struct{ id, tick, end int }{{418, 50, -1}, {572, 20, 301}, {488, 90, 205}, {487, 20, 151}, {475, 30, -1}, {484, 30, -1}} {
		m := &WorldMode{}
		m.addWorldEffectAt(ctx, tc.id, 2000000, start)
		for _, tick := range []int{tc.tick, tc.end} {
			if tick < 0 {
				continue
			}
			frame := render.NewFrame(800, 600)
			m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			v := reflect.ValueOf(frame).Elem()
			commands := v.FieldByName("worldCommands")
			count := commands.Len() + v.FieldByName("worldBillboards").Len()
			if (count > 0) != (tick == tc.tick) {
				t.Fatalf("effect %d tick %d draws=%d", tc.id, tick, count)
			}
			if tc.id == 488 && tick == 90 {
				mesh := commands.Index(0).FieldByName("Vertices")
				if mesh.Len() != 8*21*4 {
					t.Fatalf("Comfort must have eight 315-degree rings: %d vertices", mesh.Len())
				}
			}
		}
		for _, sound := range m.scheduledSounds {
			for _, path := range sound.paths {
				if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	for _, path := range []string{"T_변신.wav", "T_등록.wav", "_blind.wav"} {
		if _, err := ctx.Resources.ReadFile("data/wav/effect/" + path); err != nil {
			t.Fatal(err)
		}
	}
	ctx.Session.Selected.Job = db.JobStar2
	for _, sex := range []uint8{0, 1} {
		if view, message := loadPlayerHumanoidSpriteView(ctx.Resources, ctx.Session.Selected, sex, false); view == nil {
			t.Fatal(message)
		}
	}
}

func TestStarGladiatorBodyOverlays(t *testing.T) {
	ctx := archerEffectContext()
	start := time.Unix(100, 0)
	billboard := &spriteBillboard{image: render.NewImage(40, 80), anchorX: 20, anchorY: 80}
	entry := sceneActorDrawEntry{actor: ctx.World.Player, worldX: 10.5, worldY: 20.5}
	projection := newSceneProjectionForTarget(800, 600, 10, 20, 0)
	for _, id := range []int{544, 488} {
		m := &WorldMode{}
		m.addWorldEffectAt(ctx, id, 2000000, start)
		frame := render.NewFrame(800, 600)
		m.drawActorSpriteWithEffects3D(ctx, frame, projection, entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(200*db.EffectFrameDuration))
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() != 3 {
			t.Fatal("native body effects need two additive copies over the ordinary sprite")
		}
		wantR, wantB := 1.0, 5.0/255
		if id == 488 {
			wantR, wantB = 55.0/255, 1
		}
		for i := 1; i < 3; i++ {
			if math.Abs(draws.Index(i).FieldByName("ColorR").Float()-wantR) > .00001 || math.Abs(draws.Index(i).FieldByName("ColorB").Float()-wantB) > .00001 {
				t.Fatal("body flash palette differs from native")
			}
		}
	}
}

func TestStarGladiatorCastsAndRejectedSkills(t *testing.T) {
	ctx := archerEffectContext()
	ctx.World.Player.Job = db.JobStar
	for _, skill := range []uint16{427, 434, 444} {
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, DelayTime: 1000})
		found := false
		for _, e := range m.worldEffects {
			found = found || e.effectID == effectBlueCasting
		}
		if !found {
			t.Fatalf("skill %d needs BlueCasting", skill)
		}
	}
	for skill := uint16(427); skill <= 444; skill++ {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300})
		if len(m.worldEffects)+len(m.actorAnims)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed skill created release effects")
		}
	}
	m := &WorldMode{}
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 428, SourceID: 2000000, TargetID: 999, Result: 1})
	if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
		t.Fatal("Warmth requires both actors, like the native packet handler")
	}
	m.addWorldEffect(ctx, effectHated, 300)
	if len(m.scheduledSounds) != 0 {
		t.Fatal("the shared Hated effect must not play Kaahi's sound")
	}
}
