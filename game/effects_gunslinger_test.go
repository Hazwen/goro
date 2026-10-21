package game

import (
	"image/color"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

func TestGunslingerNativePacketOwnership(t *testing.T) {
	for skill, id := range map[uint16]int{501: 548, 502: 648, 503: 649, 507: 644, 512: 647, 514: 623, 515: 643, 518: 628, 519: 584, 520: 645} {
		ctx := archerEffectContext()
		m := &WorldMode{}
		start := time.Unix(100, 0)
		m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, HitCount: 5}, start)
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != id {
			t.Fatalf("skill %d must launch once per packet, got %+v", skill, m.worldEffects)
		}
		e := m.worldEffects[0]
		owner := uint32(300)
		if id == 623 || id == 584 || id == 645 {
			owner = 2000000
			if e.projectileDirection.x <= 0 || e.actorOrigin.x != 10.5 {
				t.Fatalf("caster direction/origin lost: %+v", e)
			}
		}
		if e.actorID != owner {
			t.Fatalf("skill %d owner=%d want %d", skill, e.actorID, owner)
		}
		m.worldEffects, m.scheduledSounds = nil, nil
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Result: 1})
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatalf("skill %d replayed on no-damage", skill)
		}
	}
	for _, tc := range []struct {
		skill uint16
		id    int
		owner uint32
		sound string
	}{
		{500, 0, 0, "effect\\플립.wav"}, {508, 0, 0, "effect\\크래커.wav"},
		{504, 625, 2000000, ""}, {505, 626, 2000000, ""}, {506, 456, 2000000, "effect\\ef_incagility.wav"},
		{513, 627, 300, "effect\\디스암.wav"}, {516, 637, 2000000, "effect\\데스페라도.wav"}, {517, 626, 2000000, ""},
	} {
		ctx := archerEffectContext()
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Result: 1}
		m.applySkillNoDamageNotify(ctx, n)
		if tc.id != 0 {
			if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.id || m.worldEffects[0].actorID != tc.owner {
				t.Fatalf("release %d: %+v", tc.skill, m.worldEffects)
			}
		} else if len(m.worldEffects) != 0 {
			t.Fatal("sound-only skill invented a visual")
		}
		if tc.sound != "" {
			if len(m.scheduledSounds) != 1 || !reflect.DeepEqual(m.scheduledSounds[0].paths, []string{tc.sound}) {
				t.Fatalf("skill %d sound %+v", tc.skill, m.scheduledSounds)
			}
		} else if len(m.scheduledSounds) != 0 {
			t.Fatalf("skill %d should be silent", tc.skill)
		}
		if tc.id == 0 && m.scheduledSounds[0].positioned {
			t.Fatal("Flip Coin/Cracker should have centered sound")
		}
		m.worldEffects, m.scheduledSounds = nil, nil
		m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}, time.Now())
		m.addSkillEffect(ctx, network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}, time.Now(), time.Now())
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatalf("release %d replayed on damage", tc.skill)
		}
		n.Result = 0
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatalf("release %d played on failure", tc.skill)
		}
	}
	// Passive accuracy/double-attack skills have no independent effect.
	for _, id := range []uint16{509, 510, 511} {
		expectEffectIDs(t, "passive", skillEffectIDs(id))
		expectEffectIDs(t, "passive hit", skillBeforeHitEffectIDs(id))
	}
}

