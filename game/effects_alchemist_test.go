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

func TestAlchemistBerserkPotionAndCoupleCasting(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 20.5, 20.5, 0)
	m := &WorldMode{}
	m.addWorldEffectAt(ctx, effectItemFast3, 300, start)
	if len(m.scheduledSounds) != 1 || !m.scheduledSounds[0].at.Equal(start.Add(18*db.EffectFrameDuration)) {
		t.Fatal("Berserk Potion sound must coincide with the tick-18 shake")
	}
	spec, _ := worldEffectSpecForID(effectItemFast3)
	if spec.components[0].strFile != "버서크" {
		t.Fatal("Berserk Potion lost its STR")
	}
	for _, tick := range []int{0, 36, 148} {
		frame := render.NewFrame(800, 600)
		m.drawBerserkPotion(frame, ctx, spec.components[1], m.worldEffects[0], start.Add(time.Duration(tick)*db.EffectFrameDuration))
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		want := 1
		if tick == 36 {
			want = 2
		}
		if tick == 148 {
			want = 0
		}
		if commands.Len() != want {
			t.Fatalf("wrong pillar/streak lifetime at tick %d", tick)
		}
	}
	texture := render.NewImage(12, 24)
	texture.Fill(color.White)
	billboard := &spriteBillboard{image: texture, anchorX: 6, anchorY: 24}
	entry := sceneActorDrawEntry{actor: ctx.World.Actors[300], worldX: 20.5, worldY: 20.5}
	for _, tick := range []int{0, 150, 151} {
		frame := render.NewFrame(800, 600)
		m.drawActorSpriteWithEffects3D(ctx, frame, projection, entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tick)*db.EffectFrameDuration))
		draw := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Index(0)
		green := 120.0 / 255
		if tick == 151 {
			green = 1
		}
		if math.Abs(draw.FieldByName("ColorG").Float()-green) > .001 {
			t.Fatal("Berserk Potion tint must end at tick 151")
		}
	}
	m = &WorldMode{}
	m.addWorldEffectAt(ctx, effectCoupleCasting, 300, start)
	frame := render.NewFrame(800, 600)
	m.drawWorldEffects(frame, ctx, projection, start.Add(180*db.EffectFrameDuration), false)
	if m.worldEffects[0].blueCasting.tick != 181 {
		t.Fatal("standalone Couple Casting must use its declared lifetime")
	}
	commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
	if commands.Len() != 1 {
		t.Fatal("Couple Casting stopped after one second")
	}
	v := commands.Index(0).FieldByName("Vertices").Index(0)
	if math.Abs(v.FieldByName("ColorG").Float()-89.0/255) > .001 || math.Abs(v.FieldByName("ColorB").Float()-182.0/255) > .001 {
		t.Fatal("Couple Casting must use its native pink tint")
	}
}

