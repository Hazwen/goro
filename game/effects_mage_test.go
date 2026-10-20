package game

import (
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestMageCastsUseNeutralAura(t *testing.T) {
	ctx := knightEffectContext()
	for _, skill := range []uint16{db.SkillMGNapalmbeat, db.SkillMGSafetywall, db.SkillMGSoulstrike, db.SkillMGColdbolt,
		db.SkillMGFrostdiver, db.SkillMGStonecurse, db.SkillMGFireball, db.SkillMGFirewall, db.SkillMGFirebolt,
		db.SkillMGLightningbolt, db.SkillMGThunderstorm, db.SkillMGEnergycoat} {
		for _, property := range []uint32{1, 3, 4, 6, 8} {
			m := &WorldMode{}
			m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, DelayTime: 1000, Property: property})
			if len(m.worldEffects) != 3 || m.worldEffects[2].effectID != effectBeginSpell {
				t.Fatalf("skill %d property %d: expected neutral casting aura", skill, property)
			}
		}
	}
}

func TestMageSuccessAndSplashEffects(t *testing.T) {
	ctx := knightEffectContext()
	start, hit := time.Unix(100, 0), time.Unix(101, 0)
	for _, tc := range []struct {
		skill uint16
		hits  []int
	}{
		{db.SkillMGNapalmbeat, []int{effectHit1, effectBashHit}},
		{db.SkillMGSoulstrike, []int{effectHit1, effectBashHit}},
		{db.SkillMGFrostdiver, []int{effectColdHit, effectFrostDiverHit}},
		{db.SkillMGFireball, []int{effectFireHit, effectBashHit}},
		{db.SkillMGFirewall, []int{effectHit1}},
		{db.SkillMGThunderstorm, []int{effectHit1}},
	} {
		for _, splash := range []bool{false, true} {
			m := &WorldMode{}
			action := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Damage: 100, HitCount: 2, Action: network.ActorActionSkill}
			if splash {
				action.Action = 5
			}
			m.addSkillBeforeHitEffect(ctx, action, start)
			m.addSkillEffect(ctx, action, start, hit)
			before := len(m.worldEffects)
			m.addSkillHitEffect(ctx, action, hit)
			if splash && tc.skill != db.SkillMGNapalmbeat {
				if len(m.worldEffects) != 0 {
					t.Fatalf("skill %d replayed effects for a splash victim", tc.skill)
				}
				continue
			}
			impacts := m.worldEffects[before:]
			if len(impacts) != len(tc.hits)+1 {
				t.Fatalf("skill %d: expected two base impacts and one secondary impact", tc.skill)
			}
			for i, id := range tc.hits {
				if impacts[i].effectID != id || impacts[i].actorID != 300 || !impacts[i].starts.Equal(hit) {
					t.Fatalf("skill %d: wrong first impact", tc.skill)
				}
			}
			last := impacts[len(impacts)-1]
			if last.effectID != tc.hits[0] || !last.starts.Equal(hit.Add(multiHitDelay)) {
				t.Fatalf("skill %d: wrong subsequent impact", tc.skill)
			}
		}
	}
	for _, tc := range []struct {
		skill uint16
		id    int
		actor uint32
	}{
		{db.SkillMGStonecurse, effectStoneCurse, 300},
		{db.SkillMGEnergycoat, effectEnergyCoat, 2000000},
	} {
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed skill played an effect or sound")
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.id || m.worldEffects[0].actorID != tc.actor {
			t.Fatalf("skill %d: incorrect success effect", tc.skill)
		}
		if tc.skill == db.SkillMGStonecurse && (len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "_stonecurse.wav") {
			t.Fatal("Stone Curse must play its original sound on success")
		}
	}
}

func TestMageProjectileSequencesAndSounds(t *testing.T) {
	ctx := knightEffectContext()
	start := time.Unix(100, 0)
	for _, tc := range []struct {
		skill uint16
		ticks []int
	}{
		{db.SkillMGFireball, []int{20, 24, 28, 32}},
		{db.SkillMGSoulstrike, []int{6, 17, 28, 39}},
		{db.SkillMGColdbolt, []int{12}},
		{db.SkillMGFirebolt, []int{12}},
	} {
		m := &WorldMode{}
		m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, HitCount: 4}, start)
		if len(m.worldEffects) != 1 || len(m.scheduledSounds) != len(tc.ticks) {
			t.Fatalf("skill %d: expected a single sequence and %d sounds", tc.skill, len(tc.ticks))
		}
		for i, tick := range tc.ticks {
			sound := m.scheduledSounds[i]
			if !sound.at.Equal(start.Add(time.Duration(tick)*db.EffectFrameDuration)) || len(sound.paths) != 1 || strings.Contains(sound.paths[0], "%d") {
				t.Fatalf("skill %d: wrong sound or timing at tick %d", tc.skill, tick)
			}
		}
		if tc.skill == db.SkillMGColdbolt || tc.skill == db.SkillMGFirebolt {
			duration := 112 * db.EffectFrameDuration
			if tc.skill == db.SkillMGFirebolt {
				duration = 160 * db.EffectFrameDuration
			}
			if effect := m.worldEffects[0]; effect.hitCount != 4 || effect.duration != duration {
				t.Fatal("bolt sequence must preserve all projectiles")
			}
		}
	}
}