func TestGunslingerNativePosesAndCasting(t *testing.T) {
	ctx := archerEffectContext()
	ctx.World.Player.Job = db.JobGunslinger
	for _, tc := range []struct {
		skill       uint16
		first, last int
		family      int
		duration    time.Duration
	}{
		{500, 4, 5, spriteActionPCSkill, 500 * time.Millisecond},
		{501, 0, 3, spriteActionPCSkill, time.Second}, {504, 0, 3, spriteActionPCSkill, time.Second},
		{505, 0, 3, spriteActionPCSkill, time.Second}, {506, 0, 3, spriteActionPCSkill, time.Second},
		{516, 0, 3, spriteActionPCSkill, time.Second}, {517, 0, 3, spriteActionPCSkill, time.Second},
		{508, 0, 2, spriteActionPCAttack1, 500 * time.Millisecond}, {521, 0, 2, spriteActionPCAttack1, 500 * time.Millisecond},
	} {
		for _, seed := range []int64{0, 1} {
			anim := skillAction(tc.skill).actorAnimationForActorWithResources(nil, ctx.World.Player, time.Unix(100, seed), time.Second)
			var total time.Duration
			for i := tc.first; i <= tc.last; i++ {
				frame := i
				if tc.last == 3 && seed == 1 {
					frame = 3 - i
				}
				if !anim.hasFixedMotion || anim.fixedMotion != frame || anim.actionFamily != tc.family {
					t.Fatalf("skill %d pose %d = %+v", tc.skill, frame, anim)
				}
				total += anim.duration
				if i < tc.last {
					if anim.next == nil {
						t.Fatal("missing next pose")
					}
					anim = *anim.next
				}
			}
			if total != tc.duration || anim.next != nil {
				t.Fatalf("skill %d lifetime=%s want %s", tc.skill, total, tc.duration)
			}
		}
	}
	for _, id := range []uint16{502, 503, 507, 512, 513, 514, 515, 518, 519, 520} {
		if skillAction(id).action != skillActorActionAttack {
			t.Fatalf("skill %d must use weapon attack", id)
		}
	}
	for _, id := range []uint16{503, 504, 505, 512, 513, 514, 515, 517, 518, 519, 520, 521} {
		m := &WorldMode{}
		now := time.Unix(100, 0)
		m.startSkillSourceCastAnimation(ctx, 2000000, id, 2*time.Second, now)
		a := m.actorAnims[2000000]
		family, frame := spriteActionPCSkill, 1
		if id >= 518 {
			family, frame = spriteActionPCAttack1, 0
		}
		if a.actionFamily != family || !a.hasFixedMotion || a.fixedMotion != frame || a.duration != 500*time.Millisecond {
			t.Fatalf("skill %d cast pose %+v", id, a)
		}
		m.addSkillCastEffects(ctx, id, 0, 2000000, 300, 0, 0, time.Second, now, "test")
		found := false
		for _, e := range m.worldEffects {
			found = found || e.effectID == effectBlueCasting
		}
		if !found {
			t.Fatalf("skill %d missing blue casting rings", id)
		}
	}
}

func TestGunslingerGroundAndCoinLifetime(t *testing.T) {
	ctx := archerEffectContext()
	m := &WorldMode{}
	for unit := uint16(190); unit <= 194; unit++ {
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 777, UnitID: unit, X: 12, Y: 20, Visible: true})
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != 638+int(unit-190) || !m.worldEffects[0].persistent {
			t.Fatalf("grenade %d: %+v", unit, m.worldEffects)
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 777})
		if len(m.worldEffects) != 0 {
			t.Fatal("grenade survived unit removal")
		}
	}
	ctx.World.Player.Job = db.JobGunslinger
	ctx.Session.Selected.Job = db.JobGunslinger
	for _, count := range []uint16{1, 5, 6, 10, 0} {
		if !applySpiritSphereCount(ctx, network.SpiritSphereCount{ActorID: 2000000, Count: count}) {
			t.Fatal("missing local actor")
		}
		m.syncActorSpiritEffects(ctx, 2000000)
		if count == 0 {
			if len(m.worldEffects) != 0 {
				t.Fatal("coins survived consumption")
			}
			continue
		}
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectChookgi3 || m.worldEffects[0].hitCount != int(count) || m.worldEffects[0].gunslingerCoins == nil {
			t.Fatalf("count %d: %+v", count, m.worldEffects)
		}
		e := m.worldEffects[0]
		p := e.gunslingerCoins
		m.syncActorSpiritEffects(ctx, 2000000)
		if m.worldEffects[0].gunslingerCoins != p {
			t.Fatal("unchanged count restarted coins")
		}
		stepped, jumped := newGunslingerCoins(e), newGunslingerCoins(e)
		for tick := 0; tick < 100; tick++ {
			stepped.advance(e, e.starts.Add(time.Duration(tick)*db.EffectFrameDuration))
		}
		jumped.advance(e, e.starts.Add(99*db.EffectFrameDuration))
		if !reflect.DeepEqual(stepped, jumped) {
			t.Fatal("coins depend on render frame rate")
		}
	}
}

