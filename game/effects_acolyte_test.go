package game

import (
	"github.com/kivutar/goro/client"
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

func TestAcolyteHealAmountAndOffensiveTiming(t *testing.T) {
	ctx := knightEffectContext()
	for _, tc := range []struct {
		amount uint32
		effect int
	}{
		{0, effectExit}, {199, effectExit}, {200, effectHeal},
		{1999, effectHeal}, {2000, effectHeal2}, {3999, effectHeal2}, {4000, effectHeal4},
	} {
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: db.SkillALHeal, SourceID: 2000000, TargetID: 300, Amount: tc.amount}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed Heal displayed an effect")
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.effect || m.worldEffects[0].actorID != 300 {
			t.Fatalf("Heal amount %d: %+v", tc.amount, m.worldEffects)
		}
		if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "_heal_effect.wav" {
			t.Fatal("Heal must play its sound once")
		}
	}
	for _, skill := range []uint16{db.SkillALHeal, db.SkillALHolylight} {
		m := &WorldMode{}
		start := time.Unix(100, 0)
		hit := start.Add(300 * time.Millisecond)
		action := network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Damage: 200, HitCount: 1, Action: 8}
		m.addSkillEffect(ctx, action, start, hit)
		before := 0
		if skill == db.SkillALHeal {
			before = 1
			if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectHealOffensive || !m.worldEffects[0].starts.Equal(start) {
				t.Fatal("offensive Heal must start its white effect at release")
			}
		} else if len(m.worldEffects) != 0 {
			t.Fatal("Holy Light impact appeared before the hit")
		}
		m.addSkillHitEffect(ctx, action, hit)
		impact := effectHolyLight
		if skill == db.SkillALHeal {
			impact = effectBashHit
		}
		if len(m.worldEffects) != before+2 {
			t.Fatalf("skill %d: missing native impact pair", skill)
		}
		for i, id := range []int{effectHit1, impact} {
			effect := m.worldEffects[before+i]
			if effect.effectID != id || effect.actorID != 300 || !effect.starts.Equal(hit) {
				t.Fatalf("skill %d: misplaced or mistimed impact", skill)
			}
		}
		m = &WorldMode{}
		action.Damage = 0
		m.addSkillHitEffect(ctx, action, hit)
		if len(m.worldEffects) != 0 {
			t.Fatal("miss played an impact")
		}
	}
}

func TestAcolyteSupportOwnershipAndActions(t *testing.T) {
	for _, tc := range []struct {
		skill  uint16
		effect int
		owner  uint32
	}{
		{db.SkillALIncagi, effectIncAgility, 300},
		{db.SkillALDecagi, effectDecAgility, 300},
		{db.SkillALHolywater, effectAqua, 2000000},
		{db.SkillALCrucis, effectSignum, 2000000},
		{db.SkillALAngelus, effectAngelus, 2000000},
		{db.SkillALBlessing, effectBlessing, 300},
		{db.SkillALCure, effectCure, 300},
	} {
		ctx := knightEffectContext()
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatalf("skill %d: failure displayed an effect", tc.skill)
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.effect || m.worldEffects[0].actorID != tc.owner {
			t.Fatalf("skill %d: wrong owner or extra release effect: %+v", tc.skill, m.worldEffects)
		}
		pose := spriteActionPCSkill
		if tc.skill == db.SkillALIncagi {
			pose = spriteActionIdle
		}
		if m.actorAnims[2000000].actionFamily != pose {
			t.Fatalf("skill %d: wrong release pose", tc.skill)
		}
	}
}

func TestAcolyteCastAuraAndPoses(t *testing.T) {
	for _, skill := range []uint16{
		db.SkillALRuwach, db.SkillALPneuma, db.SkillALTeleport, db.SkillALWarp,
		db.SkillALHeal, db.SkillALIncagi, db.SkillALDecagi, db.SkillALHolywater,
		db.SkillALCrucis, db.SkillALAngelus, db.SkillALBlessing, db.SkillALCure, db.SkillALHolylight,
	} {
		ctx := knightEffectContext()
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{
			SkillID: skill, SourceID: 2000000, TargetID: 300, Property: 6, DelayTime: 1000,
		})
		if !m.hasActiveWorldEffect(effectBeginSpell6, 2000000, time.Now()) || m.hasActiveWorldEffect(effectBeginSpell, 2000000, time.Now()) {
			t.Fatalf("skill %d: native cast aura must follow the holy property", skill)
		}
		pose := spriteActionPCReadyFight
		if skill == db.SkillALIncagi {
			pose = spriteActionPCSkill
		} else if skill == db.SkillALBlessing {
			pose = spriteActionIdle
		}
		if m.actorAnims[2000000].actionFamily != pose {
			t.Fatalf("skill %d: wrong casting pose", skill)
		}
	}
}

