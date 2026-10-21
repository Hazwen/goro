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

func TestProfessorSkillPackets(t *testing.T) {
	ctx := archerEffectContext()
	for _, tc := range []struct {
		skill  uint16
		owners map[int][]uint32
		sounds int
	}{
		{373, map[int][]uint32{383: {2000000}}, 0},
		{374, map[int][]uint32{384: {2000000}, 385: {300, 2000000}, 407: {300, 2000000}}, 1},
		{375, map[int][]uint32{406: {300}, 380: {300}}, 1},
		{402, map[int][]uint32{403: {300}}, 1},
		{403, map[int][]uint32{505: {2000000}}, 1},
		{482, map[int][]uint32{521: {300}}, 1},
	} {
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Result: 1}
		m.applySkillNoDamageNotify(ctx, n)
		owners := map[int][]uint32{}
		for _, e := range m.worldEffects {
			owners[e.effectID] = append(owners[e.effectID], e.actorID)
		}
		if !reflect.DeepEqual(owners, tc.owners) || len(m.scheduledSounds) != tc.sounds {
			t.Fatalf("skill %d: effects=%v sounds=%+v", tc.skill, owners, m.scheduledSounds)
		}
		if m.actorAnims[n.SourceID].actionFamily != spriteActionPCSkill {
			t.Fatalf("skill %d must use the skill release pose", tc.skill)
		}
		if tc.skill == 374 && (m.worldEffects[0].targetID != 300 || !m.scheduledSounds[0].positioned) {
			t.Fatal("Soul Change must connect caster to recipient and position its sound at the caster")
		}
		if tc.skill == 402 && m.scheduledSounds[0].at.Sub(m.worldEffects[0].starts) != 25*db.EffectFrameDuration {
			t.Fatal("the 2008 Mind Breaker sound plays at tick 25")
		}
		if tc.skill == 482 && m.scheduledSounds[0].positioned {
			t.Fatal("Double Casting's activation sound is centered")
		}
		n.Result = 0
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.scheduledSounds) != tc.sounds {
			t.Fatal("failed cast emitted sounds")
		}
	}
	m := &WorldMode{}
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 374, SourceID: 2000000, TargetID: 999, Result: 1})
	if len(m.worldEffects) != 0 || len(m.scheduledSounds) != 0 {
		t.Fatal("Soul Change requires both actors")
	}
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 374, SourceID: 2000000, TargetID: 300, Result: 1})
	m.removeWorldEffectsForActor(300)
	if len(m.worldEffects) != 2 || m.worldEffects[0].actorID != 2000000 || m.worldEffects[1].actorID != 2000000 {
		t.Fatal("recipient removal must remove the connecting beam and recipient effects, retaining the caster flashes")
	}
	// Soul Burn level 5 sends both no-damage and damage packets; failure can hit its caster.
	for _, target := range []uint32{300, 2000000} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 375, SourceID: 2000000, TargetID: target, Result: 1})
		at := time.Now()
		a := network.ActorActionNotify{SkillID: 375, SourceID: 2000000, TargetID: target, Action: network.ActorActionSkill, Damage: 100, HitCount: 1}
		m.addSkillEffect(ctx, a, at, at)
		m.addSkillHitEffect(ctx, a, at)
		if len(m.worldEffects) != 3 || m.worldEffects[2].effectID != 0 || m.worldEffects[2].actorID != target || len(m.scheduledSounds) != 1 {
			t.Fatal("Soul Burn damage must add the hit flash without replaying its activation")
		}
		a.Action = 5
		m.addSkillEffect(ctx, a, at, at)
		m.addSkillHitEffect(ctx, a, at)
		if len(m.worldEffects) != 3 {
			t.Fatal("splash packet replayed Soul Burn effects")
		}
	}
}

func TestProfessorGroundUnits(t *testing.T) {
	ctx := archerEffectContext()
	for _, tc := range []struct {
		skill, unit uint16
		effect      int
	}{{404, 182, 405}, {405, 183, 404}} {
		m := &WorldMode{}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: tc.skill, SourceID: 2000000, X: 14, Y: 20, Level: 1})
		if len(m.worldEffects) != 0 || len(m.scheduledSounds) != 0 || m.actorAnims[2000000].actionFamily != spriteActionPCSkill {
			t.Fatal("ground notification must animate the caster without duplicating the unit's effect")
		}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 900, UnitID: tc.unit, X: 14, Y: 20, Visible: true})
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.effect || !m.worldEffects[0].persistent {
			t.Fatal("ground unit must own the native persistent effect")
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 900})
		if len(m.worldEffects) != 0 {
			t.Fatal("ground visual survived unit deletion")
		}
	}
}