func TestGunslingerAssetsAndTiming(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	m := &WorldMode{}
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 10, 20, 0)
	for _, tc := range []struct{ id, tick int }{{456, 30}, {584, 3}, {623, 30}, {627, 40}, {628, 30}, {629, 30}, {637, 3}, {638, 3}, {639, 3}, {640, 3}, {641, 3}, {642, 3}, {643, 3}, {644, 3}, {645, 3}, {647, 3}, {648, 32}} {
		m.worldEffects = nil
		m.scheduledSounds = nil
		if !m.addWorldEffectBetweenAt(ctx, tc.id, 300, 2000000, start) {
			t.Fatalf("missing effect %d", tc.id)
		}
		e := m.worldEffects[0]
		e.hitCount = 8
		spec, _ := worldEffectSpecForID(tc.id)
		frame := render.NewFrame(800, 600)
		for _, c := range spec.components {
			files := append([]string{}, c.textureFiles...)
			if c.textureFile != "" {
				files = append(files, c.textureFile)
			}
			for _, file := range files {
				if m.effectFileTexture(ctx.Resources, file) == nil {
					t.Fatalf("effect %d missing texture %s", tc.id, file)
				}
			}
			if c.spriteFile != "" && m.effectSpriteView(ctx.Resources, c.spriteFile) == nil {
				t.Fatalf("missing sprite %s", c.spriteFile)
			}
			m.drawFuncEffect(frame, ctx, projection, e, c, 0, 20.5, 20.5, 0, 0, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		}
		v := reflect.ValueOf(frame).Elem()
		if v.FieldByName("worldCommands").Len()+v.FieldByName("worldBillboards").Len() == 0 {
			t.Fatalf("effect %d drew nothing", tc.id)
		}
	}
	for id, tick := range map[int]int{584: 5, 623: 28, 627: 21, 637: 10, 643: 20, 644: 0, 648: 25} {
		m.scheduledSounds = nil
		m.addWorldEffectBetweenAt(ctx, id, 300, 2000000, start)
		if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].at.Sub(start) != time.Duration(tick)*db.EffectFrameDuration {
			t.Fatalf("effect %d sound timing %+v", id, m.scheduledSounds)
		}
		if id == 648 && m.scheduledSounds[0].positioned {
			t.Fatal("Triple Action sound should be centered")
		}
	}
	// Body overlays are visible for both buff colours, then disappear.
	for _, id := range []int{548, 625, 626} {
		m.worldEffects = nil
		m.addWorldEffectAt(ctx, id, 300, start)
		billboard := &spriteBillboard{image: render.NewImage(10, 20)}
		entry := sceneActorDrawEntry{actor: ctx.World.Actors[300], worldX: 20.5, worldY: 20.5}
		frame := render.NewFrame(800, 600)
		m.drawActorSpriteBodyEffects3D(ctx, frame, projection, entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(5*db.EffectFrameDuration))
		if reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() < 2 {
			t.Fatalf("body flash %d missing", id)
		}
	}
}

func TestGunslingerSpriteDelayAndGroundLoops(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	m := &WorldMode{}
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 10, 20, 0)
	for _, id := range []int{effectTripleAction, effectLightningS} {
		m.worldEffects = nil
		m.addWorldEffectAt(ctx, id, 300, start)
		if id == effectLightningS {
			m.worldEffects[0].persistent = true
			m.worldEffects[0].expires = time.Time{}
		}
		for _, tick := range []int{0, 16, 29, 30, 80, 100} {
			frame := render.NewFrame(800, 600)
			m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			n := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len()
			if id == effectTripleAction {
				if (n > 0) != (tick == 30) {
					t.Fatalf("Triple Action tick %d draws=%d", tick, n)
				}
			} else {
				want := 1
				if tick >= 16 && tick%16 < 7 {
					want = 2
				}
				if n != want {
					t.Fatalf("grenade tick %d draws=%d want %d", tick, n, want)
				}
			}
		}
	}
	ctx.Session.Selected.Job = db.JobGunslinger
	ctx.Session.Selected.Weapon = 0
	for _, sex := range []uint8{0, 1} {
		view, status := loadPlayerHumanoidSpriteView(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil {
			t.Fatalf("Gunslinger sprite missing: %s", status)
		}
		for direction := 0; direction < 8; direction++ {
			for family, minFrames := range map[int]int{spriteActionPCSkill: 6, spriteActionPCAttack1: 3} {
				_, action, ok := resolveSpriteAction(view.body.act, family, direction)
				if !ok || len(action.Animations) < minFrames {
					t.Fatalf("missing Gunslinger pose sex=%d direction=%d family=%d", sex, direction, family)
				}
			}
		}
	}
	for _, path := range []string{"effect\\플립.wav", "effect\\크래커.wav"} {
		if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []int{456, 584, 623, 627, 628, 637, 643, 644, 648} {
		spec, _ := worldEffectSpecForID(id)
		for _, path := range spec.sfx {
			if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
				t.Fatal(err)
			}
		}
	}
}
