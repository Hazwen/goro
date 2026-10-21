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
)

func TestSniperCastAndBuffOwnership(t *testing.T) {
	ctx := archerEffectContext()
	start := time.Now()
	for _, tc := range []struct {
		skill uint16
		aura  int
	}{{db.SkillSNSharpshooting, effectBeginSpell6}, {db.SkillSNWindwalk, effectBeginSpell6}, {db.SkillSNFalconassault, effectBashBegin}} {
		m := &WorldMode{}
		m.startSkillSourceCastAnimation(ctx, 2000000, tc.skill, time.Second, start)
		m.addSkillCastEffects(ctx, tc.skill, 5, 2000000, 300, 0, 0, time.Second, start, "test")
		if len(m.worldEffects) != 3 || m.worldEffects[2].effectID != tc.aura {
			t.Fatalf("skill %d needs cast circle, lock-on, and native aura", tc.skill)
		}
		if tc.skill == db.SkillSNSharpshooting {
			anim := m.actorAnims[2000000]
			if anim.actionFamily*8 != 88 || !anim.hasFixedMotion || anim.fixedMotion != 0 || anim.play {
				t.Fatal("Sharp Shooting must hold ready-shot frame zero while casting")
			}
		}
	}
	for _, tc := range []struct {
		skill  uint16
		effect int
	}{{db.SkillSNSight, effectTrueSight}, {db.SkillSNWindwalk, effectPortal4}} {
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.worldEffects) != 0 {
			t.Fatal("failed buff produced effects")
		}
		n.Result = 1
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.effect || m.worldEffects[0].actorID != n.SourceID {
			t.Fatalf("skill %d must launch once on its packet source", tc.skill)
		}
		if tc.skill == db.SkillSNSight && m.actorAnims[n.SourceID].actionFamily != spriteActionIdle {
			t.Fatal("True Sight should leave the caster standing")
		}
		if tc.skill == db.SkillSNWindwalk && (!m.worldEffects[0].hasActorTransform || m.worldEffects[0].actorOrigin.x != 10.5) {
			t.Fatal("Wind Walk must capture the caster's position on activation")
		}
	}
}

func TestSniperDamageTimingAndSplash(t *testing.T) {
	ctx := archerEffectContext()
	family := attackActionFamilyForActorWithResources(nil, ctx.World.Player)
	for _, skill := range []uint16{db.SkillSNSharpshooting, db.SkillSNFalconassault} {
		m := &WorldMode{playerView: humanoidSoundView(family, 4, 2, []string{"atk"})}
		a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: skill, SourceSpeed: 500, HitCount: 5, Damage: 100}
		m.applyActorActionNotify(ctx, a)
		start := m.actorAnims[a.SourceID].started
		counts := map[int]int{}
		var arrow worldEffect
		for _, e := range m.worldEffects {
			counts[e.effectID]++
			switch e.effectID {
			case effectTripleAttack2:
				if e.actorID != a.SourceID || e.targetID != a.TargetID || !e.starts.Equal(start) || e.projectileDirection.x != 10 {
					t.Fatal("Sharp Shooting rays must start at the caster and aim through the victim")
				}
			case effectFalconAssault:
				if e.actorID != a.TargetID || !e.starts.Equal(start) || e.expires.Sub(start) != 200*db.EffectFrameDuration {
					t.Fatal("Falcon Assault's rotation and quake must start on the victim at release")
				}
			case effectArrowShot:
				arrow = e
				if e.starts.Sub(start) != 250*time.Millisecond || e.expires.Sub(e.starts) != referenceBowArrowFlightDuration {
					t.Fatal("Sharp Shooting's arrow must use the bow ACT marker and eight-tick flight")
				}
			}
		}
		if skill == db.SkillSNSharpshooting {
			if !reflect.DeepEqual(counts, map[int]int{effectTripleAttack2: 1, effectArrowShot: 1, effectHit1: 5, effectBashHit: 1}) {
				t.Fatalf("incorrect Sharp Shooting effects: %v", counts)
			}
			for _, e := range m.worldEffects {
				if e.effectID == effectBashHit && !e.starts.Equal(arrow.expires) {
					t.Fatal("secondary impact must coincide with arrow arrival")
				}
			}
		} else if !reflect.DeepEqual(counts, map[int]int{effectFalconAssault: 1, effectBlitzBeat: 1, effectHit1: 5}) {
			t.Fatalf("incorrect Falcon Assault effects: %v", counts)
		}
		before := len(m.worldEffects)
		a.Action = 5
		m.applyActorActionNotify(ctx, a)
		if len(m.worldEffects) != before {
			t.Fatal("secondary splash packet replayed the skill")
		}
	}
}

