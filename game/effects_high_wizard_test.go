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

func TestHighWizardSoulDrainPackets(t *testing.T) {
	ctx := archerEffectContext()
	m := &WorldMode{}
	n := network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillHWSouldrain, Result: 1}
	m.applySkillNoDamageNotify(ctx, n)
	if len(m.worldEffects) != 2 || len(m.scheduledSounds) != 0 {
		t.Fatal("Soul Drain needs its returning souls and victim tint, without a sound")
	}
	drain, victim := m.worldEffects[0], m.worldEffects[1]
	if drain.effectID != 378 || drain.actorID != n.SourceID || drain.targetID != n.TargetID || victim.effectID != 379 || victim.actorID != n.TargetID {
		t.Fatal("Soul Drain effects belong to the wrong actors")
	}
	if drain.actorOrigin.x != 20.5 || drain.projectileDirection.x != -10 {
		t.Fatal("souls must travel from the defeated monster back to the caster")
	}
	anim := m.actorAnims[n.SourceID]
	if anim.actionFamily*8 != 88 || anim.length != 4 || anim.speed != 10*db.EffectFrameDuration || anim.duration != time.Second {
		t.Fatal("Soul Drain must force ACT 88 frames 0–3 for one second")
	}
	for _, tick := range []int{1, 30, 60} {
		p := soulDrainPosition(drain, tick, 0)
		if tick == 30 && (p.x <= 10.5 || p.x >= 25 || p.y < 7) || tick == 60 && math.Abs(p.x-10.1) > .00001 {
			t.Fatalf("incorrect native spline at tick %d: %+v", tick, p)
		}
	}
	n.Result = 0
	m.applySkillNoDamageNotify(ctx, n)
	if len(m.worldEffects) != 2 {
		t.Fatal("failed skill must not trigger Soul Drain")
	}
	n.Result = 1
	ctx.World.Player.Moving = true
	ctx.World.Player.MoveStarted, ctx.World.Player.MoveDuration = time.Now(), 5*time.Second
	ctx.World.Player.FromX, ctx.World.Player.ToX = 10, 15
	ctx.World.Player.FromY, ctx.World.Player.ToY = 20, 20
	m.applySkillNoDamageNotify(ctx, n)
	if !ctx.World.Player.Moving || ctx.World.Player.ToX != 15 {
		t.Fatal("Soul Drain's forced animation must not discard an existing walk")
	}
	delete(ctx.World.Actors, n.TargetID)
	m.worldEffects = nil
	m.applySkillNoDamageNotify(ctx, n)
	if len(m.worldEffects) != 0 {
		t.Fatal("Soul Drain requires both the caster and victim")
	}
}

func TestHighWizardMagicPowerLifecycle(t *testing.T) {
	ctx := archerEffectContext()
	m := &WorldMode{}
	change := network.StatusEffectChange{StatusID: db.StatusMagicpower, ActorID: 2000000, Active: true}
	m.applyStatusEffectChange(ctx, change)
	if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != 382 || !m.worldEffects[0].persistent || len(m.scheduledSounds) != 0 {
		t.Fatal("status entry must restore the weapon sparkle silently")
	}
	n := network.SkillNoDamageNotify{SkillID: db.SkillHWMagicpower, SourceID: 2000000, TargetID: 2000000, Result: 1}
	for i := 0; i < 2; i++ {
		m.applySkillNoDamageNotify(ctx, n)
		count := 0
		for _, e := range m.worldEffects {
			if e.effectID == 382 {
				count++
			}
		}
		if count != 1 {
			t.Fatal("recasting must replace the weapon sparkle")
		}
	}
	if len(m.scheduledSounds) != 4 || m.scheduledSounds[0].positioned || m.scheduledSounds[0].paths[0] != "effect\\마법력 증폭.wav" {
		t.Fatal("each successful activation must play one centered amplification sound")
	}
	change.Active = false
	m.applyStatusEffectChange(ctx, change)
	if m.hasActiveWorldEffect(effectLightBlade, 2000000, time.Now()) {
		t.Fatal("weapon sparkle survived buff removal")
	}
	ctx.World.Player.Opt3State = db.Opt3Lightblade
	m.syncActorSpiritEffects(ctx, 2000000)
	if !m.hasActiveWorldEffect(effectLightBlade, 2000000, time.Now()) {
		t.Fatal("actor entry must restore the weapon sparkle")
	}
}

