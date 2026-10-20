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

func TestCreatorGroundAndProtectionPackets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobAlchemistH, db.JobAlchemistH
	start := time.Unix(100, 0)
	for _, tc := range []struct{ level, effect int }{{1, effectSlim}, {5, effectSlim}, {6, effectSlim2}, {9, effectSlim2}, {10, effectSlim3}} {
		m := &WorldMode{}
		m.startSkillSourceCastAnimation(ctx, 2000000, db.SkillCRSlimpitcher, time.Second, start)
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: db.SkillCRSlimpitcher, Level: uint16(tc.level), X: 22, Y: 21})
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.effect || m.worldEffects[0].actorID != 0 || m.worldEffects[0].x != 22 || m.worldEffects[0].y != 21 {
			t.Fatal("Aid Condensed Potion must produce one ground burst of the appropriate color")
		}
		if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "assulter_attack.wav" || m.scheduledSounds[0].volume != 40.0/150 || m.scheduledSounds[0].positioned || !m.actorAnims[2000000].started.Equal(start) {
			t.Fatal("ground burst must play its sound and preserve the casting throw animation")
		}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: db.SkillCRSlimpitcher, Level: uint16(tc.level), X: 22, Y: 21})
		if len(m.worldEffects) != 2 {
			t.Fatal("a second cast in the same cell must not be deduplicated")
		}
	}
	m := &WorldMode{}
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: db.SkillCRCultivation, X: 22, Y: 21})
	if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectBeginSpell6 || m.worldEffects[0].actorID != 2000000 || m.actorAnims[2000000].actionFamily != spriteActionPCSkill {
		t.Fatal("Cultivation must use the caster's white flash and skill pose; the server owns the spawned plant")
	}
	m = &WorldMode{}
	n := network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillCRFullprotection}
	m.applySkillNoDamageNotify(ctx, n)
	if len(m.worldEffects) != 0 {
		t.Fatal("failed Full Chemical Protection produced effects")
	}
	n.Result = 1
	m.applySkillNoDamageNotify(ctx, n)
	if len(m.worldEffects) != 2 || len(m.scheduledSounds) != 1 {
		t.Fatal("Full Chemical Protection needs rays, body flash, and one sound")
	}
	for _, e := range m.worldEffects {
		if e.actorID != n.TargetID || e.effectID != effectChemicalProt && e.effectID != effectChemicalBody {
			t.Fatal("protection effects must belong to the recipient")
		}
	}
}

func TestCreatorAcidDemonstrationPackets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobAlchemistH, db.JobAlchemistH
	m := &WorldMode{}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillCRAciddemonstration, DelayTime: 1000})
	if len(m.worldEffects) != 4 || m.worldEffects[0].effectID != effectThrowItem4 {
		t.Fatal("Acid Demonstration must throw its bottles during casting, with circle, aura, and lock-on")
	}
	e := m.worldEffects[0]
	if e.actorID != 2000000 || e.targetID != 300 || e.actorOrigin.x != 10.5 || e.projectileDirection.x != 10 {
		t.Fatal("bottle flight must capture the caster and victim at cast start")
	}
	cast := m.actorAnims[2000000].started
	m.worldEffects = nil
	m.scheduledSounds = nil
	a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillCRAciddemonstration, HitCount: 10, Damage: 100, SourceSpeed: 500, Action: network.ActorActionSkill}
	m.applyActorActionNotify(ctx, a)
	if !m.actorAnims[2000000].started.Equal(cast) || len(m.worldEffects) != 11 {
		t.Fatal("Acid Demonstration must keep its cast pose, start one impact ring, and flash for each hit")
	}
	e = m.worldEffects[0]
	if e.effectID != effectAcidDemon || !e.hasActorTransform || e.actorOrigin.x != 20.5 || !e.starts.Before(m.worldEffects[1].starts) {
		t.Fatal("Acid Demonstration rings must capture the victim's position at release, before hit flashes")
	}
	a.Action = 5
	m.applyActorActionNotify(ctx, a)
	if len(m.worldEffects) != 11 {
		t.Fatal("secondary damage packet replayed Acid Demonstration")
	}
	a.Action, a.Damage = network.ActorActionSkill, 0
	m.worldEffects = nil
	m.applyActorActionNotify(ctx, a)
	if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectAcidDemon {
		t.Fatal("a miss must still show the ground impact, without hit flashes")
	}
	m = &WorldMode{}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillCRAciddemonstration})
	if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectThrowItem4 {
		t.Fatal("a zero-duration server cast must still throw its bottles")
	}
}

