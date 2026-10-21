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
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func TestWizardPacketEffects(t *testing.T) {
	ctx := knightEffectContext()
	start, hit := time.Unix(100, 0), time.Unix(101, 0)
	for _, tc := range []struct {
		skill              uint16
		release, secondary int
	}{
		{db.SkillWZJupitel, effectJupitelThunder, effectJupitelHit},
		{db.SkillWZWaterball, effectWaterBall2, -1},
		{db.SkillWZFrostnova, -1, effectFrostDiverHit},
		{db.SkillWZMeteor, -1, effectFireHit},
		{db.SkillWZVermilion, -1, effectWindHit},
		{db.SkillWZStormgust, -1, effectStormGust},
	} {
		for _, splash := range []bool{false, true} {
			m := &WorldMode{}
			action := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Damage: 100, HitCount: 3, Action: network.ActorActionSkill}
			if splash {
				action.Action = 5
			}
			m.addSkillBeforeHitEffect(ctx, action, start)
			m.addSkillEffect(ctx, action, start, hit)
			before := len(m.worldEffects)
			m.addSkillHitEffect(ctx, action, hit)
			if splash {
				if len(m.worldEffects) != 0 {
					t.Fatalf("skill %d replayed the spell on a splash victim", tc.skill)
				}
				continue
			}
			if tc.release < 0 {
				if before != 0 {
					t.Fatalf("skill %d launched an unexpected projectile", tc.skill)
				}
			} else if before != 1 || m.worldEffects[0].effectID != tc.release || !m.worldEffects[0].starts.Equal(start) ||
				m.worldEffects[0].actorID != action.SourceID || m.worldEffects[0].targetID != action.TargetID {
				t.Fatalf("skill %d must launch one projectile from caster to target at release", tc.skill)
			}
			want := 3
			if tc.secondary >= 0 {
				want++
			}
			if len(m.worldEffects)-before != want {
				t.Fatalf("skill %d repeated or lost an impact", tc.skill)
			}
			secondary := 0
			for _, effect := range m.worldEffects[before:] {
				if effect.actorID != 300 || effect.starts.Before(hit) {
					t.Fatalf("skill %d impact has wrong target or timing", tc.skill)
				}
				if effect.effectID == tc.secondary {
					secondary++
				}
			}
			if tc.secondary >= 0 && secondary != 1 {
				t.Fatalf("skill %d must play its secondary impact once", tc.skill)
			}
		}
	}
	for _, tc := range []struct {
		skill uint16
		id    int
	}{
		{db.SkillWZSightrasher, effectSightTrasher}, {db.SkillWZFrostnova, effectFrostDiverHit},
	} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 2000000, Result: 1})
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.id || m.worldEffects[0].actorID != 2000000 {
			t.Fatalf("skill %d must work without enemies", tc.skill)
		}
	}
	for skill := db.SkillWZFirepillar; skill <= db.SkillWZQuagmire; skill++ {
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, DelayTime: 1000, Property: 3})
		if len(m.worldEffects) != 3 || m.worldEffects[2].effectID != effectBeginSpell3 {
			t.Fatalf("skill %d must use the fire-property cast aura", skill)
		}
	}
}

func TestWizardSoundTiming(t *testing.T) {
	ctx := knightEffectContext()
	start := time.Unix(100, 0)
	m := &WorldMode{}
	m.addWorldEffectAt(ctx, effectFirePillar, 300, start)
	if len(m.scheduledSounds) != 1 || !m.scheduledSounds[0].at.Equal(start.Add(75*db.EffectFrameDuration)) {
		t.Fatal("Fire Pillar must delay its sound until the pillar erupts")
	}
	m.scheduledSounds = nil
	spec, _ := worldEffectSpecForID(effectQuagmire)
	for id := uint32(9000); id < 10200; id++ {
		m.scheduleWorldEffectSound(start, spec, worldEffect{effectID: effectQuagmire, actorID: id, starts: start, x: 15, y: 20})
	}
	if n := len(m.scheduledSounds); n < 60 || n > 140 {
		t.Fatalf("Quagmire played %d sounds for 1200 cells, expected roughly one in twelve", n)
	}
	for _, sound := range m.scheduledSounds {
		x, y, ok := scheduledSoundPosition(ctx, sound, start)
		if !ok || x != 15 || y != 20 {
			t.Fatal("Quagmire sound must stay at its cell")
		}
	}
}

