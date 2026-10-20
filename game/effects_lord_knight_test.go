package game

import (
	"image/color"
	"math"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

func TestLordKnightAttackEffects(t *testing.T) {
	for _, tc := range []struct {
		skill              uint16
		effect, hit, count int
		owner              uint32
	}{
		{db.SkillLKSpiralpierce, effectMagnum2, effectPierce, 1, 300},
		{db.SkillLKHeadcrush, effectBash3D3, effectHit1, 5, 2000000},
		{db.SkillLKJointbeat, effectBash3D4, effectHit1, 5, 2000000},
	} {
		ctx := knightEffectContext()
		m := &WorldMode{}
		start := time.Unix(100, 0)
		hit := start.Add(300 * time.Millisecond)
		action := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Damage: 500, HitCount: 5, Action: 8}
		m.addSkillBeginEffect(ctx, action, start)
		m.addSkillEffect(ctx, action, start, hit)
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.effect || m.worldEffects[0].actorID != tc.owner || !m.worldEffects[0].starts.Equal(start) {
			t.Fatalf("skill %d must start once on %d: %+v", tc.skill, tc.owner, m.worldEffects)
		}
		m.addSkillHitEffect(ctx, action, hit)
		if len(m.worldEffects) != 1+tc.count {
			t.Fatalf("skill %d: wrong impact count: %+v", tc.skill, m.worldEffects)
		}
		for i, effect := range m.worldEffects[1:] {
			if effect.effectID != tc.hit || effect.actorID != 300 || !effect.starts.Equal(hit.Add(time.Duration(i)*multiHitDelay)) {
				t.Fatalf("skill %d: misplaced or mistimed impact: %+v", tc.skill, effect)
			}
		}
		m = &WorldMode{}
		action.Damage = 0
		m.addSkillHitEffect(ctx, action, hit)
		if len(m.worldEffects) != 0 {
			t.Fatal("miss displayed an impact")
		}
		action.Damage, action.Action = 500, 5
		m.applyActorActionNotify(ctx, action)
		if len(m.worldEffects) != 0 {
			t.Fatal("splash notification replayed effects")
		}
	}
}

func TestLordKnightBuffsAndCast(t *testing.T) {
	for _, tc := range []struct {
		skill  uint16
		effect int
	}{
		{db.SkillLKAurablade, effectAuraBlade},
		{db.SkillLKConcentration, effectLKConcentration},
		{db.SkillLKParrying, effectGuard},
		{db.SkillLKBerserk, effectRedBody},
	} {
		ctx := knightEffectContext()
		ctx.Session.Selected = session.Character{ID: 100, Job: db.JobKnightH}
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed buff played an effect")
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		m.applySkillNoDamageNotify(ctx, notify)
		count := 0
		for _, effect := range m.worldEffects {
			if effect.effectID == tc.effect {
				count++
				if effect.actorID != 2000000 {
					t.Fatalf("skill %d buff must belong to caster", tc.skill)
				}
			}
		}
		if tc.skill != db.SkillLKParrying && count != 1 {
			t.Fatalf("skill %d stacked buff effects", tc.skill)
		}
		if tc.skill == db.SkillLKParrying && m.actorAnims[2000000].actionFamily != spriteActionIdle {
			t.Fatal("Parrying must use standing pose")
		}
		if tc.skill == db.SkillLKAurablade {
			anim := m.actorAnims[2000000]
			if anim.actionFamily*8 != 32 || anim.length != 6 || anim.duration != time.Second || anim.speed != 10*db.EffectFrameDuration {
				t.Fatalf("Aura Blade must animate native action 32, frames 0..5: %+v", anim)
			}
		}
	}
	ctx := knightEffectContext()
	m := &WorldMode{}
	start := time.Unix(100, 0)
	m.addSkillCastEffects(ctx, db.SkillLKSpiralpierce, 0, 2000000, 300, 0, 0, 2*time.Second, start, "test")
	if !m.hasActiveWorldEffect(effectPierceBody, 2000000, start) || len(m.scheduledSounds) != 1 {
		t.Fatal("Spiral Pierce must use its yellow casting body and begin-spell sound")
	}
	if m.hasActiveWorldEffect(effectPierceBody, 2000000, start.Add(2*time.Second+time.Nanosecond)) {
		t.Fatal("Spiral Pierce body outlasted its cast")
	}
	// Relax's sitting state is sent by the server; it has no extra effect.
	m = &WorldMode{}
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillLKTensionrelax, SourceID: 2000000, TargetID: 2000000, Result: 1})
	if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
		t.Fatal("Relax invented an effect or sound")
	}
}

