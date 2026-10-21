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

func TestPriestBuffAndOffensivePaths(t *testing.T) {
	ctx := knightEffectContext()
	start, hit := time.Unix(100, 0), time.Unix(101, 0)
	for _, tc := range []struct {
		skill uint16
		hits  []int
	}{
		{db.SkillALLResurrection, []int{effectHolyLight}},
		{db.SkillPRAspersio, []int{effectHit1, effectHolyLight}},
		{db.SkillPRBenedictio, []int{effectHit1}},
		{db.SkillPRSanctuary, []int{effectHit1}},
		{db.SkillPRTurnundead, []int{effectHit1, effectHolyLight}},
		{db.SkillPRMagnus, []int{effectHit1, effectHolyLight}},
	} {
		for _, splash := range []bool{false, true} {
			m := &WorldMode{}
			action := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Damage: 100, HitCount: 1, Action: network.ActorActionSkill}
			if splash {
				action.Action = 5
			}
			m.addSkillEffect(ctx, action, start, hit)
			before := len(m.worldEffects)
			if tc.skill == db.SkillPRTurnundead && !splash {
				if before != 1 || m.worldEffects[0].effectID != effectTurnUndead || m.worldEffects[0].actorID != 2000000 || !m.worldEffects[0].starts.Equal(start) {
					t.Fatal("Turn Undead must start its burst on the caster")
				}
				if len(m.scheduledSounds) != 1 || !m.scheduledSounds[0].at.Equal(start.Add(37*db.EffectFrameDuration)) {
					t.Fatal("Turn Undead sound must wait for tick 37")
				}
			} else if before != 0 {
				t.Fatalf("skill %d: damage replayed a buff or ground effect", tc.skill)
			}
			m.addSkillHitEffect(ctx, action, hit)
			want := tc.hits
			if splash {
				want = nil
			}
			if len(m.worldEffects)-before != len(want) {
				t.Fatalf("skill %d splash=%t: wrong impact count", tc.skill, splash)
			}
			for i, id := range want {
				effect := m.worldEffects[before+i]
				if effect.effectID != id || effect.actorID != 300 || !effect.starts.Equal(hit) {
					t.Fatalf("skill %d: misplaced impact", tc.skill)
				}
			}
		}
	}
	for _, job := range []int16{db.JobPriest, 1002} {
		actor := ctx.World.Actors[300]
		actor.Job = job
		ctx.World.Actors[300] = actor
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: db.SkillALLResurrection, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed Resurrection played effects")
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		want := 1
		if job == db.JobPriest {
			want = 2
		}
		if len(m.worldEffects) != want || len(m.scheduledSounds) != 1 {
			t.Fatalf("Resurrection on job %d: effects=%d sounds=%d", job, len(m.worldEffects), len(m.scheduledSounds))
		}
	}
}

func TestTurnUndeadDelaysImpactAndDamageTogether(t *testing.T) {
	ctx := knightEffectContext()
	for _, damage := range []int32{0, 100} {
		m := &WorldMode{}
		m.applyActorActionNotify(ctx, network.ActorActionNotify{
			SkillID: db.SkillPRTurnundead, SourceID: 2000000, TargetID: 300,
			SourceSpeed: 200, Damage: damage, HitCount: 1, Action: network.ActorActionSkill,
		})
		start := m.worldEffects[0].starts
		if len(m.damageFloaters) != 1 || m.damageFloaters[0].starts.Sub(start) != 700*time.Millisecond {
			t.Fatal("Turn Undead damage/miss must wait for the attack delay plus 500 ms")
		}
		for _, effect := range m.worldEffects[1:] {
			if !effect.starts.Equal(m.damageFloaters[0].starts) {
				t.Fatal("Turn Undead damage and impact disagree")
			}
		}
	}
}

