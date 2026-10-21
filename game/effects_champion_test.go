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

func TestChampionAttacksUseReleaseEffectsAndSingleSecondaryHit(t *testing.T) {
	ctx := monkEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobMonkH, db.JobMonkH
	start := time.Unix(100, 0)
	impact := start.Add(800 * time.Millisecond)
	for _, tc := range []struct {
		skill     uint16
		owners    map[int]uint32
		wav       string
		soundTick int
	}{
		{370, map[int]uint32{376: 300, effectQuake: 2000000}, "effect\\맹호경파산.wav", 0},
		{371, map[int]uint32{377: 2000000, 263: 2000000, effectQuake: 2000000}, "effect\\mon_맹룡과강.wav", 5},
		{372, map[int]uint32{512: 2000000, 263: 2000000}, "effect\\연주붕격.wav", 44},
	} {
		m := &WorldMode{}
		a := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Action: network.ActorActionSkill, Damage: 100, HitCount: 5}
		m.addSkillEffect(ctx, a, start, impact)
		if len(m.worldEffects) != len(tc.owners) {
			t.Fatalf("skill %d: unexpected release effects %+v", tc.skill, m.worldEffects)
		}
		for _, e := range m.worldEffects {
			if e.actorID != tc.owners[e.effectID] || e.starts != start {
				t.Fatalf("skill %d: wrong effect ownership/timing %+v", tc.skill, e)
			}
			if e.effectID == 512 && (e.targetID != 300 || e.projectileDirection.x != 2 || e.actorOrigin.x != 10.5) {
				t.Fatal("Chain Crush must point from the caster toward the victim")
			}
		}
		if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != tc.wav || m.scheduledSounds[0].at.Sub(start) != time.Duration(tc.soundTick)*db.EffectFrameDuration {
			t.Fatalf("skill %d: wrong sound/timing %+v", tc.skill, m.scheduledSounds)
		}
		m.worldEffects = nil
		m.addSkillHitEffect(ctx, a, impact)
		counts := map[int]int{}
		for _, e := range m.worldEffects {
			counts[e.effectID]++
			if e.actorID != 300 || e.starts.Before(impact) || e.effectID == effectBashHit && e.starts != impact {
				t.Fatal("hit markers must stay on the victim at impact")
			}
		}
		want := map[int]int{effectHit1: 5}
		if tc.skill != 370 {
			want[effectBashHit] = 1
		}
		if !reflect.DeepEqual(counts, want) {
			t.Fatalf("skill %d: hit counts %v, want %v", tc.skill, counts, want)
		}
		m = &WorldMode{}
		a.Action = 5
		m.applyActorActionNotify(ctx, a)
		_, sourceAnimated := m.actorAnims[a.SourceID]
		if len(m.worldEffects) != 0 || sourceAnimated {
			t.Fatalf("skill %d splash: effects=%+v anims=%+v sounds=%+v", tc.skill, m.worldEffects, m.actorAnims, m.scheduledSounds)
		}
		for _, sound := range m.scheduledSounds {
			if sound.actorID != a.TargetID {
				t.Fatal("splash replayed caster sound")
			}
		}
	}
}