func TestLordKnightBerserkState(t *testing.T) {
	ctx := knightEffectContext()
	m := &WorldMode{}
	change := network.StatusEffectChange{StatusID: db.StatusBerserk, ActorID: 2000000, Active: true}
	m.applyStatusEffectChange(ctx, change)
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillLKBerserk, SourceID: 2000000, TargetID: 2000000, Result: 1})
	count := 0
	for _, effect := range m.worldEffects {
		if effect.effectID == effectRedBody {
			count++
			if !effect.persistent || !effect.expires.IsZero() {
				t.Fatal("Berserk must follow server state")
			}
		}
	}
	if count != 1 || len(m.scheduledSounds) != 1 {
		t.Fatal("status plus skill notification duplicated Berserk")
	}
	change.Active = false
	m.applyStatusEffectChange(ctx, change)
	if m.hasActiveWorldEffect(effectRedBody, 2000000, time.Now()) {
		t.Fatal("Berserk lingered after status ended")
	}
	actor := ctx.World.Actors[300]
	actor.Opt3State, actor.HasState = db.Opt3Berserk, true
	ctx.World.Actors[300] = actor
	m = &WorldMode{}
	m.syncCurrentActorEffectStateEffects(ctx)
	if !m.hasActiveWorldEffect(effectRedBody, 300, time.Now()) || len(m.scheduledSounds) != 0 {
		t.Fatal("existing Berserk state must restore silently")
	}
	m.removeActorEffectStateEffects(300)
	if m.hasActiveWorldEffect(effectRedBody, 300, time.Now()) {
		t.Fatal("vanished actor retained Berserk")
	}
}

func TestLordKnightConcentrationState(t *testing.T) {
	for _, actorID := range []uint32{0, 100, 2000000, 300} {
		for _, statusFirst := range []bool{false, true} {
			ctx := knightEffectContext()
			m := &WorldMode{}
			owner := actorID
			if owner != 300 {
				owner = 2000000
			}
			change := network.StatusEffectChange{StatusID: db.StatusLkconcentration, ActorID: actorID, Active: true}
			if statusFirst {
				m.applyStatusEffectChange(ctx, change)
			}
			notify := network.SkillNoDamageNotify{SkillID: db.SkillLKConcentration, SourceID: owner, TargetID: owner, Result: 1}
			m.applySkillNoDamageNotify(ctx, notify)
			if !statusFirst {
				m.applyStatusEffectChange(ctx, change)
			}
			later := time.Now().Add(time.Hour)
			if !m.hasActiveWorldEffect(effectLKConcentration, owner, later) || len(m.worldEffects) != 1 || len(m.scheduledSounds) != 1 {
				t.Fatalf("actor=%d statusFirst=%t: Concentration must persist with one activation sound", actorID, statusFirst)
			}
			m.applySkillNoDamageNotify(ctx, notify)
			if len(m.worldEffects) != 1 || len(m.scheduledSounds) != 2 {
				t.Fatal("recasting must replace the activation effect and play its sound once")
			}
			m.removeActorEffectStateEffects(owner)
			if len(m.worldEffects) != 0 {
				t.Fatal("vanishing retained Concentration")
			}
			if owner == 2000000 {
				m = &WorldMode{}
				m.syncCurrentActorEffectStateEffects(ctx)
				if !m.hasActiveWorldEffect(effectLKConcentration, owner, later) || len(m.scheduledSounds) != 0 {
					t.Fatal("map entry must restore the active local buff without replaying its sound")
				}
			} else {
				m.applyStatusEffectChange(ctx, change)
			}
			change.Active = false
			m.applyStatusEffectChange(ctx, change)
			if m.hasActiveWorldEffect(effectLKConcentration, owner, time.Now()) {
				t.Fatal("Concentration tint survived status end")
			}
		}
	}
	ctx := knightEffectContext()
	ctx.Session.Statuses.Active = map[uint16]session.StatusEffect{
		db.StatusLkconcentration: {HasDuration: true, ExpiresAt: time.Now().Add(-time.Second)},
	}
	m := &WorldMode{}
	m.syncCurrentActorEffectStateEffects(ctx)
	if len(m.worldEffects) != 0 {
		t.Fatal("map entry restored an expired Concentration buff")
	}
}