func TestHighWizardDamageAndGroundPackets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobWizardH, db.JobWizardH
	for _, tc := range []struct {
		skill                 uint16
		effect, sounds, begin int
	}{
		{db.SkillHWMagiccrasher, 380, 1, 0}, {db.SkillHWNapalmvulcan, 401, 5, 1},
	} {
		m := &WorldMode{}
		a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: tc.skill, SourceSpeed: 500, Damage: 100, HitCount: 3, Action: network.ActorActionSkill}
		m.applyActorActionNotify(ctx, a)
		if len(m.worldEffects) != 4+tc.begin {
			t.Fatalf("skill %d needs its release effect, optional caster flare, and three hits", tc.skill)
		}
		release := m.worldEffects[tc.begin]
		if release.effectID != tc.effect || !release.starts.Equal(m.actorAnims[2000000].started) || !release.starts.Before(m.worldEffects[tc.begin+1].starts) {
			t.Fatal("release effect must precede the hit flashes")
		}
		if len(m.scheduledSounds) != tc.sounds+1+tc.begin {
			t.Fatalf("skill %d scheduled %d sounds, want %d", tc.skill, len(m.scheduledSounds), tc.sounds+1+tc.begin)
		}
		if tc.skill == db.SkillHWNapalmvulcan {
			for i, sound := range m.scheduledSounds[1+tc.begin:] {
				if sound.at.Sub(release.starts) != time.Duration(20+10*i)*db.EffectFrameDuration {
					t.Fatal("Napalm Vulcan sound must play at ticks 20, 30, 40, 50, 60 regardless of hit count")
				}
			}
		}
		a.Action = 5
		m.applyActorActionNotify(ctx, a)
		if len(m.worldEffects) != 4+tc.begin {
			t.Fatal("splash packet replayed caster/impact effects")
		}
		a.Action, a.Damage = network.ActorActionSkill, 0
		m.worldEffects = nil
		m.scheduledSounds = nil
		m.applyActorActionNotify(ctx, a)
		if len(m.worldEffects) != 1+tc.begin || len(m.scheduledSounds) != tc.sounds+tc.begin {
			t.Fatal("a miss must retain release effects and sounds, without hit flashes")
		}
	}
	m := &WorldMode{}
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: db.SkillHWGanbantein, X: 22, Y: 21})
	if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != 223 || m.worldEffects[0].x != 22 || m.worldEffects[0].actorID != 0 || m.actorAnims[2000000].actionFamily != spriteActionPCSkill || len(m.scheduledSounds) != 1 || m.scheduledSounds[0].positioned {
		t.Fatal("Ganbantein needs its ground rings, skill pose and centered sound")
	}
	m = &WorldMode{}
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: db.SkillHWGravitation, X: 22, Y: 21})
	if len(m.worldEffects) != 0 || len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\wizard_earthspike.wav" {
		t.Fatal("Gravitation ground notification must leave the field to unit entries")
	}
	m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: 184, X: 22, Y: 21, Visible: true})
	if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != 522 || !m.worldEffects[0].persistent {
		t.Fatal("Gravitation unit must own its stone and ice spikes")
	}
	m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 500})
	if len(m.worldEffects) != 0 {
		t.Fatal("Gravitation field survived its unit")
	}
	m.applyActorActionNotify(ctx, network.ActorActionNotify{SkillID: db.SkillHWGravitation, SourceID: 500, TargetID: 300, Action: network.ActorActionSkill, Damage: 100, HitCount: 1})
	if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectHit1 || m.worldEffects[0].actorID != 300 {
		t.Fatal("Gravitation damage must flash on the victim without duplicating the field")
	}
}

func TestHighWizardGravitationNativeMotion(t *testing.T) {
	position, speed, accel := 0.0, 1.0, .01
	for frame := 0; frame < 2000; frame++ {
		if frame == 550 || frame < 550 && frame%6 == 2 {
			speed, accel = -1.2, 0
		} else if frame < 550 && frame%6 == 5 {
			speed, accel = 1.18, .01
		}
		speed += accel
		position += speed
		if math.Abs(gravitationTravel(frame+1)-position) > .000001 {
			t.Fatalf("spike diverged from Prim3DQuadHorn2 at frame %d", frame)
		}
	}
}