func TestAgilityNativeLabels(t *testing.T) {
	for _, decrease := range []bool{false, true} {
		label := agilityLabel(decrease, 49, 100)
		wantY, wantHeight := -52.05, 20.0
		if decrease {
			wantY, wantHeight = -42.75, 10
		}
		if label.halfWidth != 40 || label.halfHeight != wantHeight || math.Abs(label.y-wantY) > .0001 || label.alpha != 200 {
			t.Fatalf("decrease=%t: native pixel dimensions, acceleration or fade changed: %+v", decrease, label)
		}
		if agilityLabel(decrease, 0, 100).alpha >= label.alpha || agilityLabel(decrease, 99, 100).alpha != 0 {
			t.Fatal("label must fade at both ends")
		}
	}
}

func TestAquaBenedictaPlaysThroughLastFrame(t *testing.T) {
	ctx := knightEffectContext()
	spec, _ := worldEffectSpecForID(effectAqua)
	sprite := effectRenderTestSprite()
	sprite.act.Actions[0] = res.ACTAction{DelayMS: 25, Animations: make([]res.ACTAnimation, 42)}
	lastFrame := singleSpriteBillboardKey{motion: 41}
	sprite.billboards[lastFrame] = sprite.billboards[singleSpriteBillboardKey{}]
	m := &WorldMode{effectViews: map[string]*spriteView{spec.components[0].spriteFile: sprite}}
	start := time.Unix(100, 0)
	m.addWorldEffectAt(ctx, effectAqua, 300, start)
	end := start.Add(84 * db.EffectFrameDuration)
	if len(m.worldEffects) != 1 || !m.worldEffects[0].expires.Equal(end) {
		t.Fatalf("Aqua must last 42 frames of two native ticks: %+v", m.worldEffects)
	}
	for _, at := range []time.Time{end.Add(-time.Millisecond), end} {
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, newSceneProjectionForTarget(800, 600, 30.5, 20.5, 0), at, false)
		draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len()
		if (draws == 1) != at.Before(end) {
			t.Fatalf("Aqua at %s: got %d draws, must show its last frame until expiry", at.Sub(start), draws)
		}
	}
}

func TestAcolyteRealAssets(t *testing.T) {
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		t.Skip("set GORO_DATA_DIR to validate original Acolyte assets")
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
	for _, id := range []int{
		effectExit, effectHeal, effectHeal2, effectHeal4, effectHealOffensive,
		effectIncAgility, effectDecAgility, effectAqua, effectSignum, effectAngelus,
		effectBlessing, effectCure, effectHolyLight, effectRuwach, effectPneuma,
		effectTeleportation, effectReadyPortal, effectPortal,
	} {
		m := &WorldMode{}
		spec, _ := worldEffectSpecForID(id)
		for _, sound := range spec.sfx {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Fatalf("effect %d: %v", id, err)
			}
		}
		m.addWorldEffectAt(ctx, id, 300, start)
		drew := false
		for _, tick := range []int{0, 15, 30, 60, 90} {
			frame := render.NewFrame(800, 600)
			projection := newSceneProjectionForTarget(800, 600, 30.5, 20.5, 0)
			m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			value := reflect.ValueOf(frame).Elem()
			drew = drew || value.FieldByName("worldCommands").Len()+value.FieldByName("commands").Len()+value.FieldByName("worldBillboards").Len() > 0
		}
		if !drew {
			t.Fatalf("effect %d drew nothing", id)
		}
	}
}

func TestRuwachUsesSightSoundOncePerActivation(t *testing.T) {
	ctx := monkEffectContext()
	m := &WorldMode{}
	for _, state := range []uint32{db.EffectStateRuwach, db.EffectStateRuwach, db.EffectStateRuwach | db.EffectStateFalcon} {
		m.applyActorStateChange(ctx, network.ActorStateChange{ID: 300, EffectState: state})
	}
	if len(m.scheduledSounds) != 1 || !reflect.DeepEqual(m.scheduledSounds[0].paths, []string{"effect\\ef_sight.wav"}) {
		t.Fatalf("Ruwach sounds = %+v", m.scheduledSounds)
	}
	m.applyActorStateChange(ctx, network.ActorStateChange{ID: 300})
	m.applyActorStateChange(ctx, network.ActorStateChange{ID: 300, EffectState: db.EffectStateRuwach})
	if len(m.scheduledSounds) != 2 || len(m.worldEffects) != 1 {
		t.Fatal("Ruwach did not restart on a new activation")
	}
}