func TestPriestGroundSpellsUseUnitLifetime(t *testing.T) {
	ctx := knightEffectContext()
	for _, tc := range []struct {
		skill, unit uint16
		effect      int
	}{
		{db.SkillMGSafetywall, 126, effectSafetyWall},
		{db.SkillMGFirewall, 127, effectFireWall},
		{db.SkillPRSanctuary, 131, effectBottomSanc},
		{db.SkillHPBasilica, 180, effectBottomBasilica},
	} {
		m := &WorldMode{}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: tc.skill, SourceID: 2000000, X: 15, Y: 20})
		if len(m.worldEffects) != 0 {
			t.Fatalf("skill %d created an unowned ground effect", tc.skill)
		}
		if tc.skill == db.SkillHPBasilica && (len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "effect\\바실리카.wav") {
			t.Fatal("Basilica must play its sound once at release")
		}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 900, UnitID: tc.unit, X: 15, Y: 20, Visible: true})
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.effect || m.worldEffects[0].actorID != 900 {
			t.Fatalf("skill %d: missing unit-owned effect", tc.skill)
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 900})
		if len(m.worldEffects) != 0 {
			t.Fatalf("skill %d left an orphan effect", tc.skill)
		}
	}
	for skill, id := range map[uint16]int{db.SkillPRBenedictio: effectBenedictio, db.SkillPRMagnus: effectMagnus} {
		m := &WorldMode{}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: skill, SourceID: 2000000, X: 15, Y: 20})
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != id || m.worldEffects[0].actorID != 0 || m.worldEffects[0].x != 15 {
			t.Fatal("ground spell must launch its STR at the selected cell")
		}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: skill, SourceID: 2000000, X: 15, Y: 20})
		if len(m.worldEffects) != 2 || len(m.scheduledSounds) != 2 {
			t.Fatal("a second cast at the same cell must play its own activation")
		}
	}
	for skill, kind := range map[uint16]damageFloaterKind{db.SkillPRSanctuary: damageFloaterRecoveryHP, db.SkillMGSrecovery: damageFloaterRecoverySP} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Result: 1, Amount: 777})
		if len(m.worldEffects)+len(m.scheduledSounds)+len(m.actorAnims) != 0 || len(m.damageFloaters) != 1 || m.damageFloaters[0].text != "777" || m.damageFloaters[0].kind != kind {
			t.Fatalf("skill %d: recovery should show the amount without a spell animation or sound", skill)
		}
	}
}

func TestPriestSupportEffectOwners(t *testing.T) {
	ctx := knightEffectContext()
	for _, tc := range []struct {
		skill  uint16
		effect int
		owner  uint32
	}{
		{db.SkillPRImpositio, effectImpositio, 300},
		{db.SkillPRSuffragium, effectSuffragium, 300},
		{db.SkillPRAspersio, effectAspersio, 300},
		{db.SkillPRSlowpoison, effectSlowPoison, 300},
		{db.SkillPRStrecovery, effectRecovery, 300},
		{db.SkillPRKyrie, effectKyrie, 2000000},
		{db.SkillPRMagnificat, effectMagnificat, 2000000},
		{db.SkillPRGloria, effectGloria, 2000000},
		{db.SkillPRLexdivina, effectLexDivina, 300},
		{db.SkillPRLexaeterna, effectLexAeterna, 300},
	} {
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatalf("skill %d: failed cast played effects", tc.skill)
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.effect || m.worldEffects[0].actorID != tc.owner {
			t.Fatalf("skill %d: effect on wrong actor", tc.skill)
		}
	}
}

func TestPriestCasting(t *testing.T) {
	ctx := knightEffectContext()
	for _, skill := range []uint16{
		db.SkillMGSafetywall, db.SkillALLResurrection, db.SkillPRImpositio, db.SkillPRSuffragium,
		db.SkillPRAspersio, db.SkillPRBenedictio, db.SkillPRSanctuary, db.SkillPRStrecovery,
		db.SkillPRKyrie, db.SkillPRMagnificat, db.SkillPRGloria, db.SkillPRLexdivina,
		db.SkillPRTurnundead, db.SkillPRLexaeterna, db.SkillPRMagnus,
		db.SkillHPAssumptio, db.SkillHPBasilica, db.SkillPRRedemptio,
	} {
		m := &WorldMode{}
		// Ordinary auras follow the property; High Priest and Redemptio stay white.
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, DelayTime: 1000, Property: 3})
		aura := effectBeginSpell3
		if skill == db.SkillHPAssumptio || skill == db.SkillHPBasilica || skill == db.SkillPRRedemptio {
			aura = effectBeginSpell6
		}
		if !m.hasActiveWorldEffect(aura, 2000000, time.Now()) {
			t.Fatalf("skill %d: wrong cast aura", skill)
		}
		pose := spriteActionPCReadyFight
		if skill == db.SkillPRRedemptio {
			pose = spriteActionIdle
		}
		if m.actorAnims[2000000].actionFamily != pose {
			t.Fatalf("skill %d: wrong cast pose", skill)
		}
	}
}

