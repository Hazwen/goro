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

func TestTaekwonReadyPosesBelongToSkillPackets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobTaekwon, db.JobTaekwon
	for _, tc := range []struct {
		skill, status uint16
		family, frame int
	}{
		{412, db.StatusStormkickOn, spriteActionPCSkill, 0},
		{414, db.StatusDownkickOn, spriteActionPCSkill, 2},
		{416, db.StatusTurnkickOn, spriteActionPCSkill, 3},
		{418, db.StatusCounterOn, spriteActionPCSkill, 4},
		{420, db.StatusDodgeOn, spriteActionPickup, 1},
	} {
		m := &WorldMode{}
		for _, status := range []uint16{tc.status, tc.status + 1} {
			for _, active := range []bool{true, false} {
				m.applyStatusEffectChange(ctx, network.StatusEffectChange{ActorID: 2000000, StatusID: status, Active: active})
			}
		}
		if len(m.actorAnims)+len(m.worldEffects) != 0 {
			t.Fatal("stance status packets must only update the status icons")
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 2000000, Result: 1})
		anim := m.actorAnims[100]
		if anim.actionFamily != tc.family || !anim.hasFixedMotion || anim.fixedMotion != tc.frame || anim.play || anim.duration != 2*time.Second {
			t.Fatalf("skill %d ready animation: %+v", tc.skill, anim)
		}
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectTaeReady {
			t.Fatalf("skill %d must flash blue: %+v", tc.skill, m.worldEffects)
		}
		if _, ok := m.actorAnimation(100, anim.started.Add(2*time.Second)); ok {
			t.Fatal("ready pose did not expire")
		}
	}
}

func TestTaekwonHighJumpCastAndLanding(t *testing.T) {
	ctx := archerEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobTaekwon, db.JobTaekwon
	for _, delay := range []uint32{0, 1000} {
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: 426, SourceID: 2000000, DelayTime: delay})
		count := 2
		if delay > 0 {
			count++ // The casting circle still accompanies non-instant casts.
		}
		if len(m.worldEffects) != count || m.worldEffects[0].effectID != effectJumpBody || m.worldEffects[1].effectID != effectPeong {
			t.Fatalf("High Jump cast must launch takeoff and dust, even without a delay: %+v", m.worldEffects)
		}
		start := m.actorAnims[2000000].started
		for _, tc := range []struct{ tick, motion int }{{0, 0}, {29, 0}, {30, 1}, {40, 1}} {
			anim, ok := m.actorAnimation(2000000, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
			if !ok || anim.actionFamily != spriteActionPCAttack1 || anim.fixedMotion != tc.motion {
				t.Fatalf("High Jump casting pose tick %d: %+v", tc.tick, anim)
			}
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 426, SourceID: 100, TargetID: 100, Result: 1})
		if len(m.scheduledSounds) != 0 {
			t.Fatal("landing before takeoff must cancel the pending takeoff sound")
		}
		for _, e := range m.worldEffects {
			if e.effectID == effectJumpBody {
				t.Fatal("landing did not remove takeoff through the local actor alias")
			}
		}
		start = m.actorAnims[100].started
		anim, ok := m.actorAnimation(100, start.Add(40*db.EffectFrameDuration))
		if !ok || anim.actionFamily != spriteActionWalk || anim.fixedMotion != 5 {
			t.Fatalf("High Jump landing must finish on walk motion five: %+v", anim)
		}
		if ctx.World.Player.X != 10 || ctx.World.Player.Y != 20 {
			t.Fatal("High Jump visuals moved the authoritative actor")
		}
	}
}

func TestTaekwonParticlesDoNotDependOnFrameRate(t *testing.T) {
	ctx := archerEffectContext()
	start := time.Unix(100, 0)
	ctx.World.Player.Moving = true
	ctx.World.Player.MoveStarted, ctx.World.Player.MoveDuration = start, 2*time.Second
	ctx.World.Player.FromX, ctx.World.Player.FromY = 10, 20
	ctx.World.Player.ToX, ctx.World.Player.ToY = 20, 20
	var wantFootprints []worldEffect
	var wantJump taekwonJumpParticles
	for _, step := range []int{1, 5, 60} {
		m := &WorldMode{}
		m.addWorldEffectAt(ctx, effectRun, 100, start)
		jump := taekwonJumpParticles{tick: 35}
		for tick := step; tick <= 60; tick += step {
			at := start.Add(time.Duration(tick) * db.EffectFrameDuration)
			m.advanceTaekwonFootprints(ctx, at)
			jump.advance(ctx, worldEffect{effectID: effectPeong, actorID: 100, starts: start}, at)
		}
		if step == 1 {
			wantFootprints, wantJump = m.worldEffects[1:], jump
			continue
		}
		if !reflect.DeepEqual(m.worldEffects[1:], wantFootprints) || jump != wantJump {
			t.Fatalf("particle history changed with an update every %d ticks", step)
		}
	}
}

