package game

import (
	"math"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

func TestSageSupportEffects(t *testing.T) {
	for _, tc := range []struct {
		skill uint16
		id    int
		self  bool
	}{
		{db.SkillSAMagicrod, effectMagicRod, true},
		{db.SkillSASpellbreaker, effectSpellBreaker, false},
		{db.SkillSADispell, effectDispell, false},
		{db.SkillSAFlamelauncher, effectFlameLauncher, false},
		{db.SkillSAFrostweapon, effectFrostWeapon, false},
		{db.SkillSALightningloader, effectLightningLoad, false},
		{db.SkillSASeismicweapon, effectSeismicWeapon, false},
		{db.SkillSAElementfire, effectFlameLauncher, false},
		{db.SkillSAElementwater, effectFrostWeapon, false},
		{db.SkillSAElementwind, effectLightningLoad, false},
		{db.SkillSAElementground, effectSeismicWeapon, false},
	} {
		for _, source := range []uint32{2000000, 300} {
			ctx := knightEffectContext()
			m := &WorldMode{}
			target := uint32(300)
			if source == target {
				target = 2000000
			}
			notify := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: source, TargetID: target}
			m.applySkillNoDamageNotify(ctx, notify)
			if len(m.worldEffects) != 0 || len(m.scheduledSounds) != 0 {
				t.Fatalf("skill %d played an effect on failure", tc.skill)
			}
			notify.Result = 1
			m.applySkillNoDamageNotify(ctx, notify)
			wantActor := target
			if tc.self {
				wantActor = source
			}
			if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.id || m.worldEffects[0].actorID != wantActor {
				t.Fatalf("skill %d: wrong effect recipient: %+v", tc.skill, m.worldEffects)
			}
			if tc.id >= effectFlameLauncher && tc.id <= effectSeismicWeapon {
				if len(m.scheduledSounds) != 1 {
					t.Fatalf("skill %d: expected one endowment sound", tc.skill)
				}
				sound := m.scheduledSounds[0]
				if !sound.at.Equal(m.worldEffects[0].starts.Add(10*db.EffectFrameDuration)) ||
					len(sound.paths) != 1 || sound.paths[0] != "_enemy_hit_wind1.wav" {
					t.Fatalf("skill %d: wrong endowment sound or timing: %+v", tc.skill, sound)
				}
			}
		}
	}
	// These skills use server-driven outcomes (skills, transformations, emotions,
	// healing, crafting), without a separate activation effect in Sakexe.
	for _, skill := range []uint16{
		db.SkillSACastcancel, db.SkillSAAutospell, db.SkillSACreatecon,
		db.SkillSAAbracadabra, db.SkillSAMonocell, db.SkillSAClasschange, db.SkillSASummonmonster,
		db.SkillSAReverseorcish, db.SkillSADeath, db.SkillSAFortune, db.SkillSATamingmonster,
		db.SkillSAQuestion, db.SkillSAGravity, db.SkillSALevelup, db.SkillSAInstantdeath, db.SkillSAFullrecovery, db.SkillSAComa,
	} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(knightEffectContext(), network.SkillNoDamageNotify{
			SkillID: skill, SourceID: 2000000, TargetID: 300, Result: 1,
		})
		if len(m.worldEffects) != 0 || len(m.scheduledSounds) != 0 {
			t.Fatalf("skill %d added a non-native activation effect", skill)
		}
	}
}

func TestSageCastColors(t *testing.T) {
	for _, tc := range []struct {
		skill    uint16
		property uint32
		effect   int
	}{
		{db.SkillSAVolcano, 3, effectBeginSpell3},
		{db.SkillSADeluge, 1, effectBeginSpell2},
		{db.SkillSAViolentgale, 4, effectBeginSpell5},
		{db.SkillSALandprotector, 2, effectBeginSpell4},
		{db.SkillSAAutospell, 0, effectBeginSpell},
		{db.SkillSASpellbreaker, 0, effectBeginSpell},
		{db.SkillSADispell, 0, effectBeginSpell},
		{db.SkillSAAbracadabra, 0, effectBeginSpell},
		{db.SkillSAElementwater, 0, effectBeginSpell2},
		{db.SkillSAElementfire, 0, effectBeginSpell3},
		{db.SkillSAElementground, 0, effectBeginSpell4},
		{db.SkillSAElementwind, 0, effectBeginSpell5},
	} {
		ctx := knightEffectContext()
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{
			SkillID: tc.skill, SourceID: 2000000, TargetID: 300, DelayTime: 1000, Property: tc.property,
		})
		if len(m.worldEffects) != 3 || m.worldEffects[2].effectID != tc.effect {
			t.Fatalf("skill %d: wrong casting color for property %d", tc.skill, tc.property)
		}
	}
}