func TestAssumptioFollowsStatusInEitherPacketOrder(t *testing.T) {
	for _, statusFirst := range []bool{false, true} {
		ctx := monkEffectContext()
		m := &WorldMode{}
		status := network.StatusEffectChange{StatusID: db.StatusAssumptio, ActorID: 300, Active: true}
		if statusFirst {
			m.applyStatusEffectChange(ctx, status)
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillHPAssumptio, SourceID: 2000000, TargetID: 300, Result: 1})
		m.applyStatusEffectChange(ctx, status)
		m.applyStatusEffectChange(ctx, status)
		if len(m.worldEffects) != 2 || len(m.scheduledSounds) != 1 || !m.hasActiveWorldEffect(effectAssumptio, 300, time.Now().Add(time.Hour)) {
			t.Fatal("Assumptio must have one persistent glow and one activation")
		}
		next := &WorldMode{}
		next.syncCurrentActorEffectStateEffects(ctx)
		if len(next.worldEffects) != 1 || next.worldEffects[0].effectID != effectAssumptio || len(next.scheduledSounds) != 0 {
			t.Fatal("actor entry should restore the glow without replaying activation")
		}
		status.Active = false
		m.applyStatusEffectChange(ctx, status)
		if m.hasActiveWorldEffect(effectAssumptio, 300, time.Now()) {
			t.Fatal("Assumptio glow survived status removal")
		}
	}
}

func TestAssumptioOnlyGlowsBodyLayer(t *testing.T) {
	ctx := monkEffectContext()
	m := &WorldMode{}
	start := time.Now()
	m.addWorldEffectAt(ctx, effectAssumptio, 300, start)
	for _, group := range []int{0, 1, 2} {
		frame := render.NewFrame(800, 600)
		billboard := &spriteBillboard{image: render.NewImage(12, 24), anchorX: 6, anchorY: 24, layerGroup: group}
		entry := sceneActorDrawEntry{actor: ctx.World.Actors[300], worldX: 12, worldY: 20}
		m.drawActorSpriteWithEffects3D(ctx, frame, newSceneProjectionForTarget(800, 600, 12, 20, 0), entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start)
		count := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len()
		want := 1
		if group == 0 {
			want = 2
		}
		if count != want {
			t.Fatalf("layer %d: got %d draws, want %d", group, count, want)
		}
	}
}

func TestPriestRealAssets(t *testing.T) {
	ctx := knightEffectContext()
	ctx.Resources = realDataManager(t)
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 30.5, 20.5, 0)
	for _, id := range []int{
		effectTurnUndead, effectRevive, effectResurrection, effectAssumptio2,
		effectImpositio, effectSuffragium, effectAspersio, effectBenedictio,
		effectKyrie, effectMagnificat, effectGloria, effectLexDivina,
		effectRecovery, effectSlowPoison, effectLexAeterna, effectMagnus,
		effectSafetyWall, effectBottomSanc, effectBottomMagnus, effectBottomBasilica,
	} {
		m := &WorldMode{}
		spec, _ := worldEffectSpecForID(id)
		for _, sound := range spec.sfx {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound); err != nil {
				t.Fatalf("effect %d: %v", id, err)
			}
		}
		m.addWorldEffectAt(ctx, id, 300, start)
		drew := false
		for _, tick := range []int{0, 15, 50, 60, 90} {
			frame := render.NewFrame(800, 600)
			m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			value := reflect.ValueOf(frame).Elem()
			drew = drew || value.FieldByName("commands").Len()+value.FieldByName("worldCommands").Len()+value.FieldByName("worldBillboards").Len() > 0
		}
		if !drew || len(m.textureMiss) != 0 {
			t.Fatalf("effect %d drew nothing or has missing textures: %v", id, m.textureMiss)
		}
	}
	if _, err := ctx.Resources.ReadFile("data/wav/effect/바실리카.wav"); err != nil {
		t.Fatal(err)
	}
}