func TestWizardGroundSpellOwnership(t *testing.T) {
	ctx := knightEffectContext()
	for _, tc := range []struct {
		skill, unit uint16
		id          int
	}{
		{db.SkillWZFirepillar, 135, effectFirePillarOn},
		{db.SkillWZFirepillar, 136, effectFirePillarBomb},
		{db.SkillWZIcewall, 141, effectIceWall},
		{db.SkillWZQuagmire, 142, effectQuagmire},
	} {
		m := &WorldMode{}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: tc.skill, SourceID: 2000000, X: 15, Y: 20})
		if len(m.worldEffects) != 0 {
			t.Fatalf("skill %d created an effect outside its unit lifetime", tc.skill)
		}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 900, UnitID: tc.unit, X: 15, Y: 20, Visible: true})
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.id {
			t.Fatalf("unit %d has no visual", tc.unit)
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 900})
		if len(m.worldEffects) != 0 {
			t.Fatalf("unit %d left an orphan visual", tc.unit)
		}
	}
	for _, skill := range []uint16{db.SkillWZStormgust, db.SkillWZVermilion, db.SkillWZMeteor, db.SkillWZHeavendrive} {
		m := &WorldMode{}
		for i := 0; i < 2; i++ {
			m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: skill, SourceID: 2000000, X: 15, Y: 20})
		}
		if len(m.worldEffects) != 2 || m.worldEffects[0].actorID != 0 || m.worldEffects[0].x != 15 {
			t.Fatalf("skill %d must play every ground notification at the notified cell", skill)
		}
	}
}

func TestWizardMeteorKeepsReleasePose(t *testing.T) {
	ctx := knightEffectContext()
	ctx.World.Player.Job = db.JobWizard
	ctx.World.Player.Dir = 3
	actions := make([]res.ACTAction, 13*8)
	for i := range actions {
		actions[i].Animations = make([]res.ACTAnimation, 4)
	}
	m := &WorldMode{playerView: &humanoidSpriteView{body: &spriteView{act: &res.ACT{Actions: actions}}}}
	for _, x := range []uint16{5, 30} {
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillWZMeteor, SourceID: 2000000, X: x, Y: 20})
		anim := m.actorAnims[100]
		if ctx.World.Player.Dir != 3 || anim.actionFamily != spriteActionPCSkill || anim.play || !anim.hasFixedMotion || anim.fixedMotion != 3 {
			t.Fatal("Meteor must hold the last skill frame without turning toward each impact")
		}
	}
}

func TestWizardMeteorPreservesMovement(t *testing.T) {
	for _, sourceID := range []uint32{2000000, 100, 300} {
		ctx := knightEffectContext()
		actor, _, local := actorForCombatID(ctx, sourceID)
		actor.Job, actor.Dir = db.JobWizard, 3
		actor.Moving = true
		actor.FromX, actor.FromY = 10, 20
		actor.ToX, actor.ToY = 20, 20
		actor.MovePath = []worldstate.WalkStep{{X: 10, Y: 20}, {X: 20, Y: 20}}
		actor.MoveStarted = time.Now()
		actor.MoveDuration = 10 * time.Second
		if local {
			ctx.World.Player = actor
		} else {
			ctx.World.Actors[actor.ID] = actor
		}
		m := &WorldMode{}
		for _, x := range []uint16{5, 30} {
			m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillWZMeteor, SourceID: sourceID, X: x, Y: 20})
			got := ctx.World.Player
			if !local {
				got = ctx.World.Actors[actor.ID]
			}
			if !reflect.DeepEqual(got, actor) || len(m.actorAnims) != 0 {
				t.Fatalf("meteor interrupted actor %d's movement or walking animation", sourceID)
			}
		}
		if len(m.worldEffects) != 2 {
			t.Fatal("moving caster lost the meteor effects")
		}
	}
}

func TestWizardStormGustSimplifiedEffects(t *testing.T) {
	ctx := knightEffectContext()
	m := &WorldMode{textures: map[string]*render.Image{
		"__effectfile_effect/ice.tga|blackkey=true": render.NewImage(32, 32),
	}}
	start := time.Unix(100, 0)
	spec, _ := worldEffectSpecForID(effectStormGust)
	for _, simplified := range []bool{false, true, false} {
		ctx.Session.LessEffects = simplified
		frame := render.NewFrame(800, 600)
		m.drawGroundSpikes(frame, ctx, spec.components[1], worldEffect{effectID: effectStormGust, starts: start},
			modelPoint3{}, start.Add(60*db.EffectFrameDuration))
		got := reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len()
		if (got == 0) != simplified {
			t.Fatalf("simplified=%t: got %d spike batches", simplified, got)
		}
		wantSTR := "stormgust"
		if simplified {
			wantSTR = "storm_min"
		}
		if got := resolveEffectSTRFile(spec.components[0], worldEffect{}, lessEffectsEnabled(ctx)); got != wantSTR {
			t.Fatalf("simplified=%t: STR=%q, want %q", simplified, got, wantSTR)
		}
	}
}