func TestSageGroundSpellOwnership(t *testing.T) {
	for _, tc := range []struct {
		skill, unit uint16
		cast, field int
	}{
		{db.SkillSAVolcano, 154, effectVolcano, effectBottomVolcano},
		{db.SkillSADeluge, 155, effectDeluge, effectBottomDeluge},
		{db.SkillSAViolentgale, 156, effectViolentGale, effectBottomViolent},
		{db.SkillSALandprotector, 157, effectLandProtector, effectBottomLand},
	} {
		ctx := knightEffectContext()
		m := &WorldMode{}
		notify := network.GroundSkillNotify{SkillID: tc.skill, SourceID: 2000000, X: 15, Y: 20}
		m.applyGroundSkillNotify(ctx, notify)
		if len(m.worldEffects) != 1 {
			t.Fatalf("skill %d: missing activation", tc.skill)
		}
		cast := m.worldEffects[0]
		if cast.effectID != tc.cast || cast.actorID != 0 || cast.x != 15 || cast.y != 20 || cast.persistent {
			t.Fatalf("skill %d: activation must stay at the selected cell", tc.skill)
		}
		if len(m.actorAnims) == 0 {
			t.Fatalf("skill %d: caster has no release animation", tc.skill)
		}
		// A recast at the same location must replay the brief activation.
		m.applyGroundSkillNotify(ctx, notify)
		if len(m.worldEffects) != 2 {
			t.Fatalf("skill %d: recast was suppressed", tc.skill)
		}
		for i := uint32(0); i < 3; i++ {
			entry := network.SkillUnitEntry{ID: 900 + i, UnitID: tc.unit, X: 15 + uint16(i), Y: 20, Visible: true}
			m.applySkillUnitEntry(ctx, entry)
			m.applySkillUnitEntry(ctx, entry)
		}
		if len(m.worldEffects) != 5 {
			t.Fatalf("skill %d: duplicated or missing field cells", tc.skill)
		}
		for i, field := range m.worldEffects[2:] {
			if field.effectID != tc.field || !field.persistent || field.x != 15+i || field.actorID != uint32(900+i) {
				t.Fatalf("skill %d: incorrect field cell: %+v", tc.skill, field)
			}
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 901})
		if len(m.worldEffects) != 4 {
			t.Fatal("removing one field cell removed its neighbors or left an orphan")
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 900})
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 902})
		if len(m.worldEffects) != 2 {
			t.Fatal("field deletion left persistent visuals")
		}
	}
}

func TestSageGroundRingTiming(t *testing.T) {
	// Compare time sampling to the native float32 PrimGI4 state machine,
	// including its initial random fade and many consecutive resets.
	for initial := 0; initial < 100; initial++ {
		radius, angle, alpha := float32(2.6), 80, initial
		for n := 1; n <= 2000; n++ {
			radius += .04
			angle--
			if angle < 10 {
				angle, alpha = 10, 0
			}
			if radius >= 2.5 {
				alpha -= 2
				if alpha <= 0 {
					alpha, radius, angle = 0, float32(2.5)-float32(.36), 74
				}
			} else {
				alpha += 20
			}
			gotRadius, gotAngle, gotAlpha := propertyGroundRing(n, initial)
			if math.Abs(gotRadius-float64(radius)) > 1e-5 || gotAngle != float64(angle) || gotAlpha != alpha {
				t.Fatalf("initial %d tick %d: got (%f,%f,%d), want (%f,%d,%d)",
					initial, n, gotRadius, gotAngle, gotAlpha, radius, angle, alpha)
			}
		}
	}
}