func TestProfessorClientAssets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobSageH, db.JobSageH
	start := time.Unix(100, 0)
	for _, sex := range []uint8{0, 1} {
		view, status := loadPlayerHumanoidSpriteView(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil || !strings.Contains(view.body.actSource, res.PlayerJobToken(db.JobSageH)) {
			t.Fatalf("must load the Professor body without fallback: %s", status)
		}
		for dir := 0; dir < 8; dir++ {
			for _, family := range []int{spriteActionPCReadyFight, spriteActionPCSkill} {
				if _, a, ok := resolveSpriteAction(view.body.act, family, dir); !ok || len(a.Animations) == 0 {
					t.Fatal("Professor casting/release ACT is missing")
				}
			}
		}
	}
	for _, skill := range []uint16{373, 374, 375, 402, 403, 404, 405, 482} {
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Property: 3, DelayTime: 1000})
		aura := 55
		if skill == 404 {
			aura = 16
		}
		if m.actorAnims[2000000].actionFamily != spriteActionPCReadyFight || len(m.worldEffects) != 3 || m.worldEffects[2].effectID != aura {
			t.Fatalf("skill %d needs casting aura %d and the ready pose", skill, aura)
		}
	}
	projection := newSceneProjectionForTarget(800, 600, 20, 20, 0)
	for _, tc := range []struct{ effect, tick, billboards, vertices int }{
		{383, 0, 21, 0}, {383, 4, 105, 0}, {383, 60, 84, 0}, {383, 64, 0, 0},
		{384, 19, 0, 80}, {384, 77, 0, 0}, {404, 90, 0, 4}, {405, 90, 0, 24},
	} {
		m := &WorldMode{}
		m.addWorldEffectBetweenAt(ctx, tc.effect, 2000000, 300, start)
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		v := reflect.ValueOf(frame).Elem()
		commands := v.FieldByName("worldCommands")
		vertices := 0
		for i := 0; i < commands.Len(); i++ {
			vertices += commands.Index(i).FieldByName("Vertices").Len()
		}
		if v.FieldByName("worldBillboards").Len() != tc.billboards || vertices != tc.vertices {
			t.Fatalf("effect %d tick %d: billboards=%d vertices=%d", tc.effect, tc.tick, v.FieldByName("worldBillboards").Len(), vertices)
		}
		if tc.effect == 405 {
			cmd := commands.Index(0)
			if cmd.FieldByName("Options").FieldByName("Blend").Int() != int64(render.BlendSourceOver) {
				t.Fatal("Wall of Fog needs translucent grey walls, not additive light")
			}
		}
	}
	m := &WorldMode{}
	for _, file := range []string{"소울번", "사람효과"} {
		if m.loadWorldEffectSTR(ctx.Resources, file, "") == nil {
			t.Fatalf("missing native STR %s", file)
		}
	}
	for _, sound := range []string{"소울 체인지", "매직 크래쉬", "마법력 증폭", "swordman_provoke", "priest_suffragium"} {
		if _, err := ctx.Resources.ReadFile("data/wav/effect/" + sound + ".wav"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestProfessorBodyLayers(t *testing.T) {
	ctx := archerEffectContext()
	start := time.Unix(100, 0)
	image := render.NewImage(12, 24)
	image.Fill(color.White)
	for _, tc := range []struct {
		effect, tick, draws int
		tint                color.RGBA
		blend               render.Blend
	}{
		{383, 49, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver},
		{383, 50, 2, color.RGBA{150, 150, 250, 255}, render.BlendSourceOver},
		{383, 80, 2, color.RGBA{90, 90, 250, 255}, render.BlendSourceOver},
		{385, 40, 1, color.RGBA{200, 150, 50, 255}, render.BlendLighter},
		{385, 71, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver},
		{505, 10, 1, color.RGBA{250, 250, 100, 255}, render.BlendLighter},
		{505, 11, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver},
		{505, 36, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver},
		{521, 10, 2, color.RGBA{255, 0, 0, 255}, render.BlendLighter},
		{521, 11, 2, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver},
		{521, 51, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver},
	} {
		for layer := 0; layer < 3; layer++ {
			m := &WorldMode{}
			m.addWorldEffectAt(ctx, tc.effect, 300, start)
			frame := render.NewFrame(800, 600)
			entry := sceneActorDrawEntry{actor: ctx.World.Actors[300], worldX: 20.5, worldY: 20.5}
			billboard := &spriteBillboard{image: image, anchorX: 6, anchorY: 24, layerGroup: layer}
			m.drawActorSpriteWithEffects3D(ctx, frame, newSceneProjectionForTarget(800, 600, 20.5, 20.5, 0), entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
			draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			want, blend := tc.draws, tc.blend
			if layer != 0 {
				want, blend = 1, render.BlendSourceOver
			}
			if draws.Len() != want || draws.Index(draws.Len()-1).FieldByName("Options").FieldByName("Blend").Int() != int64(blend) {
				t.Fatalf("wrong body/weapon rendering for %+v layer %d", tc, layer)
			}
			for i, channel := range []string{"ColorR", "ColorG", "ColorB"} {
				want := []uint8{tc.tint.R, tc.tint.G, tc.tint.B}[i]
				if math.Abs(draws.Index(draws.Len()-1).FieldByName(channel).Float()-float64(want)/255) > .00001 {
					t.Fatalf("wrong %s tint for %+v layer %d", channel, tc, layer)
				}
			}
		}
	}
}