func TestTaekwonSecondaryKickDoesNotRepeatRelease(t *testing.T) {
	for _, skill := range []uint16{415, 417, 419, 421} {
		ctx := archerEffectContext()
		m := &WorldMode{}
		before := time.Now()
		m.applyActorActionNotify(ctx, network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Action: 5, SourceSpeed: 400, Damage: 10, HitCount: 1})
		if len(m.worldEffects) != 0 {
			t.Fatal("secondary kick replayed release effects")
		}
		if _, ok := m.actorAnims[100]; ok {
			t.Fatal("secondary kick restarted the caster pose")
		}
		for _, sound := range m.scheduledSounds {
			if sound.actorID != 300 || sound.at.Sub(before) > 450*time.Millisecond {
				t.Fatalf("secondary kick acquired the primary hit's extra delay: %+v", sound)
			}
		}
	}
}

func TestTaekwonFootprintsRemainAfterRunStops(t *testing.T) {
	ctx := archerEffectContext()
	start := time.Unix(100, 0)
	m := &WorldMode{}
	m.addWorldEffectAt(ctx, effectRun, 100, start)
	m.advanceTaekwonFootprints(ctx, start.Add(7*db.EffectFrameDuration))
	if len(m.worldEffects) != 1 {
		t.Fatal("stationary Run left footprints")
	}
	for tick := 14; tick <= 21; tick += 7 {
		ctx.World.Player.X++
		m.advanceTaekwonFootprints(ctx, start.Add(time.Duration(tick)*db.EffectFrameDuration))
	}
	if len(m.worldEffects) != 3 || m.worldEffects[1].effectID != effectFoot4 || m.worldEffects[2].effectID != effectFoot3 {
		t.Fatalf("Run must alternate left/right footprints: %+v", m.worldEffects)
	}
	m.removeWorldEffect(effectRun, 100)
	ctx.World.Player.X = 50
	m.advanceTaekwonFootprints(ctx, start.Add(28*db.EffectFrameDuration))
	if len(m.worldEffects) != 2 || m.worldEffects[0].actorID != 0 || m.worldEffects[0].actorOrigin.x != 11.5 || m.worldEffects[1].actorOrigin.x != 12.5 {
		t.Fatal("stopping Run must leave its footprints where they were emitted")
	}
}