func TestMageFrostDiverTrailAndBurst(t *testing.T) {
	ctx := knightEffectContext()
	start := time.Unix(100, 0)
	m := &WorldMode{textures: map[string]*render.Image{"__effectfile_effect/ice.tga|blackkey=true": render.NewImage(32, 32)}}
	m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: db.SkillMGFrostdiver, SourceID: 2000000, TargetID: 300}, start)
	effect := m.worldEffects[0]
	if effect.actorID != 2000000 || effect.targetID != 300 || !effect.starts.Equal(start) || frostDiverTrailCount(effect) != 48 {
		t.Fatal("Frost Diver must travel from the caster to the target, starting at action time")
	}
	spec, _ := worldEffectSpecForID(effectFrostDiver)
	for _, tick := range []int{0, 10, 38, 48, 86, 87} {
		frame := render.NewFrame(800, 600)
		m.drawFrostDiverSpikes(frame, ctx, spec.components[0], effect, effect.actorOrigin, start.Add(time.Duration(tick)*db.EffectFrameDuration))
		cmds := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if tick >= 87 {
			if cmds.Len() != 0 {
				t.Fatal("ice trail survived its last spike")
			}
			continue
		}
		want := min(48, tick+1) - max(0, tick-39)
		if cmds.Len() != 1 || cmds.Index(0).FieldByName("Vertices").Len() != want*12 {
			t.Fatalf("tick %d: incorrect ice spike count", tick)
		}
	}
	spec, _ = worldEffectSpecForID(effectFrostDiverHit)
	frame := render.NewFrame(800, 600)
	m.drawFrostDiverSpikes(frame, ctx, spec.components[0], worldEffect{effectID: effectFrostDiverHit, starts: start}, effect.actorOrigin, start.Add(20*db.EffectFrameDuration))
	cmd := reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Index(0)
	vertices := cmd.FieldByName("Vertices")
	if vertices.Len() != 8*12 || render.Blend(cmd.FieldByName("Options").FieldByName("Blend").Int()) != render.BlendSourceOver {
		t.Fatal("Frost Diver impact must be eight translucent ice spikes")
	}
	for i := 0; i < vertices.Len(); i += 3 {
		base, tip := vertices.Index(i), vertices.Index(i+1)
		if tip.FieldByName("Y").Float() <= base.FieldByName("Y").Float() {
			t.Fatal("ice spike points into the ground")
		}
	}
}

func TestMageSpriteSequencesAndSightLifetime(t *testing.T) {
	ctx := knightEffectContext()
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 20, 20, 0)
	for _, tc := range []struct {
		skill uint16
		file  string
		tick  int
		draws int
	}{
		{db.SkillMGFireball, "fireball", 19, 0},
		{db.SkillMGFireball, "fireball", 20, 1},
		{db.SkillMGFireball, "fireball", 32, 4},
		{db.SkillMGFireball, "fireball", 52, 0},
		{db.SkillMGSoulstrike, "particle1", 5, 0},
		{db.SkillMGSoulstrike, "particle1", 6, 12},
		{db.SkillMGSoulstrike, "particle1", 17, 24},
	} {
		m := &WorldMode{effectViews: map[string]*spriteView{tc.file: effectRenderTestSprite()}}
		m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, HitCount: 4}, start)
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		if got := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); got != tc.draws {
			t.Fatalf("skill %d tick %d: got %d sprites, want %d", tc.skill, tc.tick, got, tc.draws)
		}
	}
	m := &WorldMode{effectViews: map[string]*spriteView{"sight": effectRenderTestSprite(), "data\\sprite\\shadow": effectRenderTestSprite()}}
	m.applyActorEffectStateEffects(ctx, 300, 0, db.EffectStateSight)
	m.applyActorEffectStateEffects(ctx, 300, db.EffectStateSight, db.EffectStateSight)
	if len(m.worldEffects) != 1 || len(m.scheduledSounds) != 1 || !m.worldEffects[0].expires.IsZero() {
		t.Fatal("Sight must persist without replaying its activation sound")
	}
	frame := render.NewFrame(800, 600)
	m.drawWorldEffects(frame, ctx, projection, m.worldEffects[0].starts.Add(time.Minute), false)
	if got := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); got != 20 {
		t.Fatalf("Sight lost its ten flames and ten shadows: %d", got)
	}
	m.applyActorEffectStateEffects(ctx, 300, db.EffectStateSight, 0)
	if len(m.worldEffects) != 0 {
		t.Fatal("Sight remained after the state ended")
	}
}

func TestMageSoulStrikeFlight(t *testing.T) {
	effect := worldEffect{actorOrigin: modelPoint3{x: 10}, projectileDirection: modelPoint3{x: 20}}
	a, b := soulStrikePosition(effect, 1, 90), soulStrikePosition(effect, 20, 90)
	if a.z <= 0 || b.z <= a.z || b.x <= a.x {
		t.Fatal("Soul Strike must curve outward while moving toward the target")
	}
	end := soulStrikePosition(effect, 40, 90)
	if math.Abs(end.x-30.7) > 1e-6 || math.Abs(end.z-.4) > 1e-6 {
		t.Fatalf("unexpected native spline endpoint: %+v", end)
	}
}