func TestCreatorCastAnimationsWithClientAssets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobAlchemistH, db.JobAlchemistH
	start := time.Unix(100, 0)
	for _, sex := range []uint8{0, 1} {
		ctx.World.Player.Sex = sex
		view, status := loadPlayerHumanoidSpriteView(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil || !strings.Contains(view.body.actSource, res.PlayerJobToken(db.JobAlchemistH)) {
			t.Fatalf("must load the Creator body without a Novice fallback: %s", status)
		}
		m := &WorldMode{playerView: view}
		for _, skill := range []uint16{db.SkillCRSlimpitcher, db.SkillCRAciddemonstration} {
			m.startSkillSourceCastAnimation(ctx, 2000000, skill, 3*time.Second, start)
			anim := m.actorAnims[2000000]
			if anim.actionFamily != spriteActionPCAttack1 || anim.duration != time.Second || anim.speed != 10*db.EffectFrameDuration || anim.length != 5 || anim.loop {
				t.Fatal("Creator throw must use the finite native five-frame ACT 40 sequence")
			}
			for direction := 0; direction < 8; direction++ {
				index, action, ok := resolveSpriteAction(view.body.act, spriteActionPCAttack1, direction)
				if !ok || index/8 != spriteActionPCAttack1 || len(action.Animations) < 5 {
					t.Fatal("Creator body lacks its full throwing sequence")
				}
				state := spriteState{direction: direction}
				applyActorAnimationToSpriteState(&state, anim, true)
				for i, want := range []int{0, 1, 2, 3, 4, 4} {
					at := start.Add(time.Duration(i)*anim.speed + time.Millisecond)
					if got := bodyMotionForState(action, state, start, at); got != want {
						t.Fatalf("throw step %d: got frame %d, want %d", i, got, want)
					}
				}
			}
			if _, ok := m.actorAnimation(2000000, start.Add(time.Second)); ok {
				t.Fatal("throw pose outlived its native one-second duration")
			}
		}
	}
	for _, tc := range []struct {
		skill      uint16
		pose, aura int
	}{
		{db.SkillCRSlimpitcher, spriteActionPCAttack1, effectBeginSpell2},
		{db.SkillCRAciddemonstration, spriteActionPCAttack1, effectBeginSpell2},
		{db.SkillCRFullprotection, spriteActionIdle, effectBeginSpell2},
		{db.SkillCRCultivation, spriteActionPCReadyFight, effectBashBegin},
	} {
		m := &WorldMode{}
		m.startSkillSourceCastAnimation(ctx, 2000000, tc.skill, time.Second, start)
		m.addSkillCastEffects(ctx, tc.skill, 1, 2000000, 0, 0, 0, time.Second, start, "test")
		if m.actorAnims[2000000].actionFamily != tc.pose || len(m.worldEffects) != 2 || m.worldEffects[1].effectID != tc.aura {
			t.Fatalf("incorrect cast pose or aura for skill %d", tc.skill)
		}
	}
}

func TestCreatorEffectRenderingWithClientAssets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 20, 20, 0)
	m := &WorldMode{}
	m.addWorldEffectBetweenAt(ctx, effectThrowItem4, 300, 2000000, start)
	for _, tc := range []struct{ tick, billboards int }{{4, 0}, {5, 1}, {9, 3}, {10, 4}, {20, 8}, {60, 0}} {
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		billboards := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if billboards.Len() != tc.billboards {
			t.Fatalf("bottle frame %d: got %d billboards, want %d", tc.tick, billboards.Len(), tc.billboards)
		}
		for i := 0; i < billboards.Len(); i++ {
			if billboards.Index(i).FieldByName("ColorG").Float() != 1 {
				t.Fatal("Acid Demonstration bottles must be untinted")
			}
		}
	}
	for variant, id := range []int{effectSlim, effectSlim2, effectSlim3} {
		m := &WorldMode{}
		m.addWorldEffectAtCellLifetime(ctx, id, 0, 22, 21, start, 0, false)
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(25*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		billboards := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if billboards.Len() != 4 || commands.Len() != 1 {
			t.Fatal("potion burst needs its falling icon, three afterimages, and a ground wave")
		}
		wantVertices := 4
		if variant == 0 {
			wantVertices = 8
		}
		if commands.Index(0).FieldByName("Vertices").Len() != wantVertices {
			t.Fatal("red potion must draw its ground wave twice")
		}
		m.drawWorldEffects(render.NewFrame(800, 600), ctx, projection, start.Add(5*time.Second), false)
		if len(m.worldEffects) != 0 {
			t.Fatal("potion burst failed to expire")
		}
	}
	m = &WorldMode{}
	m.addWorldEffectAt(ctx, effectAcidDemon, 300, start)
	actor := ctx.World.Actors[300]
	actor.X = 60
	ctx.World.Actors[300] = actor
	for _, tick := range []int{19, 43, 44} {
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if tick == 44 {
			if commands.Len() != 0 {
				t.Fatal("Acid Demonstration rings survived their fade")
			}
			continue
		}
		if commands.Len() != 1 || commands.Index(0).FieldByName("Vertices").Len() != 168 {
			t.Fatal("Acid Demonstration must batch four rings using the original 20-segment meshes")
		}
		v := commands.Index(0).FieldByName("Vertices").Index(0)
		if math.Abs(v.FieldByName("X").Float()-20.5) > 1 || math.Abs(v.FieldByName("ColorG").Float()-50.0/255) > .001 {
			t.Fatal("impact rings must remain at the original victim position with their native magenta tint")
		}
	}
	texture := m.textures["__effectfile_effect/ring_black.tga|blackkey=false"]
	if texture == nil {
		t.Fatal("Acid Demonstration must retain the TGA's black pixels")
	}
	source, _, err := res.LoadImageExact(ctx.Resources, []string{"data/texture/effect/ring_black.tga"})
	if err != nil {
		t.Fatal(err)
	}
	partialAlpha := 0
	for y := 0; y < source.Bounds().Dy(); y++ {
		for x := 0; x < source.Bounds().Dx(); x++ {
			want := color.NRGBAModel.Convert(source.At(x, y)).(color.NRGBA)
			got := texture.RGBA().RGBAAt(x, y)
			if got.R != want.R || got.G != want.G || got.B != want.B || got.A != want.A {
				t.Fatalf("impact texture changed straight-alpha pixel at %d,%d: %v != %v", x, y, got, want)
			}
			if want.A > 0 && want.A < 255 {
				partialAlpha++
			}
		}
	}
	if partialAlpha == 0 {
		t.Fatal("fixture lacks the TGA's translucent edges")
	}
	spec, _ := worldEffectSpecForID(effectAcidDemon)
	if spec.cameraShakeDelay != 5*db.EffectFrameDuration || len(m.scheduledSounds) != 1 || !m.scheduledSounds[0].at.Equal(start) {
		t.Fatal("impact sound must start immediately and quake at tick five")
	}
}
