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
)

func TestPaladinAttackEffects(t *testing.T) {
	for _, tc := range []struct {
		skill  uint16
		effect int
		caster bool
	}{
		{db.SkillCRShieldcharge, effectShieldCharge, false},
		{db.SkillCRShieldboomerang, effectShieldBoomer, true},
		{db.SkillCRHolycross, effectHolyCross, false},
		{db.SkillCRGrandcross, effectGrandCross, true},
		{db.SkillPaPressure, effectPressure, false},
		{db.SkillPaSacrifice, effectBash3D, true},
		{db.SkillPaShieldchain, effectShieldBoomerang3, false},
	} {
		ctx := knightEffectContext()
		ctx.World.Player.Job = db.JobCrusaderH
		m := &WorldMode{}
		start := time.Unix(100, 0)
		hit := start.Add(300 * time.Millisecond)
		action := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Damage: 500, HitCount: 5, Action: 8}
		m.addSkillEffect(ctx, action, start, hit)
		m.addSkillBeforeHitEffect(ctx, action, start)
		owner := uint32(300)
		if tc.caster {
			owner = 2000000
		}
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.effect || m.worldEffects[0].actorID != owner || !m.worldEffects[0].starts.Equal(start) {
			t.Fatalf("skill %d must launch once at action start on %d: %+v", tc.skill, owner, m.worldEffects)
		}
		m.addSkillHitEffect(ctx, action, hit)
		if len(m.worldEffects) != 6 {
			t.Fatalf("skill %d must retain each ordinary impact, got %d", tc.skill, len(m.worldEffects))
		}
		for i, effect := range m.worldEffects[1:] {
			if effect.effectID != effectHit1 || effect.actorID != 300 || !effect.starts.Equal(hit.Add(time.Duration(i)*multiHitDelay)) {
				t.Fatalf("skill %d impact %d: %+v", tc.skill, i, effect)
			}
		}
		action.Action = 5
		m = &WorldMode{}
		m.applyActorActionNotify(ctx, action)
		if len(m.worldEffects) != 0 {
			t.Fatalf("skill %d replayed its effect on a splash victim", tc.skill)
		}
	}
}

func TestPaladinBuffOwnershipAndSacrificeActivation(t *testing.T) {
	for _, tc := range []struct {
		skill  uint16
		effect int
		target uint32
	}{
		{db.SkillCRAutoguard, effectGuard, 2000000},
		{db.SkillCRReflectshield, effectReflectShield, 2000000},
		{db.SkillCRDefender, effectCrusaderDef, 2000000},
		{db.SkillCRDevotion, effectDevotion, 2000000},
		{db.SkillCRSpearquicken, effectSpearQuicken, 2000000},
		{db.SkillCRProvidence, effectProvidence, 300},
		{db.SkillCRShrink, effectShrink, 2000000},
		{db.SkillPaSacrifice, -1, 0},
	} {
		m := &WorldMode{}
		ctx := knightEffectContext()
		notify := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects) != 0 {
			t.Fatal("failed cast played effects")
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		if tc.effect == -1 {
			if len(m.worldEffects) != 0 || len(m.scheduledSounds) != 0 {
				t.Fatal("Sacrifice activation must not play its attack effect")
			}
		} else if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.effect || m.worldEffects[0].actorID != tc.target {
			t.Fatalf("skill %d affected the wrong actor: %+v", tc.skill, m.worldEffects)
		}
	}
}

func TestDevotionSnapshotAndRemoval(t *testing.T) {
	ctx := knightEffectContext()
	m := &WorldMode{}
	m.addWorldEffectAt(ctx, effectDevotion, 2000000, time.Now())
	notify := network.ActorDevotion{SourceID: 2000000, Targets: [5]uint32{300, 300}}
	m.applyActorDevotion(ctx, notify)
	m.applyActorDevotion(ctx, notify)
	if len(m.worldEffects) != 2 || !m.worldEffects[1].expires.IsZero() || m.worldEffects[1].targetID != 300 {
		t.Fatalf("snapshots must replace links without duplicating them: %+v", m.worldEffects)
	}
	m.removeWorldEffectsForActor(300)
	if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectDevotion {
		t.Fatal("vanishing target left a link or removed the cast animation")
	}
	m.applyActorDevotion(ctx, notify)
	m.applyActorDevotion(ctx, network.ActorDevotion{SourceID: 2000000})
	if len(m.worldEffects) != 1 {
		t.Fatal("empty snapshot did not clear the caster's links")
	}
	m.applyActorDevotion(ctx, notify)
	m.removeWorldEffectsForActor(2000000)
	if len(m.worldEffects) != 0 {
		t.Fatal("vanishing caster left Devotion effects")
	}
}

