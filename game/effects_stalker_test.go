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

func TestStalkerPacketEffects(t *testing.T) {
	ctx := archerEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobRogueH, db.JobRogueH
	for _, tc := range []struct {
		skill uint16
		ids   []int
		owner uint32
	}{
		{389, []int{db.SkillEffectChaseWalkSmoke}, 2000000},
		{390, []int{392}, 2000000},
		{475, []int{496}, 2000000},
		{476, []int{495}, 300},
	} {
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Amount: 65535}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed skill must not play effects")
		}
		n.Result = 1
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.worldEffects) != len(tc.ids) || m.actorAnims[n.SourceID].actionFamily != spriteActionPCSkill {
			t.Fatalf("skill %d: wrong effects or release pose", tc.skill)
		}
		for i, e := range m.worldEffects {
			if e.effectID != tc.ids[i] || e.actorID != tc.owner {
				t.Fatalf("skill %d: wrong packet owner: %+v", tc.skill, e)
			}
		}
		if tc.skill == 476 {
			if len(m.scheduledSounds) != 4 {
				t.Fatal("Full Strip needs four fragment sounds")
			}
			for i, sound := range m.scheduledSounds {
				if sound.at.Sub(m.worldEffects[0].starts) != time.Duration(1+10*i)*db.EffectFrameDuration || !sound.positioned || sound.paths[0] != "effect\\복호격.wav" {
					t.Fatal("wrong Full Strip sound timing")
				}
			}
		}
		m = &WorldMode{}
		m.addSkillEffect(ctx, network.ActorActionNotify{SkillID: tc.skill, SourceID: n.SourceID, TargetID: n.TargetID, Damage: 100, Action: network.ActorActionSkill}, time.Now(), time.Now())
		if len(m.worldEffects) != 0 {
			t.Fatalf("skill %d: damage packet replayed a no-damage effect", tc.skill)
		}
	}
}

func TestStalkerCastingAndSpin(t *testing.T) {
	ctx := archerEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobRogueH, db.JobRogueH
	for _, tc := range []struct {
		skill        uint16
		aura, action int
	}{{389, 501, spriteActionIdle}, {390, 16, spriteActionPCReadyFight}, {475, 59, spriteActionIdle}, {476, 16, spriteActionPCReadyFight}} {
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: tc.skill, SourceID: 2000000, DelayTime: 5000, Property: 3})
		if len(m.worldEffects) != 2 || m.worldEffects[1].effectID != tc.aura || m.actorAnims[2000000].actionFamily != tc.action {
			t.Fatalf("skill %d: wrong casting aura/pose", tc.skill)
		}
		if tc.skill != 389 {
			continue
		}
		start := m.worldEffects[1].starts
		if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\EF_BeginSpell.wav" {
			t.Fatal("Chase Walk needs its native casting sound")
		}
		for _, test := range []struct{ tick, direction int }{{0, 0}, {15, 4}, {16, 5}, {30, 2}, {40, 0}, {41, 1}, {100, 0}} {
			now := start.Add(time.Duration(test.tick) * db.EffectFrameDuration)
			if got := m.skillEffectDirection(ctx, 100, 0, now); got != test.direction {
				t.Fatalf("Chase Walk tick %d: direction %d, want %d", test.tick, got, test.direction)
			}
		}
		if ctx.World.Player.Dir != 0 {
			t.Fatal("casting spin changed authoritative facing")
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 389, SourceID: 2000000, TargetID: 2000000, Result: 1})
		if m.skillEffectDirection(ctx, 100, 0, start.Add(20*db.EffectFrameDuration)) != 0 {
			t.Fatal("casting spin survived skill release")
		}
		delete(m.actorAnims, 2000000)
		if m.skillEffectDirection(ctx, 100, 0, start.Add(20*db.EffectFrameDuration)) != 0 {
			t.Fatal("casting spin survived interrupted animation")
		}
	}
}