func TestChampionZenCastPoseAndServerOwnedSpheres(t *testing.T) {
	ctx := monkEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobMonkH, db.JobMonkH
	m := &WorldMode{}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: 401, SourceID: 2000000})
	if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != 402 {
		t.Fatal("instant Zen casting must still launch its wind")
	}
	m = &WorldMode{}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: 401, SourceID: 2000000, TargetID: 2000000, DelayTime: 4000, Property: 3})
	anim := m.actorAnims[2000000]
	if anim.actionFamily != spriteActionPCSkill || !anim.hasFixedMotion || anim.fixedMotion != 0 || anim.play {
		t.Fatalf("Zen must hold skill frame zero: %+v", anim)
	}
	if !m.hasActiveWorldEffect(effectBeginSpell3, 2000000, time.Now()) {
		t.Fatal("Zen must respect the casting property in the server packet")
	}
	portalCount := 0
	for _, e := range m.worldEffects {
		if e.effectID == 402 {
			portalCount++
			if e.actorID != 2000000 || e.expires.Sub(e.starts) != 200*db.EffectFrameDuration || !e.hasActorTransform {
				t.Fatal("Zen's wind needs its own native lifetime and a fixed origin")
			}
		}
	}
	if portalCount != 1 {
		t.Fatal("Zen did not launch its wind during casting")
	}
	m.worldEffects, m.scheduledSounds = nil, nil
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 401, SourceID: 2000000, TargetID: 2000000, Result: 1})
	if len(m.worldEffects)+len(m.scheduledSounds) != 0 || ctx.World.Player.SpiritSpheres != 0 || m.actorAnims[2000000].actionFamily != spriteActionIdle {
		t.Fatal("Zen release must stand, without replaying casting effects or inventing spheres")
	}
	for count := uint16(1); count <= 5; count++ {
		m.handleNetworkPacket(ctx, spiritSpherePacket(network.PacketZCSpirits, 2000000, count), time.Now())
	}
	if ctx.World.Player.SpiritSpheres != 5 || len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectChookgi {
		t.Fatal("Champion needs five ordinary spheres, not the baby Monk variant")
	}
	for _, skill := range []uint16{db.SkillMOCallspirits, db.SkillMOKitranslation} {
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, DelayTime: 1000})
		anim := m.actorAnims[2000000]
		if anim.actionFamily != spriteActionPCSkill || !anim.hasFixedMotion || anim.fixedMotion != 0 || anim.play {
			t.Fatal("the inherited sphere skills must use the same casting pose as Zen")
		}
	}
}

func TestChampionAttackPoses(t *testing.T) {
	ctx := monkEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobMonkH, db.JobMonkH
	for _, skill := range []uint16{370, 371, 372} {
		m := &WorldMode{}
		m.applyActorActionNotify(ctx, network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Damage: 100, HitCount: 1, SourceSpeed: 800, Action: network.ActorActionSkill})
		anim := m.actorAnims[2000000]
		switch skill {
		case 370:
			if anim.actionFamily != spriteActionPCSkill || anim.frameOffset != 3 || anim.length != 2 || !anim.play || anim.speed != 10*db.EffectFrameDuration || anim.duration != 20*db.EffectFrameDuration {
				t.Fatalf("Palm Strike must play skill frames 3 and 4 once: %+v", anim)
			}
		case 371:
			if anim.actionFamily != spriteActionPCSkill || anim.fixedMotion != 4 || !anim.hasFixedMotion || anim.play {
				t.Fatalf("Tiger Fist must hold skill frame 4: %+v", anim)
			}
		case 372:
			if anim.actionFamily != attackActionFamilyForActorWithResources(nil, ctx.World.Player) {
				t.Fatal("Chain Crush must retain the weapon's normal attack pose")
			}
		}
	}
}

func TestAsuraNativeGlyphTimingAndPalette(t *testing.T) {
	ctx := monkEffectContext()
	ctx.Resources = realDataManager(t)
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 12, 20, 0)
	for _, id := range []int{328, 510} {
		m := &WorldMode{}
		m.addWorldEffectAt(ctx, id, 300, start)
		effect := m.worldEffects[0]
		spec, _ := worldEffectSpecForID(id)
		if len(spec.components) != 2 || spec.duration != 350*db.EffectFrameDuration || len(m.scheduledSounds) != 1 {
			t.Fatal("Asura needs native rings, glyphs, and one casting sound")
		}
		for _, tick := range []int{0, 1, 21, 291} {
			frame := render.NewFrame(800, 600)
			m.drawAsuraGlyphs(frame, ctx, projection, spec.components[1], effect, start.Add(time.Duration(tick)*db.EffectFrameDuration))
			draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			count := 0
			if tick == 1 {
				count = 10
			}
			if tick == 21 {
				count = 20
			}
			shade := 0.0
			if id == 510 {
				count += count / 10
				shade = 10.0 / 255
			}
			if draws.Len() != count {
				t.Fatalf("Asura %d tick %d: %d glyph layers, want %d", id, tick, draws.Len(), count)
			}
			for i := 0; i < draws.Len(); i++ {
				draw := draws.Index(i)
				if math.Abs(draw.FieldByName("ColorR").Float()-shade) > .00001 || draw.FieldByName("Options").FieldByName("Blend").Int() != int64(render.BlendSourceOver) || math.Abs(draw.FieldByName("Center").Index(1).Float()-6.07) > .0001 {
					t.Fatal("Asura must use the 2008 dark glyph palette, normal alpha, and native height")
				}
			}
		}
	}
}