func TestWizardNativeGeometry(t *testing.T) {
	ctx := knightEffectContext()
	texture := render.NewImage(32, 32)
	m := &WorldMode{textures: map[string]*render.Image{
		"__effectfile_effect/ice.tga|blackkey=true":   texture,
		"__effectfile_effect/stone.bmp|blackkey=true": texture,
	}}
	start := time.Unix(100, 0)
	for _, tc := range []struct{ id, tick, spikes int }{
		{effectIceWall, 30, 3}, {effectIceWall, 10000, 3},
		{effectEarthSpike, 20, 7}, {effectHeavenDrive, 20, 25},
		{effectStormGust, 57, 0}, {effectStormGust, 58, 2}, {effectStormGust, 145, 8}, {effectStormGust, 215, 0},
	} {
		frame := render.NewFrame(800, 600)
		spec, _ := worldEffectSpecForID(tc.id)
		component := spec.components[len(spec.components)-1]
		m.drawGroundSpikes(frame, ctx, component, worldEffect{effectID: tc.id, starts: start}, modelPoint3{}, start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		cmds := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if tc.spikes == 0 {
			if cmds.Len() != 0 {
				t.Fatalf("effect %d tick %d emitted spikes too early or late", tc.id, tc.tick)
			}
			continue
		}
		if cmds.Len() != 1 || cmds.Index(0).FieldByName("Vertices").Len() != tc.spikes*12 {
			t.Fatalf("effect %d tick %d: expected %d four-sided spikes", tc.id, tc.tick, tc.spikes)
		}
		options := cmds.Index(0).FieldByName("Options")
		if options.FieldByName("AfterBillboards").Bool() != (tc.id == effectIceWall) ||
			!options.FieldByName("DepthTest").Bool() || options.FieldByName("DepthWrite").Bool() {
			t.Fatalf("effect %d: incorrect translucent spike depth/order", tc.id)
		}
		vertices := cmds.Index(0).FieldByName("Vertices")
		for i := 0; i < vertices.Len(); i += 3 {
			if vertices.Index(i+1).FieldByName("Y").Float() <= vertices.Index(i).FieldByName("Y").Float() {
				t.Fatal("native spike points down")
			}
		}
	}
}

func TestWizardFirePillarStaysUpright(t *testing.T) {
	ctx := knightEffectContext()
	m := &WorldMode{textures: map[string]*render.Image{
		"__effectfile_effect/magic_red.tga|blackkey=true": render.NewImage(32, 32),
	}}
	start := time.Unix(100, 0)
	m.addWorldEffectAtCellLifetime(ctx, effectFirePillarOn, 900, 15, 20, start, 0, true)
	projection := newSceneProjectionForTarget(800, 600, 20, 20, 0)
	for _, tick := range []int{0, 60, 300} {
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
		cmds := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if cmds.Len() != 3 {
			t.Fatal("Fire Pillar must have three spinning bands")
		}
		for i := 0; i < cmds.Len(); i++ {
			vertices := cmds.Index(i).FieldByName("Vertices")
			bottom, top := vertices.Index(0).FieldByName("Y").Float(), vertices.Index(1).FieldByName("Y").Float()
			if top <= bottom {
				t.Fatal("Fire Pillar points underground")
			}
			for j := 2; j < vertices.Len(); j += 2 {
				if math.Abs(vertices.Index(j).FieldByName("Y").Float()-bottom) > 1e-5 ||
					math.Abs(vertices.Index(j+1).FieldByName("Y").Float()-top) > 1e-5 {
					t.Fatal("Fire Pillar must rotate without tilting")
				}
			}
		}
	}
}

func TestWizardProjectilesAndSight(t *testing.T) {
	ctx := knightEffectContext()
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 20, 20, 0)
	m := &WorldMode{effectViews: map[string]*spriteView{"waterball": effectRenderTestSprite(), "sight": effectRenderTestSprite(), "data\\sprite\\shadow": effectRenderTestSprite()}}
	m.addSkillEffect(ctx, network.ActorActionNotify{SkillID: db.SkillWZWaterball, SourceID: 2000000, TargetID: 300, HitCount: 25}, start, start.Add(time.Second))
	if len(m.worldEffects) != 1 || len(m.scheduledSounds) != 1 {
		t.Fatal("Water Ball duplicated its launch")
	}
	frame := render.NewFrame(800, 600)
	m.drawWorldEffects(frame, ctx, projection, start.Add(20*db.EffectFrameDuration), false)
	if reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() != 12 {
		t.Fatal("Water Ball needs one head with its eleven trailing sprites")
	}

	m.worldEffects, m.scheduledSounds = nil, nil
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusWzSightblaster, ActorID: 2000000, Active: true})
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusWzSightblaster, ActorID: 100, Active: true})
	if len(m.worldEffects) != 1 || !m.worldEffects[0].expires.IsZero() || len(m.scheduledSounds) != 0 {
		t.Fatal("Sight Blaster must persist once, silently")
	}
	frame = render.NewFrame(800, 600)
	m.drawWorldEffects(frame, ctx, projection, m.worldEffects[0].starts.Add(time.Minute), false)
	cmds := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
	if cmds.Len() == 0 {
		t.Fatal("Sight Blaster stopped drawing")
	}
	for i := 0; i < cmds.Len(); i++ {
		if a := cmds.Index(i).FieldByName("ColorA").Float(); a <= 0 || a > 50.0/255 {
			t.Fatalf("Sight Blaster alpha wrapped: %g", a)
		}
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusWzSightblaster, ActorID: 2000000, Active: false})
	if len(m.worldEffects) != 0 {
		t.Fatal("Sight Blaster survived status removal")
	}

	m.addWorldEffectAt(ctx, effectSightTrasher, 2000000, start)
	for _, tick := range []int{0, 5, 15, 84, 85} {
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
		got := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len()
		want := 16 * (min(tick, 15)/5 + 1)
		if tick >= 84 {
			want = 0
		}
		if got != want {
			t.Fatalf("Sightrasher tick %d: %d sprites, want %d", tick, got, want)
		}
	}
}