func TestSageGroundGeometry(t *testing.T) {
	ctx := knightEffectContext()
	start := time.Unix(100, 0)
	texture := render.NewImage(64, 64)
	m := &WorldMode{textures: map[string]*render.Image{
		"__effectfile_effect/ring_red.tga|blackkey=true":    texture,
		"__effectfile_effect/magic_green.tga|blackkey=true": texture,
		"__effectfile_effect/aaa copy.bmp|blackkey=true":    texture,
	}}
	projection := newSceneProjectionForTarget(800, 600, 0, 0, 0)
	for _, tc := range []struct {
		id, tick, vertices int
		alpha, maxU        float64
		additive           bool
	}{
		{effectVolcano, 9, 320, 200.0 / 255, 20.0 / 21, false},
		{effectVolcano, 109, 0, 0, 0, false},
		{effectBottomViolent, 58, 40, 0, 10.0 / 21, false},
		{effectBottomLand, 0, 4, 100.0 / 255, 1, true},
		{effectBottomLand, 120, 4, 100.0 / 255, 1, true},
	} {
		frame := render.NewFrame(800, 600)
		spec, _ := worldEffectSpecForID(tc.id)
		effect := worldEffect{effectID: tc.id, starts: start}
		m.drawWorldEffectComponent(frame, ctx, projection, effect, spec.components[0], 0, 0, 0, 0,
			0, spec.duration, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		cmds := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if tc.vertices == 0 {
			if cmds.Len() != 0 {
				t.Fatal("ground activation remains visible after fading")
			}
			continue
		}
		if cmds.Len() != 1 || cmds.Index(0).FieldByName("Vertices").Len() != tc.vertices {
			t.Fatalf("effect %d: wrong ring/tile geometry", tc.id)
		}
		cmd := cmds.Index(0)
		options := cmd.FieldByName("Options")
		if !options.FieldByName("DepthTest").Bool() ||
			(options.FieldByName("Blend").Int() == int64(render.BlendLighter)) != tc.additive {
			t.Fatalf("effect %d: wrong blending/depth", tc.id)
		}
		vertices := cmd.FieldByName("Vertices")
		maxU := 0.0
		for i := 0; i < vertices.Len(); i++ {
			vertex := vertices.Index(i)
			maxU = math.Max(maxU, vertex.FieldByName("SrcX").Float()/64)
			if tc.alpha != 0 && math.Abs(vertex.FieldByName("ColorA").Float()-tc.alpha) > 1e-6 {
				t.Fatalf("effect %d: wrong opacity", tc.id)
			}
			if tc.id == effectBottomLand {
				x, y, z := vertex.FieldByName("X").Float(), vertex.FieldByName("Y").Float(), vertex.FieldByName("Z").Float()
				radius := math.Hypot(x, z)
				if math.Abs(y-.4) > 1e-6 || radius < 1.12 || radius > 1.4 || math.Abs(x*z) > 1e-6 {
					t.Fatal("Land Protector must pulse as a raised diamond with native dimensions")
				}
				if i == 0 && (x != 0 || z >= 0 || vertex.FieldByName("SrcY").Float() != 0) {
					t.Fatal("Land Protector texture is flipped")
				}
			}
		}
		if math.Abs(maxU-tc.maxU) > 1e-6 {
			t.Fatalf("effect %d: texture stretched across the wrong portion of the ring", tc.id)
		}
	}
}

func TestSageRealAssets(t *testing.T) {
	ctx := knightEffectContext()
	ctx.Resources = realDataManager(t)
	start := time.Unix(100, 0)
	for _, id := range []int{
		effectVolcano, effectDeluge, effectViolentGale, effectLandProtector,
		effectBottomVolcano, effectBottomDeluge, effectBottomViolent, effectBottomLand,
		effectMagicRod, effectSpellBreaker, effectDispell,
		effectFlameLauncher, effectFrostWeapon, effectLightningLoad, effectSeismicWeapon,
	} {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			m := &WorldMode{}
			if !m.addWorldEffectAt(ctx, id, 300, start) {
				t.Fatal("missing effect")
			}
			for _, sound := range m.scheduledSounds {
				for _, path := range sound.paths {
					if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
						t.Fatal(err)
					}
				}
			}
			draws := 0
			for _, tick := range []int{0, 15, 60, 150} {
				frame := render.NewFrame(800, 600)
				m.drawWorldEffects(frame, ctx, newSceneProjectionForTarget(800, 600, 20, 20, 0), start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
				v := reflect.ValueOf(frame).Elem()
				draws += v.FieldByName("worldCommands").Len() + v.FieldByName("worldBillboards").Len() + v.FieldByName("commands").Len()
			}
			if draws == 0 {
				t.Fatal("effect produced no geometry with real assets")
			}
		})
	}
}