func TestTurnUndeadPhases(t *testing.T) {
	ctx := knightEffectContext()
	m := &WorldMode{textures: map[string]*render.Image{
		"__effectfile_effect/alpha_down.tga|blackkey=true":   render.NewImage(16, 16),
		"__effectfile_effect/alpha_center.tga|blackkey=true": render.NewImage(16, 16),
		"__effectfile_effect/ring_liner.tga|blackkey=true":   render.NewImage(16, 16),
	}}
	spec, _ := worldEffectSpecForID(effectTurnUndead)
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 10, 20, 0)
	for _, tc := range []struct{ tick, screenDraws, worldDraws int }{
		{-1, 0, 0}, {0, 2, 0}, {19, 2, 0}, {20, 0, 0}, {49, 0, 0}, {50, 0, 1}, {65, 0, 1}, {80, 0, 0},
	} {
		frame := render.NewFrame(800, 600)
		m.drawTurnUndeadEffect(frame, ctx, projection, spec.components[0], worldEffect{starts: start}, 10, 20, 3, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		value := reflect.ValueOf(frame).Elem()
		world := value.FieldByName("worldCommands")
		if value.FieldByName("commands").Len() != tc.screenDraws || world.Len() != tc.worldDraws {
			t.Fatalf("tick %d: wrong Turn Undead phase", tc.tick)
		}
		if world.Len() > 0 {
			vertices := world.Index(0).FieldByName("Vertices")
			for i := 0; i < vertices.Len(); i++ {
				if math.Abs(vertices.Index(i).FieldByName("Y").Float()-4.4) > 1e-5 {
					t.Fatal("Turn Undead ring must lie horizontally above the caster's feet")
				}
			}
		}
	}
}

func TestPriestGroundGeometryAndBlending(t *testing.T) {
	ctx := knightEffectContext()
	start := time.Unix(100, 0)
	for _, tc := range []struct {
		id      int
		half    float64
		sides   int
		alpha   float64
		blend   render.Blend
		texture string
	}{
		{effectBottomSanc, .5, 4, 120.0 / 255, render.BlendSourceOver, "effect/alpha_down.tga"},
		{effectBottomMagnus, 1, 4, 30.0 / 255, render.BlendLighter, "effect/ring_red.tga"},
		{effectBottomBasilica, 2.96, 24, 65.0 / 255, render.BlendLighter, "effect/alpha_down.tga"},
	} {
		m := &WorldMode{textures: map[string]*render.Image{
			"__effectfile_" + tc.texture + "|blackkey=true": render.NewImage(16, 16),
		}}
		spec, _ := worldEffectSpecForID(tc.id)
		frame := render.NewFrame(800, 600)
		m.drawPriestGroundEffect(frame, ctx, spec.components[0], worldEffect{effectID: tc.id, starts: start},
			10, 20, 3, start.Add(89*db.EffectFrameDuration))
		cmds := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if cmds.Len() != 1 {
			t.Fatalf("effect %d: expected one batch", tc.id)
		}
		cmd := cmds.Index(0)
		vertices := cmd.FieldByName("Vertices")
		if vertices.Len() != tc.sides*4 || render.Blend(cmd.FieldByName("Options").FieldByName("Blend").Int()) != tc.blend {
			t.Fatalf("effect %d: wrong face count (%d) or blend", tc.id, vertices.Len()/4)
		}
		minY := math.Inf(1)
		maxRadius := 0.0
		for i := 0; i < vertices.Len(); i++ {
			v := vertices.Index(i)
			x, y, z := v.FieldByName("X").Float(), v.FieldByName("Y").Float(), v.FieldByName("Z").Float()
			maxRadius = math.Max(maxRadius, math.Max(math.Abs(x-10), math.Abs(z-20)))
			minY = math.Min(minY, y)
		}
		if math.Abs(maxRadius-tc.half) > 1e-5 || math.Abs(minY-3.2) > 1e-5 {
			t.Fatalf("effect %d: wrong footprint or elevation (%g, %g)", tc.id, maxRadius, minY)
		}
		if math.Abs(vertices.Index(0).FieldByName("ColorA").Float()-tc.alpha) > 1e-5 {
			t.Fatalf("effect %d: wrong wall opacity", tc.id)
		}
	}
}