func TestAlchemistBottlePackets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobAlchemist, db.JobAlchemist
	for level := 1; level <= 5; level++ {
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: db.SkillAMPotionpitcher, SourceID: 2000000, TargetID: 300, Amount: uint32(level)}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.worldEffects) != 0 {
			t.Fatal("failed Potion Pitcher created a bottle")
		}
		n.Result = 1
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.worldEffects) != 1 || len(m.scheduledSounds) != 0 {
			t.Fatal("Potion Pitcher must produce one silent bottle; the server owns the heal effect")
		}
		e := m.worldEffects[0]
		if e.effectID != effectThrowItem2 || e.actorID != n.SourceID || e.targetID != n.TargetID || e.itemLevel != level || m.actorAnims[n.SourceID].actionFamily != spriteActionPCAttack1 {
			t.Fatal("incorrect bottle owner, level or throwing pose")
		}
		p, _, _ := throwItemAt(e, 13)
		if math.Abs(p.x-15.7) > .001 || math.Abs(p.y-7.67) > .001 {
			t.Fatalf("incorrect half-flight arc: %+v", p)
		}
		actor := ctx.World.Actors[300]
		actor.X = 60
		ctx.World.Actors[300] = actor
		end, _, alpha := throwItemAt(e, 28)
		if math.Abs(end.x-21.3) > .001 || alpha != 235 {
			t.Fatalf("bottle must retain its destination and fade after tick 27: %+v alpha=%d", end, alpha)
		}
		actor.X = 20
		ctx.World.Actors[300] = actor
	}
	for _, damage := range []int32{0, 100} {
		m := &WorldMode{}
		n := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillAMAcidterror, Damage: damage, Action: network.ActorActionSkill, SourceSpeed: 1000}
		m.applyActorActionNotify(ctx, n)
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectThrowItem || !m.worldEffects[0].starts.Equal(m.actorAnims[n.SourceID].started) {
			t.Fatal("Acid Terror must throw at release, including misses, with no extra hit spark")
		}
		n.Action = 5
		m.applyActorActionNotify(ctx, n)
		if len(m.worldEffects) != 1 {
			t.Fatal("splash notification replayed Acid Terror")
		}
	}
	m := &WorldMode{}
	start := time.Unix(100, 0)
	m.startSkillSourceCastAnimation(ctx, 2000000, db.SkillAMBerserkpitcher, time.Second, start)
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 2000000, SkillID: db.SkillAMBerserkpitcher, Result: 1})
	if len(m.worldEffects) != 2 || m.worldEffects[0].effectID != effectThrowItem5 || m.worldEffects[0].itemLevel != 4 || m.worldEffects[1].effectID != effectItemFast3 || !m.actorAnims[2000000].started.Equal(start) {
		t.Fatal("Berserk Pitcher must keep its casting pose, throw a white bottle and buff its target")
	}
	p, _, _ := throwItemAt(m.worldEffects[0], 13)
	if math.IsNaN(p.y) || math.Abs(p.y-7.67) > .001 {
		t.Fatal("a same-cell throw must have a finite vertical arc")
	}
}

func TestAlchemistCastAndGroundOwnership(t *testing.T) {
	ctx := archerEffectContext()
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobAlchemist, db.JobAlchemist
	start := time.Unix(100, 0)
	for _, tc := range []struct {
		skill        uint16
		pose, effect int
	}{
		{db.SkillAMPharmacy, spriteActionPCReadyFight, effectBashBegin},
		{db.SkillAMDemonstration, spriteActionPCReadyFight, effectBashBegin},
		{db.SkillAMSpheremine, spriteActionPCReadyFight, effectBashBegin},
		{db.SkillAMAcidterror, spriteActionIdle, effectBeginSpell2},
		{db.SkillAMPotionpitcher, spriteActionIdle, effectBeginSpell2},
		{db.SkillAMCannibalize, spriteActionIdle, effectBeginSpell2},
		{db.SkillAMCpWeapon, spriteActionIdle, effectBeginSpell2},
		{db.SkillAMCpShield, spriteActionIdle, effectBeginSpell2},
		{db.SkillAMCpArmor, spriteActionIdle, effectBeginSpell2},
		{db.SkillAMCpHelm, spriteActionIdle, effectBeginSpell2},
		{db.SkillAMCallhomun, spriteActionIdle, effectCoupleCasting},
		{db.SkillAMRest, spriteActionIdle, effectCoupleCasting},
		{db.SkillAMResurrecthomun, spriteActionIdle, effectCoupleCasting},
		{db.SkillAMBerserkpitcher, spriteActionPCAttack1, effectBeginSpell2},
		{db.SkillAMTwilight1, spriteActionPCSkill, effectTwilight1},
		{db.SkillAMTwilight2, spriteActionPCSkill, effectTwilight2},
		{db.SkillAMTwilight3, spriteActionPCSkill, effectTwilight3},
	} {
		m := &WorldMode{}
		m.startSkillSourceCastAnimation(ctx, 2000000, tc.skill, 3*time.Second, start)
		m.addSkillCastEffects(ctx, tc.skill, 1, 2000000, 0, 0, 0, 3*time.Second, start, "test")
		if m.actorAnims[2000000].actionFamily != tc.pose || len(m.worldEffects) != 2 || m.worldEffects[1].effectID != tc.effect {
			t.Fatalf("incorrect cast pose or aura for skill %d", tc.skill)
		}
	}
	for _, skill := range []uint16{db.SkillAMDemonstration, db.SkillAMCannibalize, db.SkillAMSpheremine} {
		m := &WorldMode{}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: skill, X: 20, Y: 20})
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 || m.actorAnims[2000000].started.IsZero() {
			t.Fatal("ground notification must animate the caster and leave the visual to its unit/entity")
		}
		if skill != db.SkillAMDemonstration {
			continue
		}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: 177, X: 20, Y: 20, Visible: true})
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectDemonstration || !m.worldEffects[0].persistent {
			t.Fatal("Demonstration must belong to its persistent ground unit")
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 500})
		if len(m.worldEffects) != 0 {
			t.Fatal("Demonstration survived unit removal")
		}
	}
	for _, skill := range []uint16{db.SkillAMPharmacy, db.SkillAMTwilight1, db.SkillAMTwilight2, db.SkillAMTwilight3} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 2000000, SkillID: skill, Result: 1})
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 || m.actorAnims[2000000].started.IsZero() {
			t.Fatal("crafting completion must animate the caster without a release flare or sound")
		}
	}
}