func TestPaladinNativeTiming(t *testing.T) {
	ctx := knightEffectContext()
	start := time.Unix(100, 0)
	m := &WorldMode{}
	m.addWorldEffectBetweenAt(ctx, effectShieldBoomer, 300, 2000000, start)
	flight := m.worldEffects[0].shield
	origin := flight.images[0].position
	flight.advance(10, origin)
	if math.Abs(flight.images[0].position.x-origin.x-4) > 1e-6 || flight.phase != 0 {
		t.Fatal("shield must travel two native units per frame")
	}
	before := *flight
	flight.advance(10, origin)
	if *flight != before {
		t.Fatal("drawing twice advanced the shield twice")
	}
	flight.advance(499, origin)
	if flight.phase != 2 || flight.images[0].alpha != 0 {
		t.Fatal("shield did not turn back and fade")
	}
	effect := worldEffect{actorOrigin: modelPoint3{y: 2}, starts: start}
	if math.Abs(pressureImage(effect, 32).position.y-5.8) > 1e-6 || pressureImage(effect, 72).alpha != 0 {
		t.Fatal("Pressure must land at frame 32 and fade over 40 frames")
	}
	pressure, _ := worldEffectSpecForID(effectPressure)
	sacrifice, _ := worldEffectSpecForID(effectBash3D)
	if pressure.sfxDelays[0] != 3*db.EffectFrameDuration || pressure.cameraShakeDelay != 32*db.EffectFrameDuration || sacrifice.sfxDelays[0] != 22*db.EffectFrameDuration {
		t.Fatal("native audio or impact timing changed")
	}
	m.worldEffects = []worldEffect{{effectID: effectShrink, actorID: 300, starts: start}}
	base := color.RGBA{255, 255, 255, 255}
	if m.actorBodyColorTint(300, base, start.Add(6*db.EffectFrameDuration)) != (color.RGBA{250, 250, 100, 255}) ||
		m.actorBodyColorTint(300, base, start.Add(7*db.EffectFrameDuration)) != base ||
		m.actorBodyColorTint(300, base, start.Add(26*db.EffectFrameDuration)) != base {
		t.Fatal("Shrink must blink only during frames 5..25")
	}
}

func TestPaladinCastAurasAndQuickenRefresh(t *testing.T) {
	ctx := knightEffectContext()
	for _, tc := range []struct {
		skill uint16
		aura  int
	}{
		{db.SkillPaPressure, effectBeginSpell6},
		{db.SkillPaShieldchain, effectBeginSpell6},
		{db.SkillCRGrandcross, effectBeginSpell},
		{db.SkillCRProvidence, effectBeginSpell},
		{db.SkillCRDevotion, effectBeginSpell},
	} {
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: tc.skill, Property: 5, SourceID: 2000000, TargetID: 300, DelayTime: 1000})
		if !m.hasActiveWorldEffect(tc.aura, 2000000, time.Now()) || m.hasActiveWorldEffect(effectBeginSpell7, 2000000, time.Now()) {
			t.Fatalf("skill %d used the element's aura instead of its native cast effect", tc.skill)
		}
	}
	m := &WorldMode{}
	notify := network.SkillNoDamageNotify{SkillID: db.SkillCRSpearquicken, SourceID: 2000000, TargetID: 2000000, Result: 1}
	m.applySkillNoDamageNotify(ctx, notify)
	m.applySkillNoDamageNotify(ctx, notify)
	if len(m.worldEffects) != 1 {
		t.Fatal("Spear Quicken must replace its previous STR")
	}
}

func TestPressureTracksOnlyDuringItsFall(t *testing.T) {
	ctx := knightEffectContext()
	ctx.Resources = nil
	start := time.Unix(100, 0)
	actor := ctx.World.Actors[300]
	actor.Moving, actor.MoveStarted, actor.MoveDuration = true, start, 2*time.Second
	actor.FromX, actor.FromY, actor.ToX, actor.ToY = 30, 20, 50, 20
	ctx.World.Actors[300] = actor
	projection := newSceneProjectionForTarget(800, 600, 20, 20, 0)
	wantX := cellCenter(30 + 10*(31*db.EffectFrameDuration).Seconds())
	for _, ticks := range [][]int{{0, 30, 31, 33}, {0, 30, 33}, {33}} {
		m := &WorldMode{}
		m.addWorldEffectBetweenAt(ctx, effectPressure, 300, 2000000, start)
		for _, tick := range ticks {
			now := start.Add(time.Duration(tick) * db.EffectFrameDuration)
			m.drawWorldEffects(render.NewFrame(800, 600), ctx, projection, now, false)
		}
		effect := m.worldEffects[0]
		if !effect.pressureLanded || math.Abs(effect.actorOrigin.x-wantX) > 1e-6 {
			t.Fatalf("frames %v: landing X=%f, want %f at the final falling tick", ticks, effect.actorOrigin.x, wantX)
		}
		// The victim keeps moving, but the landed cross must stay put.
		m.drawWorldEffects(render.NewFrame(800, 600), ctx, projection, start.Add(time.Second), false)
		if m.worldEffects[0].actorOrigin != effect.actorOrigin {
			t.Fatal("landed cross followed the victim")
		}
	}
}