func TestMageBoltGeometryAndCadence(t *testing.T) {
	ctx := knightEffectContext()
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 20, 20, 0)
	texture := render.NewImage(32, 32)
	m := &WorldMode{textures: map[string]*render.Image{
		"__effectfile_effect/icearrow.tga|blackkey=true":  texture,
		"__effectfile_effect/ring_blue.tga|blackkey=true": texture,
	}}
	effect := worldEffect{effectID: effectColdBolt, starts: start, hitCount: 3}
	spec, _ := worldEffectSpecForID(effectColdBolt)
	firstHeight := 0.0
	for _, tick := range []int{11, 12, 22, 32} {
		frame := render.NewFrame(800, 600)
		m.drawMageBolt(frame, ctx, projection, spec.components[0], effect, modelPoint3{}, start.Add(time.Duration(tick)*db.EffectFrameDuration))
		cmds := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		want := 0
		if tick >= 12 {
			want = (tick-12)/10 + 1
		}
		if cmds.Len() != want {
			t.Fatalf("tick %d: got %d bolts, want %d", tick, cmds.Len(), want)
		}
		for i := 0; i < cmds.Len(); i++ {
			cmd := cmds.Index(i)
			vertices := cmd.FieldByName("Vertices")
			if vertices.Len() != 8 || !cmd.FieldByName("Options").FieldByName("DepthTest").Bool() ||
				render.Blend(cmd.FieldByName("Options").FieldByName("Blend").Int()) != render.BlendLighter {
				t.Fatal("bolt must be two crossed quads, with additive blending and depth testing")
			}
			height := 0.0
			for j := 0; j < vertices.Len(); j++ {
				height += vertices.Index(j).FieldByName("Y").Float() / float64(vertices.Len())
			}
			if i == 0 && tick == 12 {
				firstHeight = height
				if height < 11 || height > 12 {
					t.Fatal("bolt must start 60 native units above the target")
				}
			} else if i == 0 && height >= firstHeight-4 {
				t.Fatal("bolt did not descend between emission ticks")
			}
		}
	}
	effect.hitCount = 1
	frame := render.NewFrame(800, 600)
	m.drawMageBolt(frame, ctx, projection, spec.components[0], effect, modelPoint3{}, start.Add(60*db.EffectFrameDuration))
	cmds := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
	foundRing := false
	for i := 0; i < cmds.Len(); i++ {
		cmd := cmds.Index(i)
		if cmd.FieldByName("Vertices").Len() == 22 {
			foundRing = true
			if render.Address(cmd.FieldByName("Options").FieldByName("Address").Int()) != render.AddressRepeat {
				t.Fatal("impact ring must repeat its texture around the whole circle")
			}
		}
	}
	if !foundRing {
		t.Fatal("bolt impact ring is missing")
	}
}

func TestMageBoltLandsOnRaisedTerrain(t *testing.T) {
	ctx := knightEffectContext()
	effect := worldEffect{starts: time.Unix(100, 0)}
	origin := modelPoint3{x: 10, z: 10}
	start, direction, flatLanding := mageBoltFlight(ctx, effect, origin, 0)
	ctx.World.GAT = &res.GAT{Width: 20, Height: 20, Cells: make([]res.GATCell, 400)}
	for i := range ctx.World.GAT.Cells {
		ctx.World.GAT.Cells[i].Heights = [4]float32{4, 4, 4, 4}
	}
	_, _, raisedLanding := mageBoltFlight(ctx, effect, origin, 0)
	before := add3(origin, add3(start, mul3(direction, .55*float64(raisedLanding))))
	after := add3(before, mul3(direction, .55))
	if raisedLanding >= flatLanding || before.y < 4 || after.y >= 4 {
		t.Fatalf("bolt missed raised terrain: flat=%d raised=%d height=%g -> %g", flatLanding, raisedLanding, before.y, after.y)
	}
}

func TestMageRealAssets(t *testing.T) {
	ctx := knightEffectContext()
	ctx.Resources = realDataManager(t)
	start := time.Unix(100, 0)
	for _, id := range []int{effectSoulStrike, effectFireBall, effectFrostDiver, effectFrostDiverHit, effectFireBolt, effectColdBolt, effectSight, effectStoneCurse, effectEnergyCoat, effectSafetyWall, effectFireWall, effectThunderStorm} {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			m := &WorldMode{}
			if !m.addWorldEffectBetweenAt(ctx, id, 300, 2000000, start) {
				t.Fatal("effect unavailable")
			}
			for _, sound := range m.scheduledSounds {
				for _, path := range sound.paths {
					if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
						t.Fatal(err)
					}
				}
			}
			draws := 0
			for _, tick := range []int{0, 12, 20, 30, 45} {
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
	if _, err := ctx.Resources.ReadFile("data/wav/_stonecurse.wav"); err != nil {
		t.Fatal(err)
	}
}