func TestAlchemistEffectsWithClientAssets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 20, 20, 0)
	for _, skill := range []uint16{db.SkillAMCpWeapon, db.SkillAMCpShield, db.SkillAMCpArmor, db.SkillAMCpHelm} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 300, SkillID: skill, Result: 1})
		if len(m.worldEffects) != 1 || m.worldEffects[0].actorID != 300 || len(m.scheduledSounds) != 1 {
			t.Fatal("Chemical Protection must play once on the recipient")
		}
		e := m.worldEffects[0]
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, e.starts.Add(9*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if commands.Len() != 1 || commands.Index(0).FieldByName("Vertices").Len() != 64 {
			t.Fatal("Chemical Protection must draw eight double-layer rays")
		}
	}
	for _, id := range []int{effectThrowItem, effectThrowItem2, effectThrowItem5, effectCoupleCasting, effectDemonstration} {
		m := &WorldMode{}
		m.addWorldEffectBetweenAt(ctx, id, 300, 2000000, start)
		m.worldEffects[0].itemLevel = 4
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(20*db.EffectFrameDuration), false)
		v := reflect.ValueOf(frame).Elem()
		if v.FieldByName("worldBillboards").Len()+v.FieldByName("worldCommands").Len()+v.FieldByName("commands").Len() == 0 {
			t.Fatalf("effect %d did not render with client assets", id)
		}
	}
	for level, rgb := range [][3]float64{{255, 135, 135}, {255, 205, 135}, {255, 255, 135}, {255, 255, 255}, {135, 135, 255}} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillAMPotionpitcher, Amount: uint32(level + 1), Result: 1})
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, m.worldEffects[0].starts.Add(12*db.EffectFrameDuration), false)
		billboards := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if billboards.Len() != 4 {
			t.Fatal("potion must have three fading afterimages")
		}
		for j, field := range []string{"ColorR", "ColorG", "ColorB"} {
			if math.Abs(billboards.Index(0).FieldByName(field).Float()-rgb[j]/255) > .001 {
				t.Fatalf("incorrect potion color for level %d", level+1)
			}
		}
	}
	for _, id := range []int{effectTwilight1, effectTwilight2, effectTwilight3} {
		m := &WorldMode{}
		m.addWorldEffectBetweenAtDuration(ctx, id, 300, 0, start, 3*time.Second)
		e := m.worldEffects[0]
		one, many := newTwilightParticles(ctx, e), newTwilightParticles(ctx, e)
		one.advance(e, 100)
		for tick := 0; tick <= 100; tick++ {
			many.advance(e, tick)
		}
		if !reflect.DeepEqual(one, many) {
			t.Fatal("Twilight movement depends on render frequency")
		}
		for _, p := range e.twilight.particles {
			if p.position.x < -5.5 || p.position.x > 26.3 || p.position.z < 4.5 || p.position.z > 36.3 {
				t.Fatal("Twilight must surround the viewer, not the distant caster")
			}
		}
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(60*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		vertices := 0
		for i := 0; i < commands.Len(); i++ {
			vertices += commands.Index(i).FieldByName("Vertices").Len()
		}
		if vertices != 1280 || commands.Len() > 3 {
			t.Fatal("Twilight must batch its 320 ingredients by texture")
		}
		frame = render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(4*time.Second), false)
		if len(m.worldEffects) != 0 || reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() != 0 {
			t.Fatal("Twilight survived its cast")
		}
	}
}