func TestShieldChainCapturesEachSpawnPosition(t *testing.T) {
	ctx := knightEffectContext()
	start := time.Unix(100, 0)
	actor := ctx.World.Actors[300]
	actor.Moving, actor.MoveStarted, actor.MoveDuration = true, start, 2*time.Second
	actor.FromX, actor.FromY, actor.ToX, actor.ToY = 30, 20, 50, 20
	ctx.World.Actors[300] = actor
	projection := newSceneProjectionForTarget(800, 600, 20, 20, 0)
	for _, ticks := range [][]int{{0, 3, 6, 9, 12}, {12}} {
		m := &WorldMode{textures: map[string]*render.Image{
			"__effectfile_effect/shield_boomerang.bmp|blackkey=true": render.NewImage(1, 1),
		}}
		m.addWorldEffectBetweenAt(ctx, effectShieldBoomerang3, 300, 2000000, start)
		for _, tick := range ticks {
			now := start.Add(time.Duration(tick) * db.EffectFrameDuration)
			m.drawWorldEffects(render.NewFrame(800, 600), ctx, projection, now, false)
		}
		effect := m.worldEffects[0]
		if effect.shieldChain.count != 5 {
			t.Fatalf("frames %v: spawned %d shields, want five", ticks, effect.shieldChain.count)
		}
		for i, origin := range effect.shieldChain.origins {
			wantX := cellCenter(30 + 10*(time.Duration(3*i)*db.EffectFrameDuration).Seconds())
			particle := shieldChainImage(effect, origin, i, 15)
			if math.Abs(particle.position.x-wantX) > 1e-6 {
				t.Fatalf("frames %v: shield %d crosses X=%f, want spawn position %f", ticks, i, particle.position.x, wantX)
			}
		}
		origins := effect.shieldChain.origins
		m.drawWorldEffects(render.NewFrame(800, 600), ctx, projection, start.Add(30*db.EffectFrameDuration), false)
		if effect.shieldChain.origins != origins {
			t.Fatal("existing shield trajectories moved with the victim")
		}
	}
}

func TestPaladinRealAssetsAndRendering(t *testing.T) {
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		t.Skip("set GORO_DATA_DIR to validate original Paladin assets")
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
	projection := newSceneProjectionForTarget(800, 600, 20, 20, 0)
	for _, id := range []int{effectGuard, effectShieldCharge, effectShieldBoomer, effectReflectShield, effectHolyCross, effectGrandCross, effectDevotion, effectProvidence, effectCrusaderDef, effectSpearQuicken, effectPressure, effectBash3D, effectBottomGospel, effectGospelGround, effectShieldBoomerang3, effectShrink, effectLineLink} {
		m := &WorldMode{}
		spec, ok := worldEffectSpecForID(id)
		if !ok {
			t.Fatalf("missing effect %d", id)
		}
		for _, component := range spec.components {
			for _, path := range append([]string{component.textureFile}, component.textureFiles...) {
				if path != "" && m.effectFileTexture(manager, path) == nil {
					t.Fatalf("effect %d missing %s", id, path)
				}
			}
			if component.kind == effectComponentFUNC && component.funcAdapter == effectFuncUnknown {
				t.Fatalf("effect %d has no renderer for %s", id, component.funcName)
			}
			if component.strFile != "" {
				str := m.loadWorldEffectSTR(manager, component.strFile, component.texturePath)
				if str == nil {
					t.Fatalf("missing STR %s", component.strFile)
				}
				for _, layer := range str.Layers {
					for _, path := range layer.Textures {
						if m.strEffectTexture(manager, path) == nil {
							t.Fatalf("missing STR texture %s", path)
						}
					}
				}
			}
		}
		for _, sound := range spec.sfx {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Fatalf("effect %d sound: %v", id, err)
			}
		}
		m.addWorldEffectBetweenAt(ctx, id, 300, 2000000, start)
		drew := false
		for _, tick := range []int{0, 6, 15, 30, 40, 60, 100, 190} {
			frame := render.NewFrame(800, 600)
			now := start.Add(time.Duration(tick) * db.EffectFrameDuration)
			m.drawWorldEffects(frame, ctx, projection, now, true)
			m.drawWorldEffects(frame, ctx, projection, now, false)
			v := reflect.ValueOf(frame).Elem()
			drew = drew || v.FieldByName("worldCommands").Len()+v.FieldByName("worldBillboards").Len() > 0
		}
		if len(spec.components) > 0 && !drew {
			t.Fatalf("effect %d submitted no geometry", id)
		}
	}
}