func TestBlessingCircleIsFlatAndUsesOriginalFade(t *testing.T) {
	spec, _ := worldEffectSpecForID(effectBlessing)
	component := spec.components[3]
	starts := time.Unix(100, 0)
	mode := WorldMode{textures: map[string]*render.Image{"__effect_alpha_down": render.NewImage(64, 64)}}
	for _, tt := range []struct {
		frame int
		alpha float64
	}{{-1, 0}, {0, 100.0 / 255 / 30}, {29, 100.0 / 255}, {119, 100.0 / 255}, {120, 100.0 / 255 * 29 / 30}, {149, 0}, {150, 0}} {
		if got := blessingCircleAlpha(component, time.Duration(tt.frame)*db.EffectFrameDuration); math.Abs(got-tt.alpha) > 1e-9 {
			t.Fatalf("frame %d alpha = %g want %g", tt.frame, got, tt.alpha)
		}
	}
	frame := render.NewFrame(800, 600)
	mode.drawBlessingCircleEffect(frame, client.Context{Resources: &res.Manager{}}, component, worldEffect{starts: starts}, 10, 20, 3, starts.Add(40*db.EffectFrameDuration))
	commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
	if commands.Len() != 1 {
		t.Fatal("missing blessing circle")
	}
	cmd := commands.Index(0)
	vertices := cmd.FieldByName("Vertices")
	for i := 0; i < vertices.Len(); i++ {
		v := vertices.Index(i)
		x, y, z := v.FieldByName("X").Float(), v.FieldByName("Y").Float(), v.FieldByName("Z").Float()
		radius := math.Hypot(x-10, z-20)
		wantRadius := 2.0
		if i%3 == 0 {
			wantRadius = 0
		}
		if y != 3 || math.Abs(radius-wantRadius) > 1e-5 {
			t.Fatalf("vertex %d is not on the ground circle: %g,%g,%g", i, x, y, z)
		}
	}
	options := cmd.FieldByName("Options")
	if options.FieldByName("DepthTest").Bool() || options.FieldByName("DepthWrite").Bool() || render.Blend(options.FieldByName("Blend").Int()) != render.BlendSourceOver {
		t.Fatal("blessing must use RF_ALPHA|RF_NODEPTHCHECK")
	}
}

func TestBlessingEffectSpecUsesRobrowserSpritesAndParticles(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectBlessing)
	if !ok {
		t.Fatal("blessing effect spec missing")
	}
	if spec.duration != 2500*time.Millisecond {
		t.Fatalf("duration = %s, want 2500ms", spec.duration)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\ef_blessing.wav" {
		t.Fatalf("sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.components))
	}
	sprite := spec.components[0]
	if sprite.kind != effectComponentSPR || sprite.spriteFile != "축복" {
		t.Fatalf("sprite component = %+v", sprite)
	}
	if sprite.duration != 1500*time.Millisecond || sprite.spriteDelay != 30*time.Millisecond || !sprite.spriteRepeat || !sprite.spriteHead || sprite.spriteYOffset != -120 || !sprite.worldSizedSprite {
		t.Fatalf("sprite timing/placement = %+v", sprite)
	}

	particleCases := []struct {
		index       int
		delay       time.Duration
		posXRand    float64
		posYRand    float64
		sparkling   bool
		sparkNumber int
	}{
		{index: 1, delay: 300 * time.Millisecond, posXRand: 1.2, posYRand: 1, sparkling: true, sparkNumber: 2},
		{index: 2, delay: 400 * time.Millisecond, posXRand: 1.4, posYRand: 1.1},
	}
	for _, tc := range particleCases {
		component := spec.components[tc.index]
		if component.kind != effectComponent3D || component.spriteFile != "particle6" {
			t.Fatalf("particle %d resource = %+v", tc.index, component)
		}
		if component.duration != 1200*time.Millisecond || component.delay != tc.delay || component.duplicateDelay != 0 || component.duplicate != 6 {
			t.Fatalf("particle %d timing = %+v", tc.index, component)
		}
		if component.alphaMax != 1 || !component.fadeIn || !component.fadeOut || component.sparkling != tc.sparkling || component.sparkNumber != tc.sparkNumber {
			t.Fatalf("particle %d fade/sparkle = %+v", tc.index, component)
		}
		if component.posXRand != tc.posXRand || component.posYRand != tc.posYRand || component.posZStartRand != 2 || component.posZStartMiddle != 5.5 || component.posZEndRand != 0.5 || component.posZEndMiddle != 1 {
			t.Fatalf("particle %d position = %+v", tc.index, component)
		}
		if component.sizeStart != 50*effectPixelRatio || component.sizeEnd != 50*effectPixelRatio {
			t.Fatalf("particle %d size = %+v", tc.index, component)
		}
	}

	aura := spec.components[3]
	if aura.funcAdapter != effectFuncBlessingCircle || aura.textureName != "alpha_down" || !aura.overlay {
		t.Fatalf("blessing ground circle = %+v", aura)
	}
}