func TestSniperBodyFlashAndRotation(t *testing.T) {
	ctx := archerEffectContext()
	start := time.Unix(100, 0)
	for _, tc := range []struct {
		id, frame int
		flash     bool
	}{
		{effectFalconAssault, 29, false}, {effectFalconAssault, 30, true},
		{effectFalconAssault, 35, false}, {effectFalconAssault, 40, true},
		{effectFalconAssault, 50, true}, {effectFalconAssault, 51, false},
		{effectPortal4, 4, false}, {effectPortal4, 5, true},
		{effectPortal4, 25, true}, {effectPortal4, 26, false},
	} {
		m := &WorldMode{}
		m.addWorldEffectAt(ctx, tc.id, 300, start)
		now := start.Add(time.Duration(tc.frame) * db.EffectFrameDuration)
		if (m.actorRenderBlend(300, now) == render.BlendLighter) != tc.flash {
			t.Fatalf("effect %d frame %d: incorrect body blend", tc.id, tc.frame)
		}
		if tc.id == effectPortal4 && tc.flash && m.actorBodyColorTint(300, color.RGBA{255, 255, 255, 255}, now) != (color.RGBA{220, 250, 220, 255}) {
			t.Fatal("Wind Walk's body flash must be pale green")
		}
	}
	m := &WorldMode{}
	m.addWorldEffectAt(ctx, effectFalconAssault, 2000000, start)
	for _, tc := range []struct{ frame, direction int }{{29, 0}, {30, 1}, {31, 1}, {32, 2}, {41, 0}, {54, 1}, {55, 0}} {
		if d := m.skillEffectDirection(ctx, 100, 0, start.Add(time.Duration(tc.frame)*db.EffectFrameDuration)); d != tc.direction {
			t.Fatalf("frame %d: direction %d, want %d (including local account/character alias)", tc.frame, d, tc.direction)
		}
	}
}

func TestSniperEffectsWithClientAssets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 15, 20, 0)
	for _, id := range []int{effectTrueSight, effectPortal4, effectTripleAttack2} {
		m := &WorldMode{}
		m.addWorldEffectBetweenAt(ctx, id, 300, 2000000, start)
		for _, sound := range m.scheduledSounds {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound.paths[0]); err != nil {
				t.Fatal(err)
			}
			if id == effectTripleAttack2 && sound.at.Sub(start) != 30*db.EffectFrameDuration {
				t.Fatal("Sharp Shooting sound should wait 30 ticks")
			}
		}
		at := 10
		if id == effectTripleAttack2 {
			at = 60
		}
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(at)*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() == 0 {
			t.Fatalf("effect %d drew nothing", id)
		}
		frame = render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(90*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() != 0 {
			t.Fatalf("effect %d still drew after particles faded", id)
		}
	}
	effect := worldEffect{actorOrigin: modelPoint3{x: 10}, projectileDirection: modelPoint3{z: 8}, starts: start}
	for i := 0; i < 16; i++ {
		from, to, _ := sharpShootingLine(effect, i, 60)
		if math.Abs(to.x-from.x) > 1e-6 || to.z-from.z < 8 || from.y < 1.6 || from.y > 2.4 {
			t.Fatal("Sharp Shooting rays have incorrect direction, length, or height")
		}
		_, _, alpha := sharpShootingLine(effect, i, 40)
		if alpha != 0 {
			t.Fatal("Sharp Shooting ray appeared before native delay")
		}
	}
}
