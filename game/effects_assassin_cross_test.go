package game

import (
	"image/color"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestAssassinCrossPacketSequence(t *testing.T) {
	ctx := archerEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobAssassinH, db.JobAssassinH
	ctx.World.Player.Weapon, ctx.Session.Selected.Weapon = db.WeaponKatar, db.WeaponKatar
	start := time.Now()
	for _, skill := range []uint16{378, 406, 407} {
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed skill emitted effects")
		}
		n.Result = 1
		m.applySkillNoDamageNotify(ctx, n)
		want := []int{16, 493}
		if skill == 406 {
			want = []int{409}
		} else if skill == 407 {
			want = []int{16}
		}
		pose := spriteActionPCSkill
		if skill == 406 {
			pose = spriteActionPCAttack3
		}
		if len(m.worldEffects) != len(want) || m.actorAnims[2000000].actionFamily != pose {
			t.Fatalf("skill %d: effects=%+v pose=%+v", skill, m.worldEffects, m.actorAnims)
		}
		for i, e := range m.worldEffects {
			if e.effectID != want[i] || e.actorID != 2000000 {
				t.Fatalf("skill %d: effect %d must belong to caster: %+v", skill, want[i], e)
			}
		}
		if skill == 378 {
			var ticks []int
			for _, sound := range m.scheduledSounds {
				if len(sound.paths) == 1 && sound.paths[0] == "effect\\assasin_enchantpoison.wav" {
					ticks = append(ticks, int(sound.at.Sub(m.worldEffects[1].starts)/db.EffectFrameDuration))
					if !sound.positioned {
						t.Fatal("poison sounds must be positioned")
					}
				}
			}
			if !reflect.DeepEqual(ticks, []int{0, 5, 11, 18, 26, 40}) {
				t.Fatalf("EDP sound ticks = %v", ticks)
			}
		}
		if skill == 406 {
			a := network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Action: network.ActorActionSkill, Damage: 100}
			m.addSkillEffect(ctx, a, start, start)
			m.addSkillHitEffect(ctx, a, start)
			if len(m.worldEffects) != 2 || m.worldEffects[1].effectID != 0 || len(m.scheduledSounds) != 1 {
				t.Fatal("Meteor Assault damage replayed the burst")
			}
		}
	}
	for _, damage := range []int32{0, 100} {
		m := &WorldMode{}
		a := network.ActorActionNotify{SkillID: 379, SourceID: 2000000, TargetID: 300, Action: network.ActorActionSkill, Damage: damage}
		m.addSkillBeginEffect(ctx, a, start)
		m.addSkillBeforeHitEffect(ctx, a, start)
		m.addSkillEffect(ctx, a, start, start.Add(time.Second))
		m.addSkillHitEffect(ctx, a, start.Add(time.Second))
		want := 1
		if damage != 0 {
			want = 2
		}
		if len(m.worldEffects) != want || len(m.scheduledSounds) != 1 {
			t.Fatalf("Soul Breaker damage %d: effects=%+v sounds=%+v", damage, m.worldEffects, m.scheduledSounds)
		}
		e := m.worldEffects[0]
		if e.effectID != 361 || e.actorID != 2000000 || e.targetID != 300 || !e.starts.Equal(start) || e.actorOrigin.x != 10.5 || e.projectileDirection.x != 10 {
			t.Fatalf("Soul Breaker must snapshot caster to victim at release: %+v", e)
		}
		if m.scheduledSounds[0].at.Sub(start) != 20*db.EffectFrameDuration || !m.scheduledSounds[0].positioned || !m.cameraShakeEnd.IsZero() {
			t.Fatal("Soul Breaker must play its delayed sound without HighPriest's extra Asura sound/quake")
		}
		for _, skill := range []uint16{379, 406} {
			a.SkillID, a.Action = skill, 5
			m.addSkillBeginEffect(ctx, a, start)
			m.addSkillBeforeHitEffect(ctx, a, start)
			m.addSkillEffect(ctx, a, start, start)
			m.addSkillHitEffect(ctx, a, start)
		}
		if len(m.worldEffects) != want || len(m.scheduledSounds) != 1 {
			t.Fatal("secondary splash replayed effects or sounds")
		}
	}
	m := &WorldMode{}
	m.addSkillEffect(ctx, network.ActorActionNotify{SkillID: 379, SourceID: 2000000, TargetID: 999}, start, start)
	if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
		t.Fatal("Soul Breaker requires a visible target")
	}
}