func TestLordKnightBodyAndWeaponRendering(t *testing.T) {
	ctx := knightEffectContext()
	actor := ctx.World.Actors[300]
	start := time.Unix(100, 0)
	texture := render.NewImage(12, 24)
	texture.Fill(color.White)
	billboard := &spriteBillboard{image: texture, anchorX: 6, anchorY: 24}
	for _, tc := range []struct {
		effect, frame, group, count int
		aura                        bool
		blend                       render.Blend
		green, blue                 float64
	}{
		{effectRedBody, 0, 0, 1, false, render.BlendLighter, 100.0 / 255, 100.0 / 255},
		{effectRedBody, 0, 1, 1, false, render.BlendSourceOver, 100.0 / 255, 100.0 / 255},
		{effectRedBody, 0, 2, 1, false, render.BlendSourceOver, 100.0 / 255, 100.0 / 255},
		{0, 0, 1, 1, true, render.BlendLighter, 1, 1},
		{0, 0, 0, 1, true, render.BlendSourceOver, 1, 1},
		{0, 0, 2, 1, true, render.BlendSourceOver, 1, 1},
		{effectPierceBody, 6, 0, 2, false, render.BlendLighter, 250.0 / 255, 100.0 / 255},
		{effectPierceBody, 6, 1, 1, false, render.BlendSourceOver, 250.0 / 255, 100.0 / 255},
		{effectPierceBody, 6, 2, 1, false, render.BlendSourceOver, 250.0 / 255, 100.0 / 255},
		{effectPierceBody, 7, 0, 2, false, render.BlendSourceOver, 1, 1},
		{effectPierceBody, 141, 0, 1, false, render.BlendSourceOver, 1, 1},
		{effectBash3D3, 20, 0, 2, false, render.BlendSourceOver, 1, 200.0 / 255},
		{effectBash3D3, 20, 1, 1, false, render.BlendSourceOver, 1, 200.0 / 255},
		{effectBash3D3, 20, 2, 1, false, render.BlendSourceOver, 1, 200.0 / 255},
		{effectBash3D4, 20, 0, 2, false, render.BlendSourceOver, 1, 1},
		{effectBash3D4, 20, 1, 1, false, render.BlendSourceOver, 1, 1},
		{effectBash3D4, 20, 2, 1, false, render.BlendSourceOver, 1, 1},
		{effectLKConcentration, 0, 0, 1, false, render.BlendSourceOver, 1, 160.0 / 255},
		{effectLKConcentration, 0, 1, 1, false, render.BlendSourceOver, 1, 160.0 / 255},
		{effectLKConcentration, 0, 2, 1, false, render.BlendSourceOver, 1, 160.0 / 255},
	} {
		m := &WorldMode{}
		m.addWorldEffectAt(ctx, tc.effect, 300, start)
		actor.Opt3State = 0
		if tc.aura {
			actor.Opt3State = db.Opt3Aurablade
		}
		entry := sceneActorDrawEntry{actor: actor, worldX: 30.5, worldY: 20.5}
		frame := render.NewFrame(800, 600)
		billboard.layerGroup = tc.group
		m.drawActorSpriteWithEffects3D(ctx, frame, newSceneProjectionForTarget(800, 600, 30.5, 20.5, 0), entry, billboard, 1, 1, 1,
			color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tc.frame)*db.EffectFrameDuration))
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() != tc.count {
			t.Fatalf("%+v: got %d draws", tc, draws.Len())
		}
		body := draws.Index(draws.Len() - 1)
		if body.FieldByName("Options").FieldByName("Blend").Int() != int64(tc.blend) ||
			math.Abs(body.FieldByName("ColorG").Float()-tc.green) > .0001 ||
			math.Abs(body.FieldByName("ColorB").Float()-tc.blue) > .0001 {
			t.Fatalf("%+v: incorrect tint or blend", tc)
		}
	}
}