func TestTaekwonEffectsRenderWithNativeAssets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	for _, job := range []int16{db.JobTaekwon, db.JobStar, db.JobLinker} {
		ctx.World.Player.Job, ctx.Session.Selected.Job = job, job
		ctx.World.Player.Weapon, ctx.Session.Selected.Weapon = 0, 0
		for _, sex := range []uint8{0, 1} {
			view, status := loadPlayerHumanoidSpriteView(ctx.Resources, ctx.Session.Selected, sex, false)
			if view == nil {
				t.Fatalf("job %d sprite missing: %s", job, status)
			}
			for _, pose := range []struct{ family, motion int }{{spriteActionPCSkill, 5}, {spriteActionPCAttack1, 4}, {spriteActionPCAttack2, 8}, {spriteActionPCAttack3, 7}, {spriteActionPickup, 1}, {spriteActionWalk, 5}} {
				if job == db.JobLinker && pose.family == spriteActionPCAttack2 {
					continue // Soul Linkers cannot learn Heel Drop.
				}
				for dir := 0; dir < 8; dir++ {
					_, action, ok := resolveSpriteAction(view.body.act, pose.family, dir)
					if !ok || len(action.Animations) <= pose.motion {
						t.Fatalf("job %d sex %d family %d direction %d lacks motion %d", job, sex, pose.family, dir, pose.motion)
					}
				}
			}
		}
	}
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 15, 20, 0)
	for _, tc := range []struct{ id, tick, end int }{{411, 60, 200}, {430, 40, 90}, {435, 20, 110}, {439, 20, 60}, {444, 10, 30}, {447, 30, 301}, {448, 30, 301}, {461, 20, 110}, {467, 20, 251}, {473, 20, 251}} {
		m := &WorldMode{}
		if !m.addWorldEffectBetweenAt(ctx, tc.id, 300, 2000000, start) {
			t.Fatalf("effect %d has no implementation", tc.id)
		}
		spec, _ := worldEffectSpecForID(tc.id)
		for _, component := range spec.components {
			for _, file := range append([]string{component.textureFile}, component.textureFiles...) {
				if file != "" && m.effectFileTexture(ctx.Resources, file) == nil {
					t.Fatalf("effect %d missing texture %s", tc.id, file)
				}
			}
		}
		for _, tick := range []int{tc.tick, tc.end} {
			frame := render.NewFrame(800, 600)
			m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			commands := reflect.ValueOf(frame).Elem()
			count := commands.FieldByName("worldCommands").Len() + commands.FieldByName("worldBillboards").Len()
			if (count > 0) != (tick == tc.tick) {
				t.Fatalf("effect %d tick %d: draws %d", tc.id, tick, count)
			}
		}
	}
	for _, sound := range []string{"T_낙법.wav", "T_피링.wav", "T_바람방출.wav", "T_벽튕김.wav", "T_회피2.wav", "T_돌려차기.wav", "T_내려찍기.wav", "T_날라차기.wav", "T_회오리차기.wav", "T_효과음1.wav"} {
		if _, err := ctx.Resources.ReadFile("data/wav/effect/" + sound); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTaekwonBodyTransformsOnlyChangeRendering(t *testing.T) {
	ctx := archerEffectContext()
	start := time.Unix(100, 0)
	actor := ctx.World.Actors[300]
	entry := sceneActorDrawEntry{actor: actor, worldX: 20.5, worldY: 20.5}
	billboard := &spriteBillboard{image: render.NewImage(40, 80), anchorX: 20, anchorY: 80}
	projection := newSceneProjectionForTarget(800, 600, 20, 20, 0)
	for _, tc := range []struct{ id, tick int }{{413, 22}, {414, 7}, {415, 45}, {426, 5}, {445, 40}, {446, 20}, {458, 15}, {466, 25}} {
		m := &WorldMode{}
		m.addWorldEffectAt(ctx, tc.id, actor.ID, start)
		frame := render.NewFrame(800, 600)
		m.drawActorSpriteWithEffects3D(ctx, frame, projection, entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if draws.Len() != 1 {
			t.Fatalf("effect %d should transform the single body: %d draws", tc.id, draws.Len())
		}
		for _, name := range []string{"Center", "RightAxis", "UpAxis", "DepthUpAxis"} {
			for axis := 0; axis < 3; axis++ {
				v := draws.Index(0).FieldByName(name).Index(axis).Float()
				if math.IsNaN(v) || math.IsInf(v, 0) {
					t.Fatalf("effect %d produced invalid %s", tc.id, name)
				}
			}
		}
		if tc.id == 413 {
			// Keeping the feet fixed requires anchorY/height to remain one.
			if math.Abs(draws.Index(0).FieldByName("AnchorY").Float()-80) > .0001 {
				t.Fatal("Heel Drop squash lifted the victim off the floor")
			}
		}
		if !reflect.DeepEqual(ctx.World.Actors[300], actor) || billboard.anchorY != 80 {
			t.Fatal("body effect mutated the actor or shared sprite")
		}
	}
}

func TestTaekwonKicksUseNativeMotionSequences(t *testing.T) {
	ctx := archerEffectContext()
	for _, tc := range []struct {
		skill  uint16
		job    int16
		family int
		frames [5]int
		ticks  int
	}{
		{415, db.JobTaekwon, spriteActionPCAttack2, [5]int{3, 5, 7, 8, 2}, 5},
		{417, db.JobTaekwon, spriteActionPCAttack3, [5]int{4, 5, 6, 7, 2}, 5},
		{419, db.JobTaekwon, spriteActionPCAttack1, [5]int{2, 3, 4, 4, 4}, 4},
		{419, db.JobLinker, spriteActionPCAttack3, [5]int{4, 5, 6, 7, 2}, 4},
	} {
		ctx.World.Player.Job, ctx.Session.Selected.Job = tc.job, int16(tc.job)
		m := &WorldMode{}
		m.applyActorActionNotify(ctx, network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Action: network.ActorActionSkill, SourceSpeed: 400, Damage: 10, HitCount: 1})
		start := m.actorAnims[100].started
		for i, frame := range tc.frames {
			anim, ok := m.actorAnimation(100, start.Add(time.Duration(i*tc.ticks)*db.EffectFrameDuration))
			if !ok || anim.actionFamily != tc.family || !anim.hasFixedMotion || anim.fixedMotion != frame || anim.play {
				t.Fatalf("skill %d job %d motion %d = %+v", tc.skill, tc.job, i, anim)
			}
		}
		if _, ok := m.actorAnimation(100, start.Add(500*time.Millisecond)); ok {
			t.Fatal("kick animation did not expire")
		}
	}
}

func TestTaekwonRunPacketOrderAndStopMarker(t *testing.T) {
	ctx := archerEffectContext()
	m := &WorldMode{}
	run := network.SkillNoDamageNotify{SkillID: 411, SourceID: 2000000, TargetID: 2000000, Result: 1}
	status := network.StatusEffectChange{ActorID: 100, StatusID: db.StatusRun, Active: true}
	m.applyStatusEffectChange(ctx, status)
	if len(m.worldEffects) != 0 {
		t.Fatal("initial status must not start or stop running")
	}
	m.applySkillNoDamageNotify(ctx, run)
	if !m.hasActiveWorldEffect(effectRun, localSkillTarget(ctx), time.Now()) {
		t.Fatal("Run skill did not start footprints")
	}
	status.Active = false
	m.applyStatusEffectChange(ctx, status)
	m.applySkillNoDamageNotify(ctx, run)
	ids := []int{}
	for _, e := range m.worldEffects {
		ids = append(ids, e.effectID)
	}
	if !reflect.DeepEqual(ids, []int{effectStopRun, effectStopEffect}) {
		t.Fatalf("terminal Run packet restarted running: %v", ids)
	}
	if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\T_효과음1.wav" {
		t.Fatal("Run stop sound must play once")
	}
}