func TestWizardJupitelTravelAndImpact(t *testing.T) {
	ctx := knightEffectContext()
	start := time.Unix(100, 0)
	m := &WorldMode{textures: make(map[string]*render.Image)}
	for _, id := range []int{effectJupitelThunder, effectJupitelHit} {
		spec, _ := worldEffectSpecForID(id)
		c := spec.components[0]
		for _, path := range append([]string{c.textureFile}, c.textureFiles...) {
			m.textures["__effectfile_"+path+"|blackkey=true"] = render.NewImage(32, 32)
		}
	}
	m.addWorldEffectBetweenAt(ctx, effectJupitelThunder, 300, 2000000, start)
	effect := m.worldEffects[0]
	projection := newSceneProjectionForTarget(800, 600, 20, 20, 0)
	for _, tick := range []int{0, 10, 19} {
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
		cmds := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if cmds.Len() != 2 {
			t.Fatal("Jupitel needs a core and animated shell")
		}
		expected := effect.actorOrigin.x + effect.projectileDirection.x*float64(tick+1)/20
		if math.Abs(cmds.Index(0).FieldByName("Center").Index(0).Float()-expected) > 1e-5 || !cmds.Index(0).FieldByName("Options").FieldByName("DepthTest").Bool() {
			t.Fatal("Jupitel must travel from the caster with depth testing")
		}
	}
	m.worldEffects = nil
	m.addWorldEffectAt(ctx, effectJupitelHit, 300, start)
	for _, tick := range []int{0, 10, 20, 240} {
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(tick)*db.EffectFrameDuration), false)
		cmds := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		want := 1
		if tick >= 20 {
			want = 2
		}
		if cmds.Len() != want {
			t.Fatalf("Jupitel impact tick %d: lost its flashes or plasma", tick)
		}
	}
}

func TestWizardRealAssets(t *testing.T) {
	ctx := knightEffectContext()
	ctx.Resources = realDataManager(t)
	start := time.Unix(100, 0)
	for _, id := range []int{effectSightTrasher, effectSightBlaster, effectIceWall, effectEarthSpike, effectHeavenDrive, effectStormGust, effectLordVermilion, effectMeteorStorm, effectJupitelThunder, effectJupitelHit, effectWaterBall2, effectQuagmire, effectFirePillar, effectFirePillarBomb, effectFirePillarOn} {
		t.Run(strconv.Itoa(id), func(t *testing.T) {
			m := &WorldMode{}
			if !m.addWorldEffectBetweenAt(ctx, id, 300, 2000000, start) {
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
