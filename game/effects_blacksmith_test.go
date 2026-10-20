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

func TestBlacksmithSkillPacketOwnership(t *testing.T) {
	ctx := merchantEffectContext()
	ctx.World.Player.Job = db.JobBlacksmith
	ctx.Session.Selected.Job = db.JobBlacksmith
	ctx.Session.Selected.ID = 100
	for _, tc := range []struct {
		skill  uint16
		effect int
		actor  uint32
		sounds int
	}{
		{db.SkillBSAdrenaline, effectHasteUp, 2000000, 2},
		{db.SkillBSAdrenaline2, effectHasteUp, 2000000, 2},
		{db.SkillBSWeaponperfect, effectWeaponPerfect, 300, 1},
		{db.SkillBSRepairweapon, effectRepairWeapon, 300, 2},
		{db.SkillBSOverthrust, effectOverthrust, 2000000, 2},
		{db.SkillBSMaximize, effectMaximizePower, 2000000, 4},
		{db.SkillBSGreed, effectGreedSound, 2000000, 1},
	} {
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed skill must be silent")
		}
		n.Result = 1
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.effect || m.worldEffects[0].actorID != tc.actor || len(m.scheduledSounds) != tc.sounds {
			t.Fatalf("skill %d has incorrect effect, owner, or sounds", tc.skill)
		}
		if tc.skill == db.SkillBSOverthrust {
			m.applySkillNoDamageNotify(ctx, n)
			if len(m.worldEffects) != 1 {
				t.Fatal("Overthrust refresh must replace the previous ring")
			}
		}
	}
	for _, skill := range []uint16{db.SkillBSIron, db.SkillBSSteel, db.SkillBSEnchantedstone, db.SkillBSOrideocon,
		db.SkillBSDagger, db.SkillBSSword, db.SkillBSTwohandsword, db.SkillBSAxe, db.SkillBSMace, db.SkillBSKnuckle,
		db.SkillBSSpear, db.SkillBSHiltbinding, db.SkillBSFindingore, db.SkillBSWeaponresearch, db.SkillBSSkintemper, db.SkillBSUnfairlytrick} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Result: 1})
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatalf("passive/forging skill %d must not invent an activation effect", skill)
		}
	}
}

func TestBlacksmithHammerFallAndCastPoses(t *testing.T) {
	ctx := merchantEffectContext()
	ctx.World.Player.Job, ctx.World.Player.Weapon = db.JobBlacksmith, db.WeaponAxe
	ctx.Session.Selected.Job, ctx.Session.Selected.Weapon = db.JobBlacksmith, db.WeaponAxe
	ctx.Session.Selected.ID = 100
	start := time.Unix(100, 0)
	m := &WorldMode{}
	m.startSkillSourceCastAnimation(ctx, 2000000, db.SkillBSHammerfall, time.Second, start)
	if m.actorAnims[2000000].actionFamily != attackActionFamilyForActorWithResources(ctx.Resources, ctx.World.Player) {
		t.Fatal("Hammer Fall must cast with the equipped weapon's attack pose")
	}
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillBSHammerfall, SourceID: 2000000, X: 15, Y: 20})
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillBSHammerfall, SourceID: 2000000, X: 15, Y: 20})
	if len(m.worldEffects) != 2 || m.actorAnims[2000000].actionFamily != spriteActionPCSkill {
		t.Fatal("each Hammer Fall must play its release pose and create a fresh ground impact")
	}
	for _, e := range m.worldEffects {
		if e.effectID != effectCrashEarth || e.actorID != 0 || e.x != 15 || e.y != 20 {
			t.Fatal("Hammer Fall must stay at its target cell")
		}
	}
	for _, tc := range []struct {
		skill uint16
		aura  int
	}{
		{db.SkillBSHammerfall, effectBashBegin}, {db.SkillBSAdrenaline, effectBashBegin},
		{db.SkillBSMaximize, effectBashBegin}, {db.SkillBSRepairweapon, effectBeginSpell},
	} {
		m := &WorldMode{}
		m.addSkillCastEffects(ctx, tc.skill, 3, 2000000, 300, 0, 0, time.Second, start, "test")
		if len(m.worldEffects) != 3 || m.worldEffects[2].effectID != tc.aura {
			t.Fatalf("skill %d has the wrong casting aura", tc.skill)
		}
		for _, s := range m.scheduledSounds {
			if s.paths[0] == "effect\\black_adrenalinerush_a.wav" || s.paths[0] == "effect\\black_maximize_power_circle.wav" {
				t.Fatal("buff audio must wait for the success packet")
			}
		}
	}
}

func TestBlacksmithEffectsWithClientAssets(t *testing.T) {
	ctx := merchantEffectContext()
	ctx.Resources = realDataManager(t)
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 10, 20, 0)
	for _, tc := range []struct {
		id    int
		ticks []int
	}{
		{effectHasteUp, []int{0, 30}}, {effectRepairWeapon, []int{20, 55}},
		{effectCrashEarth, []int{30}}, {effectWeaponPerfect, []int{0}},
		{effectMaximizePower, []int{0, 40, 47, 65}}, {effectOverthrust, []int{0, 0}},
	} {
		m := &WorldMode{}
		m.addWorldEffectAt(ctx, tc.id, 2000000, start)
		if len(m.scheduledSounds) != len(tc.ticks) {
			t.Fatalf("effect %d: incorrect sound count", tc.id)
		}
		for i, sound := range m.scheduledSounds {
			if sound.at.Sub(start) != time.Duration(tc.ticks[i])*db.EffectFrameDuration {
				t.Fatalf("effect %d sound %d has incorrect timing", tc.id, i)
			}
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound.paths[0]); err != nil {
				t.Fatal(err)
			}
		}
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(20*db.EffectFrameDuration), false)
		value := reflect.ValueOf(frame).Elem()
		if value.FieldByName("commands").Len()+value.FieldByName("worldCommands").Len()+value.FieldByName("worldBillboards").Len() == 0 {
			t.Fatalf("effect %d drew nothing with client assets", tc.id)
		}
		if tc.id == effectHasteUp {
			if got := value.FieldByName("worldBillboards").Len(); got != 4 {
				t.Fatalf("Adrenaline Rush needs four orbiting particles, got %d", got)
			}
			center := value.FieldByName("worldBillboards").Index(0).FieldByName("Center")
			if math.Abs(center.Index(1).Float()-(.07+(2.1+.462)/5)) > .001 {
				t.Fatal("Adrenaline particles have the wrong rising trajectory")
			}
		}
		if tc.id == effectCrashEarth {
			spec, _ := worldEffectSpecForID(tc.id)
			if spec.cameraShakeDelay != 35*db.EffectFrameDuration || m.scheduledSounds[0].paths[0] != "effect\\wizard_fire_pillar_b.wav" {
				t.Fatal("Hammer Fall must use the native impact sound and quake tick")
			}
		}
	}
}
