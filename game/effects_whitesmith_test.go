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

func TestWhitesmithBuffsAndCasting(t *testing.T) {
	ctx := archerEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobBlacksmithH, db.JobBlacksmithH
	for _, tc := range []struct {
		skill                  uint16
		effect, action, sounds int
	}{
		{db.SkillWSMeltdown, effectMeltdown, spriteActionIdle, 1},
		{db.SkillWSCartboost, effectCartBoost, spriteActionIdle, 1},
		{db.SkillWSOverthrustmax, effectOverthrust, spriteActionPCSkill, 2},
	} {
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed buff must not play effects or sounds")
		}
		n.Result = 1
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.effect || m.worldEffects[0].actorID != n.SourceID || len(m.scheduledSounds) != tc.sounds {
			t.Fatalf("skill %d must play once on its caster", tc.skill)
		}
		if m.actorAnims[n.SourceID].actionFamily != tc.action {
			t.Fatalf("skill %d has the wrong release pose", tc.skill)
		}
		if tc.skill == db.SkillWSOverthrustmax {
			m.applySkillNoDamageNotify(ctx, n)
			if len(m.worldEffects) != 1 {
				t.Fatal("Maximum Power Thrust must replace its previous ring")
			}
		}
	}
	for _, skill := range []uint16{db.SkillWSMeltdown, db.SkillWSCartboost, db.SkillWSOverthrustmax, db.SkillWSCarttermination, db.SkillWSWeaponrefine} {
		m := &WorldMode{}
		m.addSkillCastEffects(ctx, skill, 3, 2000000, 300, 0, 0, time.Second, time.Now(), "test")
		if len(m.worldEffects) != 3 || m.worldEffects[2].effectID != effectBashBegin {
			t.Fatalf("skill %d must use the physical casting flare", skill)
		}
	}
}

func TestCartTerminationDamageTiming(t *testing.T) {
	ctx := archerEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobBlacksmithH, db.JobBlacksmithH
	for _, damage := range []int32{0, 100} {
		m := &WorldMode{}
		a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillWSCarttermination, Action: network.ActorActionSkill, Damage: damage, SourceSpeed: 1000}
		m.applyActorActionNotify(ctx, a)
		anim := m.actorAnims[a.SourceID]
		start := anim.started
		if anim.actionFamily != spriteActionPCSkill || start.IsZero() {
			t.Fatal("Cart Termination must use the skill release pose")
		}
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectCartTermination || m.worldEffects[0].actorID != 300 || !m.worldEffects[0].starts.Equal(start) {
			t.Fatal("Cart Termination must start on its victim at release, including misses, without an extra hit spark")
		}
		a.Action = 5
		m.applyActorActionNotify(ctx, a)
		if len(m.worldEffects) != 1 {
			t.Fatal("secondary splash notification replayed Cart Termination")
		}
		bursts := 0
		for _, sound := range m.scheduledSounds {
			if sound.paths[0] == "effect\\wizard_fire_pillar_b.wav" {
				bursts++
				if !sound.at.Equal(start.Add(30 * db.EffectFrameDuration)) {
					t.Fatal("Cart Termination impact audio must coincide with its burst at tick 30")
				}
			}
		}
		if bursts != 1 {
			t.Fatal("primary and splash packets must produce just one burst sound")
		}
	}
}

func TestWhitesmithEffectsWithClientAssets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 20, 20, 0)
	for _, tc := range []struct {
		effect int
		sound  string
	}{
		{effectMeltdown, "effect\\black_overthrust.wav"}, {effectCartBoost, "effect\\ef_incagility.wav"},
	} {
		m := &WorldMode{}
		m.addWorldEffectAt(ctx, tc.effect, 300, start)
		if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != tc.sound || !m.scheduledSounds[0].at.Equal(start) {
			t.Fatal("buff must play its native sound on activation")
		}
		if _, err := ctx.Resources.ReadFile("data/wav/" + tc.sound); err != nil {
			t.Fatal(err)
		}
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(20*db.EffectFrameDuration), false)
		v := reflect.ValueOf(frame).Elem()
		if v.FieldByName("commands").Len()+v.FieldByName("worldCommands").Len() == 0 {
			t.Fatalf("effect %d drew nothing with client STR assets", tc.effect)
		}
	}
	m := &WorldMode{}
	m.addWorldEffectAt(ctx, effectCartTermination, 300, start)
	if _, err := ctx.Resources.ReadFile("data/wav/" + m.scheduledSounds[0].paths[0]); err != nil {
		t.Fatal(err)
	}
	for _, tick := range []int{29, 55, 93} {
		frame := render.NewFrame(800, 600)
		at := start.Add(time.Duration(tick) * db.EffectFrameDuration)
		m.drawWorldEffects(frame, ctx, projection, at, false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if tick != 55 {
			if commands.Len() != 0 {
				t.Fatalf("unexpected Cart Termination particles at tick %d", tick)
			}
			continue
		}
		if commands.Len() != 1 {
			t.Fatal("Cart Termination did not draw its particle burst")
		}
		vertices := commands.Index(0).FieldByName("Vertices")
		if vertices.Len() < 24*4 || vertices.Len() > 48*4 {
			t.Fatal("incorrect size of Cart Termination burst")
		}
		center := effectWorldAnchor(ctx, m.worldEffects[0], at)
		for i := 0; i < vertices.Len(); i += 4 {
			var x, y, z float64
			for j := 0; j < 4; j++ {
				v := vertices.Index(i + j)
				x += v.FieldByName("X").Float() / 4
				y += v.FieldByName("Y").Float() / 4
				z += v.FieldByName("Z").Float() / 4
				if math.Abs(v.FieldByName("ColorA").Float()-224.0/255) > .001 {
					t.Fatal("incorrect particle fade")
				}
			}
			radius := math.Sqrt((x-center.x)*(x-center.x) + (y-center.y)*(y-center.y) + (z-center.z)*(z-center.z))
			if math.Abs(radius-5.2) > .001 {
				t.Fatal("particles must travel 26 native units before slowing")
			}
		}
	}
}