func TestMildWindReplacesCasterEffectsAndUsesLevel(t *testing.T) {
	ctx := archerEffectContext()
	for level := uint16(0); level <= 8; level++ {
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: 425, SourceID: 2000000, TargetID: 300, Amount: uint32(level), Result: 1}
		m.applySkillNoDamageNotify(ctx, notify)
		if level == 0 || level == 8 {
			if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
				t.Fatal("invalid Mild Wind level produced an effect")
			}
			continue
		}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects) != 2 || m.worldEffects[0].effectID != effectStormKick3 || m.worldEffects[1].effectID != effectBeginAsura1+int(level)-1 {
			t.Fatalf("level %d effects: %+v", level, m.worldEffects)
		}
		for _, e := range m.worldEffects {
			if !isLocalActor(ctx, e.actorID) {
				t.Fatal("Mild Wind belongs to the caster")
			}
		}
		if len(m.scheduledSounds) != 2 || m.scheduledSounds[0].positioned || m.scheduledSounds[0].paths[0] != "effect\\T_바람방출.wav" {
			t.Fatal("Mild Wind must play its centered sound once per use")
		}
	}
}

func TestTaekwonKickEffectsHaveNativeOwnershipAndTiming(t *testing.T) {
	ctx := archerEffectContext()
	start := time.Unix(100, 0)
	for _, tc := range []struct {
		skill   uint16
		effects map[int]uint32
	}{
		{413, map[int]uint32{}}, // The no-damage packet owns Tornado Kick.
		{415, map[int]uint32{effectPressedBody: 300, effectHitLine6: 300}},
		{417, map[int]uint32{effectSpinedBody2: 300, effectHitLine4: 300}},
		{419, map[int]uint32{effectKickedBody: 300, effectHitLine5: 300}},
		{421, map[int]uint32{effectQuakeBody2: 300, effectChemical3: 2000000, effectJumpKick: 2000000}},
	} {
		m := &WorldMode{}
		a := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Damage: 10, HitCount: 1, Action: network.ActorActionSkill}
		m.addSkillEffect(ctx, a, start, start.Add(time.Second))
		got := map[int]uint32{}
		for _, e := range m.worldEffects {
			got[e.effectID] = e.actorID
			if e.starts != start {
				t.Fatal("kick effects must start when the attack begins")
			}
			if e.effectID == effectChemical3 && (e.targetID != 300 || e.projectileDirection.x <= 0) {
				t.Fatal("Flying Kick streaks must point from caster to victim")
			}
		}
		if !reflect.DeepEqual(got, tc.effects) {
			t.Fatalf("skill %d effects: %v want %v", tc.skill, got, tc.effects)
		}
		m.worldEffects = nil
		m.addSkillHitEffect(ctx, a, start.Add(time.Second))
		if len(m.worldEffects) != 0 {
			t.Fatal("Taekwon skills must not add the ordinary hit flash")
		}
	}
}