func TestHighWizardClientAssets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobWizardH, db.JobWizardH
	start := time.Unix(100, 0)
	for _, sex := range []uint8{0, 1} {
		view, status := loadPlayerHumanoidSpriteView(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil || !strings.Contains(view.body.actSource, res.PlayerJobToken(db.JobWizardH)) {
			t.Fatalf("must load the High Wizard body without fallback: %s", status)
		}
		m := &WorldMode{playerView: view}
		m.startSkillNoDamageSourceAnimation(ctx, network.SkillNoDamageNotify{SkillID: db.SkillHWSouldrain, SourceID: 2000000, TargetID: 300}, start)
		for dir := 0; dir < 8; dir++ {
			_, action, ok := resolveSpriteAction(view.body.act, 11, dir)
			if !ok || len(action.Animations) < 4 {
				t.Fatal("High Wizard lacks Soul Drain's forced ACT sequence")
			}
			state := spriteState{direction: dir}
			applyActorAnimationToSpriteState(&state, m.actorAnims[2000000], true)
			for tick, want := range []int{0, 1, 2, 3, 3, 3} {
				if got := bodyMotionForState(action, state, start, start.Add(time.Duration(tick)*10*db.EffectFrameDuration+time.Millisecond)); got != want {
					t.Fatalf("Soul Drain frame %d: got %d, want %d", tick, got, want)
				}
			}
		}
	}
	for _, tc := range []struct {
		skill uint16
		aura  int
	}{
		{db.SkillHWMagiccrasher, 12}, {db.SkillHWMagicpower, 16}, {db.SkillHWNapalmvulcan, 12}, {db.SkillHWGanbantein, 58}, {db.SkillHWGravitation, 12},
	} {
		m := &WorldMode{}
		m.addSkillCastEffects(ctx, tc.skill, 3, 2000000, 0, 0, 0, time.Second, start, "test")
		if len(m.worldEffects) != 2 || m.worldEffects[1].effectID != tc.aura {
			t.Fatalf("incorrect native casting aura for %d", tc.skill)
		}
	}
	projection := newSceneProjectionForTarget(800, 600, 20, 20, 0)
	for _, tc := range []struct{ effect, tick, billboards, meshes int }{
		{378, 25, 21, 0}, {378, 60, 0, 0}, {223, 9, 0, 1}, {223, 34, 0, 0}, {522, 25, 0, 2}, {401, 20, 0, 0},
	} {
		m := &WorldMode{}
		if tc.effect == 378 {
			m.addWorldEffectBetweenAt(ctx, tc.effect, 2000000, 300, start)
		} else {
			m.addWorldEffectAtCellLifetime(ctx, tc.effect, 0, 20, 20, start, 0, false)
		}
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		v := reflect.ValueOf(frame).Elem()
		if v.FieldByName("worldBillboards").Len() != tc.billboards || v.FieldByName("worldCommands").Len() != tc.meshes {
			t.Fatalf("effect %d frame %d rendered the wrong geometry", tc.effect, tc.tick)
		}
		if tc.effect == 522 && v.FieldByName("worldCommands").Index(0).FieldByName("Vertices").Len() != 24 {
			t.Fatal("Gravitation must draw two four-sided spikes per texture")
		}
	}
	for _, sound := range []string{"effect\\마법력 증폭.wav", "effect\\매직 크래쉬.wav", "effect\\EF_NapalmBeat.wav", "effect\\EF_FireWall.wav", "effect\\wizard_earthspike.wav"} {
		if _, err := ctx.Resources.ReadFile("data/wav/" + sound); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHighWizardBodyLayers(t *testing.T) {
	ctx := archerEffectContext()
	start := time.Unix(100, 0)
	image := render.NewImage(12, 24)
	image.Fill(color.White)
	for _, tc := range []struct {
		effect, tick, layer, draws int
		blend                      render.Blend
	}{
		{382, 0, 0, 1, render.BlendSourceOver}, {382, 0, 1, 2, render.BlendSourceOver},
		{382, 1, 1, 1, render.BlendSourceOver}, {382, 0, 2, 1, render.BlendSourceOver},
		{380, 30, 0, 1, render.BlendLighter}, {380, 30, 1, 1, render.BlendSourceOver}, {380, 61, 0, 1, render.BlendSourceOver},
		{378, 54, 0, 1, render.BlendSourceOver}, {378, 55, 0, 1, render.BlendLighter},
		{378, 65, 0, 1, render.BlendLighter}, {378, 66, 0, 1, render.BlendSourceOver}, {378, 55, 1, 1, render.BlendSourceOver},
	} {
		m := &WorldMode{}
		m.addWorldEffectAt(ctx, tc.effect, 300, start)
		frame := render.NewFrame(800, 600)
		entry := sceneActorDrawEntry{actor: ctx.World.Actors[300], worldX: 20.5, worldY: 20.5}
		billboard := &spriteBillboard{image: image, anchorX: 6, anchorY: 24, layerGroup: tc.layer}
		m.drawActorSpriteWithEffects3D(ctx, frame, newSceneProjectionForTarget(800, 600, 20.5, 20.5, 0), entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() != tc.draws || draws.Index(draws.Len()-1).FieldByName("Options").FieldByName("Blend").Int() != int64(tc.blend) {
			t.Fatalf("wrong body/weapon rendering: %+v", tc)
		}
		if tc.draws == 2 && draws.Index(0).FieldByName("Options").FieldByName("Blend").Int() != int64(render.BlendLighter) {
			t.Fatal("weapon sparkle must add a separate additive draw")
		}
		if tc.draws == 2 && draws.Index(0).FieldByName("Options").FieldByName("DepthWrite").Bool() {
			t.Fatal("the additional weapon sparkle must not write depth")
		}
	}
}