func TestLordKnightRealAssets(t *testing.T) {
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		t.Skip("set GORO_DATA_DIR to validate original Lord Knight assets")
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, archive := range manager.Archives {
		t.Cleanup(func() { _ = archive.Close() })
	}
	ctx := knightEffectContext()
	ctx.Resources = manager
	start := time.Unix(100, 0)
	for _, skill := range []uint16{db.SkillLKConcentration, db.SkillLKBerserk} {
		for _, sound := range db.SkillEffects[skill].SFX {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, id := range []int{effectAuraBlade, effectMagnum2, effectBash3D3, effectBash3D4, effectGuard, effectLKConcentration} {
		m := &WorldMode{}
		spec, _ := worldEffectSpecForID(id)
		for _, sound := range spec.sfx {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Fatal(err)
			}
		}
		m.addWorldEffectAt(ctx, id, 300, start)
		drew := false
		for _, tick := range []int{0, 15, 24, 30, 40, 60} {
			frame := render.NewFrame(800, 600)
			m.drawWorldEffects(frame, ctx, newSceneProjectionForTarget(800, 600, 30.5, 20.5, 0), start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			drew = drew || reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() > 0
		}
		if !drew {
			t.Fatalf("effect %d drew nothing", id)
		}
	}
	m := &WorldMode{}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusLkconcentration, ActorID: 300, Active: true})
	effect := m.worldEffects[0]
	spec, _ := worldEffectSpecForID(effectLKConcentration)
	duration := m.worldEffectResolvedComponentDuration(ctx, spec, spec.components[0])
	for _, cycle := range []int{0, 1, 2} {
		frame := render.NewFrame(800, 600)
		at := effect.starts.Add(time.Duration(cycle)*duration + duration/2)
		m.drawWorldEffects(frame, ctx, newSceneProjectionForTarget(800, 600, 30.5, 20.5, 0), at, false)
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len()
		if (draws > 0) != (cycle == 0) {
			t.Fatalf("Concentration activation must play exactly once: cycle=%d draws=%d", cycle, draws)
		}
		if !m.hasActiveWorldEffect(effectLKConcentration, 300, at) {
			t.Fatal("Concentration tint expired with its activation STR")
		}
	}
	for _, tc := range []struct {
		id     int
		delays []time.Duration
	}{
		{effectMagnum2, []time.Duration{20 * db.EffectFrameDuration, 30 * db.EffectFrameDuration}},
		{effectBash3D3, []time.Duration{0}},
		{effectBash3D4, []time.Duration{22 * db.EffectFrameDuration}},
	} {
		spec, _ := worldEffectSpecForID(tc.id)
		if !reflect.DeepEqual(spec.sfxDelays, tc.delays) {
			t.Fatalf("effect %d sound timing", tc.id)
		}
	}
	view, status := loadHumanoidSpriteViewWithAppearance(manager, humanoidAppearance{
		job: db.JobKnightH, head: 1, sex: 1, weapon: int(db.WeaponSword), shield: 1,
	}, "test")
	if view == nil {
		t.Fatal(status)
	}
	// Separating blend groups must preserve the original IMF order and anchors.
	for _, imf := range []*res.IMF{view.imf, nil} {
		copy := *view
		copy.imf = imf
		copy.billboards = make(map[humanoidBillboardKey]*spriteBillboard)
		for direction := 0; direction < 8; direction++ {
			state := spriteState{actionFamily: spriteActionPCAttack2, direction: direction, hasFixedMotion: true, fixedMotion: 2}
			whole, ok := humanoidBillboardForState(&copy, state, start)
			if !ok {
				t.Fatal("missing Lord Knight attack")
			}
			state.separateEquipment = true
			split, ok := humanoidBillboardForState(&copy, state, start)
			if !ok || len(split.parts) == 0 {
				t.Fatal("missing split Lord Knight")
			}
			got, want := render.NewImage(512, 512), render.NewImage(512, 512)
			draw := func(target *render.Image, b *spriteBillboard) {
				options := &render.DrawImageOptions{}
				options.GeoM.Translate(256-b.anchorX, 256-b.anchorY)
				target.DrawImage(b.image, options)
			}
			draw(want, whole)
			for _, part := range split.parts {
				draw(got, part)
			}
			for i, channel := range want.RGBA().Pix {
				if absInt(int(channel)-int(got.RGBA().Pix[i])) > 2 {
					t.Fatalf("split layers changed composition: direction=%d imf=%t byte=%d", direction, imf != nil, i)
				}
			}
			if again, _ := humanoidBillboardForState(&copy, state, start); again != split {
				t.Fatal("split sprite was not cached")
			}
		}
	}
}