func TestStalkerClientAssets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobRogueH, db.JobRogueH
	for _, sex := range []uint8{0, 1} {
		view, status := loadPlayerHumanoidSpriteView(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil {
			t.Fatalf("Stalker sprite missing: %s", status)
		}
		for dir := 0; dir < 8; dir++ {
			for _, family := range []int{spriteActionIdle, spriteActionPCReadyFight, spriteActionPCSkill} {
				if _, action, ok := resolveSpriteAction(view.body.act, family, dir); !ok || len(action.Animations) == 0 {
					t.Fatal("Stalker casting/release ACT missing")
				}
			}
		}
	}
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 20, 20, 0)
	for _, tc := range []struct{ id, tick, billboards, vertices int }{
		{495, 0, 0, 0}, {495, 50, 32, 0}, {495, 109, 0, 0},
		{496, 10, 0, 180}, {496, 44, 0, 0},
		{db.SkillEffectChaseWalkSmoke, 0, 1, 0},
	} {
		m := &WorldMode{}
		if !m.addWorldEffectAt(ctx, tc.id, 300, start) {
			t.Fatalf("missing effect %d", tc.id)
		}
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		draws := reflect.ValueOf(frame).Elem()
		commands := draws.FieldByName("worldCommands")
		vertices := 0
		for i := 0; i < commands.Len(); i++ {
			vertices += commands.Index(i).FieldByName("Vertices").Len()
		}
		if tc.id == 495 && tc.tick == 50 {
			billboards := draws.FieldByName("worldBillboards")
			for i := 0; i < billboards.Len(); i++ {
				draw := billboards.Index(i)
				c := draw.FieldByName("Center")
				dx, dy, dz := c.Index(0).Float()-20.5, c.Index(1).Float()-.07, c.Index(2).Float()-20.5
				radius := math.Sqrt(dx*dx + dy*dy + dz*dz)
				if radius < 3.19 || radius > 9.01 || draw.FieldByName("Options").FieldByName("Blend").Int() != int64(render.BlendLighter) {
					t.Fatal("Full Strip fragments need native travel distance and additive blending")
				}
			}
		}
		if draws.FieldByName("worldBillboards").Len() != tc.billboards || vertices != tc.vertices {
			t.Fatalf("effect %d tick %d: billboards=%d vertices=%d", tc.id, tc.tick, draws.FieldByName("worldBillboards").Len(), vertices)
		}
	}
	guard := &WorldMode{}
	guard.addWorldEffectAt(ctx, 496, 300, start)
	before := render.NewFrame(800, 600)
	guard.drawWorldEffects(before, ctx, projection, start.Add(10*db.EffectFrameDuration), false)
	actorBefore := ctx.World.Actors[300]
	actorAfter := actorBefore
	actorAfter.Dir = (actorBefore.Dir + 4) % 8
	actorAfter.X += 1
	ctx.World.Actors[300] = actorAfter
	after := render.NewFrame(800, 600)
	guard.drawWorldEffects(after, ctx, projection, start.Add(10*db.EffectFrameDuration), false)
	for c := 0; c < 3; c++ {
		a := reflect.ValueOf(before).Elem().FieldByName("worldCommands").Index(c).FieldByName("Vertices")
		b := reflect.ValueOf(after).Elem().FieldByName("worldCommands").Index(c).FieldByName("Vertices")
		for v := 0; v < a.Len(); v++ {
			if math.Abs(b.Index(v).FieldByName("X").Float()-a.Index(v).FieldByName("X").Float()-1) > .00001 || b.Index(v).FieldByName("Z").Float() != a.Index(v).FieldByName("Z").Float() {
				t.Fatal("Preserve must follow position but keep its initial rotation")
			}
		}
	}
	ctx.World.Actors[300] = actorBefore
	for _, id := range []int{392, 495, 496, 501} {
		spec, _ := worldEffectSpecForID(id)
		for _, sound := range spec.sfx {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound); err != nil {
				t.Fatalf("effect %d sound %s: %v", id, sound, err)
			}
		}
	}
	m := &WorldMode{}
	view := m.effectSpriteView(ctx.Resources, "smoke")
	if view == nil || len(view.act.Actions[0].Animations) == 0 {
		t.Fatal("native smoke ACT missing")
	}
	m.addWorldEffectAt(ctx, db.SkillEffectChaseWalkSmoke, 300, start)
	e := m.worldEffects[0]
	if e.expires.Sub(start) != time.Duration(len(view.act.Actions[0].Animations))*(5*db.EffectFrameDuration/6) {
		t.Fatal("smoke must play once at native speed")
	}
	actor := ctx.World.Actors[300]
	actor.X += 10
	ctx.World.Actors[300] = actor
	frame := render.NewFrame(800, 600)
	m.drawWorldEffects(frame, ctx, projection, start, false)
	draw := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Index(0)
	if math.Abs(draw.FieldByName("Center").Index(0).Float()-20.5) > .001 {
		t.Fatal("smoke followed the hidden actor instead of staying at release position")
	}
}

func TestStalkerBodyFlashes(t *testing.T) {
	ctx := archerEffectContext()
	start := time.Unix(100, 0)
	img := render.NewImage(12, 24)
	img.Fill(color.White)
	for _, tc := range []struct {
		id, tick, shade int
		double, light   bool
	}{{392, 0, 100, true, false}, {392, 3, 250, true, false}, {392, 30, 255, false, false}, {496, 9, 255, false, false}, {496, 10, 250, false, true}, {496, 11, 255, false, false}, {496, 30, 250, false, true}, {496, 31, 255, false, false}} {
		for layer := 0; layer < 3; layer++ {
			m := &WorldMode{}
			m.addWorldEffectAt(ctx, tc.id, 300, start)
			frame := render.NewFrame(800, 600)
			entry := sceneActorDrawEntry{actor: ctx.World.Actors[300], worldX: 20.5, worldY: 20.5}
			billboard := &spriteBillboard{image: img, anchorX: 6, anchorY: 24, layerGroup: layer}
			m.drawActorSpriteWithEffects3D(ctx, frame, newSceneProjectionForTarget(800, 600, 20.5, 20.5, 0), entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
			draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			count := 1
			if tc.double && layer == 0 {
				count = 2
			}
			if draws.Len() != count {
				t.Fatalf("effect %d tick %d layer %d: body copies %d", tc.id, tc.tick, layer, draws.Len())
			}
			draw := draws.Index(count - 1)
			blend := render.BlendSourceOver
			if tc.light && layer == 0 {
				blend = render.BlendLighter
			}
			if draw.FieldByName("Options").FieldByName("Blend").Int() != int64(blend) || math.Abs(draw.FieldByName("ColorR").Float()-float64(tc.shade)/255) > .00001 {
				t.Fatalf("effect %d tick %d layer %d: wrong tint/blend", tc.id, tc.tick, layer)
			}
		}
	}
}