func TestChampionEffectsRenderWithNativeAssets(t *testing.T) {
	ctx := monkEffectContext()
	ctx.Resources = realDataManager(t)
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobMonkH, db.JobMonkH
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 12, 20, 0)
	for _, sex := range []uint8{0, 1} {
		view, status := loadPlayerHumanoidSpriteView(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil {
			t.Fatalf("Champion sprite missing: %s", status)
		}
		for dir := 0; dir < 8; dir++ {
			_, action, ok := resolveSpriteAction(view.body.act, spriteActionPCSkill, dir)
			if !ok || len(action.Animations) <= 4 {
				t.Fatal("Champion is missing the fixed skill poses")
			}
		}
	}
	for _, tc := range []struct{ id, tick, end int }{{376, 35, 100}, {377, 10, 50}, {402, 10, 80}, {512, 30, 125}, {263, 10, 65}} {
		m := &WorldMode{}
		m.addWorldEffectBetweenAt(ctx, tc.id, 300, 2000000, start)
		for _, sound := range m.scheduledSounds {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound.paths[0]); err != nil {
				t.Fatal(err)
			}
		}
		for _, tick := range []int{tc.tick, tc.end} {
			frame := render.NewFrame(800, 600)
			m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
			commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
			if (commands.Len() > 0) != (tick == tc.tick) {
				t.Fatalf("effect %d tick %d: commands %d", tc.id, tick, commands.Len())
			}
			if tc.id == 512 && tick == tc.tick {
				vertices := commands.Index(0).FieldByName("Vertices")
				if vertices.Len() != 32*4 {
					t.Fatal("Chain Crush needs thirty-two flat streaks")
				}
				for i := 0; i < vertices.Len(); i++ {
					if math.Abs(vertices.Index(i).FieldByName("Y").Float()-1.27) > .0001 {
						t.Fatal("Chain Crush streaks must be six native units above ground")
					}
				}
			}
		}
	}
}

func TestChampionBodyFlashWindows(t *testing.T) {
	ctx := monkEffectContext()
	start := time.Unix(100, 0)
	img := render.NewImage(12, 24)
	img.Fill(color.White)
	for _, tc := range []struct {
		id, first, last int
		tint            color.RGBA
		double          bool
	}{
		{376, 10, 20, color.RGBA{255, 120, 50, 255}, true},
		{512, 20, 100, color.RGBA{255, 255, 150, 255}, true},
		{377, 5, 35, color.RGBA{255, 255, 255, 255}, false},
		{402, 5, 65, color.RGBA{250, 250, 200, 255}, false},
	} {
		for _, tick := range []int{tc.first - 1, tc.first, tc.last, tc.last + 1} {
			for layer := 0; layer < 3; layer++ {
				m := &WorldMode{}
				m.addWorldEffectAt(ctx, tc.id, 300, start)
				frame := render.NewFrame(800, 600)
				active := tick >= tc.first && tick <= tc.last
				tint, blend, count := (color.RGBA{255, 255, 255, 255}), render.BlendSourceOver, 1
				if active {
					tint = tc.tint
					if layer == 0 {
						if tc.double {
							count = 2
						} else {
							blend = render.BlendLighter
						}
					}
				}
				m.drawActorSpriteWithEffects3D(ctx, frame, newSceneProjectionForTarget(800, 600, 12, 20, 0), sceneActorDrawEntry{actor: ctx.World.Actors[300], worldX: 12.5, worldY: 20.5}, &spriteBillboard{image: img, anchorX: 6, anchorY: 24, layerGroup: layer}, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tick)*db.EffectFrameDuration))
				draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
				if draws.Len() != count {
					t.Fatalf("effect %d tick %d layer %d: wrong body copies", tc.id, tick, layer)
				}
				draw := draws.Index(count - 1)
				for name, want := range map[string]uint8{"ColorR": tint.R, "ColorG": tint.G, "ColorB": tint.B, "ColorA": 255} {
					if math.Abs(draw.FieldByName(name).Float()-float64(want)/255) > .00001 {
						t.Fatalf("effect %d tick %d: wrong %s", tc.id, tick, name)
					}
				}
				if draw.FieldByName("Options").FieldByName("Blend").Int() != int64(blend) {
					t.Fatal("wrong body flash blend")
				}
			}
		}
	}
}