func TestAssassinCrossClientAssets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobAssassinH, db.JobAssassinH
	ctx.World.Player.Weapon, ctx.Session.Selected.Weapon = db.WeaponKatar, db.WeaponKatar
	for _, sex := range []uint8{0, 1} {
		view, status := loadPlayerHumanoidSpriteView(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil || !strings.Contains(view.body.actSource, res.PlayerJobToken(db.JobAssassinH)) {
			t.Fatalf("must load the Assassin Cross body without fallback: %s", status)
		}
		for dir := 0; dir < 8; dir++ {
			for _, family := range []int{spriteActionPCReadyFight, spriteActionPCSkill, spriteActionPCAttack2, spriteActionPCAttack3} {
				if _, a, ok := resolveSpriteAction(view.body.act, family, dir); !ok || len(a.Animations) == 0 {
					t.Fatal("Assassin Cross casting/release ACT missing")
				}
			}
		}
	}
	for _, skill := range []uint16{378, 379, 406, 407} {
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, DelayTime: 1000, Property: 3})
		aura := 16
		if skill == 379 || skill == 406 {
			aura = 59
		}
		if len(m.worldEffects) != 2 || m.worldEffects[1].effectID != aura || m.actorAnims[2000000].actionFamily != spriteActionPCReadyFight {
			t.Fatalf("skill %d: wrong casting aura or pose", skill)
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Result: 1})
		pose := spriteActionPCSkill
		if skill == 379 || skill == 406 {
			pose = spriteActionPCAttack3
		}
		if m.actorAnims[2000000].actionFamily != pose {
			t.Fatalf("skill %d: wrong 2008 release pose", skill)
		}
	}
	for _, weapon := range []int16{db.WeaponShortsword, db.WeaponKatar} {
		ctx.World.Player.Weapon, ctx.Session.Selected.Weapon = weapon, weapon
		for _, skill := range []uint16{379, 406} {
			m := &WorldMode{}
			m.applyActorActionNotify(ctx, network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Action: network.ActorActionSkill, Damage: 100, SourceSpeed: 600})
			pose := spriteActionPCAttack2
			if weapon == db.WeaponKatar {
				pose = spriteActionPCAttack3
			}
			if m.actorAnims[2000000].actionFamily != pose {
				t.Fatalf("skill %d must use the weapon's attack pose %d", skill, pose)
			}
		}
	}
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 14, 20, 0)
	for _, tc := range []struct{ effect, tick, billboards, vertices int }{
		{361, 20, 0, 0}, {361, 30, 0, 16}, {361, 61, 0, 0},
		{409, 20, 0, 0}, {409, 40, 0, 128}, {409, 61, 0, 0},
		{493, 0, 1, 0}, {493, 39, 13, 0}, {493, 120, 0, 0},
	} {
		m := &WorldMode{}
		m.addWorldEffectBetweenAt(ctx, tc.effect, 2000000, 300, start)
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		v := reflect.ValueOf(frame).Elem()
		commands := v.FieldByName("worldCommands")
		vertices := 0
		for i := 0; i < commands.Len(); i++ {
			cmd := commands.Index(i)
			vertices += cmd.FieldByName("Vertices").Len()
			if cmd.FieldByName("Options").FieldByName("Blend").Int() != int64(render.BlendSourceOver) {
				t.Fatal("2008 slashes use normal alpha")
			}
		}
		if v.FieldByName("worldBillboards").Len() != tc.billboards || vertices != tc.vertices {
			t.Fatalf("effect %d tick %d: billboards=%d vertices=%d", tc.effect, tc.tick, v.FieldByName("worldBillboards").Len(), vertices)
		}
		if vertices > 0 {
			ctx.World.Player.X += 10
			moved := render.NewFrame(800, 600)
			m.drawWorldEffects(moved, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
			ctx.World.Player.X -= 10
			before := commands.Index(0).FieldByName("Vertices")
			after := reflect.ValueOf(moved).Elem().FieldByName("worldCommands").Index(0).FieldByName("Vertices")
			for i := 0; i < before.Len(); i++ {
				if before.Index(i).FieldByName("X").Float() != after.Index(i).FieldByName("X").Float() {
					t.Fatal("released slashes followed the moving caster")
				}
			}
		}
	}
	for _, sound := range []string{"기공포", "메테오 어썰트", "assasin_enchantpoison"} {
		if _, err := ctx.Resources.ReadFile("data/wav/effect/" + sound + ".wav"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAssassinCrossBodyLayers(t *testing.T) {
	ctx := archerEffectContext()
	start := time.Unix(100, 0)
	image := render.NewImage(12, 24)
	image.Fill(color.White)
	for _, tc := range []struct {
		effect, tick int
		purple       bool
	}{
		{361, 0, true}, {361, 10, true}, {361, 11, false},
		{409, 0, true}, {409, 11, false},
		{493, 0, true}, {493, 1, false}, {493, 80, true}, {493, 82, false},
	} {
		for layer := 0; layer < 3; layer++ {
			m := &WorldMode{}
			m.addWorldEffectAt(ctx, tc.effect, 300, start)
			frame := render.NewFrame(800, 600)
			entry := sceneActorDrawEntry{actor: ctx.World.Actors[300], worldX: 20.5, worldY: 20.5}
			billboard := &spriteBillboard{image: image, anchorX: 6, anchorY: 24, layerGroup: layer}
			m.drawActorSpriteWithEffects3D(ctx, frame, newSceneProjectionForTarget(800, 600, 20.5, 20.5, 0), entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
			draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			blend, green := render.BlendSourceOver, 1.0
			if tc.purple {
				green = 0
				if layer == 0 {
					blend = render.BlendLighter
				}
			}
			if draws.Len() != 1 || draws.Index(0).FieldByName("Options").FieldByName("Blend").Int() != int64(blend) || math.Abs(draws.Index(0).FieldByName("ColorG").Float()-green) > .00001 {
				t.Fatalf("effect %d tick %d layer %d: wrong tint or blend", tc.effect, tc.tick, layer)
			}
		}
	}
}
