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

func TestNinjaBeforeHitRouting(t *testing.T) {
	for skill, id := range map[uint16]int{523: 613, 524: 614, 525: 615, 526: 616, 528: 632, 530: 630, 534: 618, 537: 619, 540: 621, 542: 624, 544: 633} {
		ctx := archerEffectContext()
		m := &WorldMode{}
		start := time.Unix(100, 0)
		m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: skill, SkillLevel: 10, HitCount: 3, SourceID: 2000000, TargetID: 300}, start)
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != id {
			t.Fatalf("skill %d needs one launch, got %+v", skill, m.worldEffects)
		}
		e := m.worldEffects[0]
		if id >= 613 && id <= 616 {
			if e.actorID != 2000000 || e.targetID != 300 || !e.hasActorTransform || e.projectileDirection.x <= 0 || len(m.scheduledSounds) != 1 || m.scheduledSounds[0].positioned {
				t.Fatalf("throw must fly from caster to target with centered sound: %+v", e)
			}
		}
		if id == 621 && (e.hitCount != 3 || e.actorID != 2000000 || e.targetID != 300 || e.actorOrigin.x != 10.5 || e.ninjaWind == nil) {
			t.Fatalf("wind must launch from caster with packet hit count: %+v", e)
		}
		if id == 618 && (e.ninjaPetals == nil || len(m.scheduledSounds) != 4 || m.scheduledSounds[0].at.Sub(start) != 12*db.EffectFrameDuration) {
			t.Fatal("Fire Petal needs drifting petals and four delayed sounds")
		}
		m.worldEffects, m.scheduledSounds = nil, nil
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Result: 1})
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatalf("skill %d duplicated its damage effect on no-damage packet", skill)
		}
	}
	expectEffectIDs(t, "Fire Petal impact", skillHitEffectIDs(534), effectFireHit)
	expectEffectIDs(t, "Ice Spear impact", skillHitEffectIDs(537), effectColdHit)
}

func TestNinjaReleaseAndGroundOwnership(t *testing.T) {
	for _, tc := range []struct {
		skill  uint16
		id     int
		owner  uint32
		sounds int
	}{{527, 0, 0, 1}, {536, 635, 300, 1}, {539, 636, 2000000, 1}, {541, 622, 2000000, 1}, {531, 0, 0, 0}, {532, 0, 0, 0}, {543, 0, 0, 0}, {529, 0, 0, 0}} {
		ctx := archerEffectContext()
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Result: 1})
		if len(m.scheduledSounds) != tc.sounds {
			t.Fatalf("skill %d sounds=%d", tc.skill, len(m.scheduledSounds))
		}
		if tc.id == 0 {
			if len(m.worldEffects) != 0 {
				t.Fatalf("skill %d invented an effect", tc.skill)
			}
		} else if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.id || m.worldEffects[0].actorID != tc.owner {
			t.Fatalf("skill %d wrong effect owner: %+v", tc.skill, m.worldEffects)
		}
		if tc.skill == 527 && m.scheduledSounds[0].positioned {
			t.Fatal("Tatami's sound is centered")
		}
		m.worldEffects, m.scheduledSounds = nil, nil
		m.addSkillEffect(ctx, network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}, time.Now(), time.Now())
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatalf("skill %d replayed its release on damage", tc.skill)
		}
	}
	for _, tc := range []struct {
		skill, unit uint16
		id          int
	}{{527, 188, 631}, {535, 189, 634}, {538, 187, 620}} {
		ctx := archerEffectContext()
		m := &WorldMode{}
		expectEffectIDs(t, "ground is unit-owned", skillGroundEffectIDs(tc.skill))
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 777, UnitID: tc.unit, X: 12, Y: 20, Visible: true})
		if len(m.worldEffects) != 1 || !m.worldEffects[0].persistent || m.worldEffects[0].effectID != tc.id || (tc.unit != 189 && len(m.scheduledSounds) != 0) {
			t.Fatalf("unit %d ownership: %+v", tc.unit, m.worldEffects)
		}
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 777})
		if len(m.worldEffects) != 0 {
			t.Fatal("ground effect outlived its server unit")
		}
	}
}

func TestNinjaPosesAndMirrorImage(t *testing.T) {
	for _, skill := range []uint16{523, 527, 528, 534, 537, 540, 543} {
		spec := skillAction(skill)
		first, last, duration := 0, 1, 500*time.Millisecond
		if skill == 523 || skill == 534 || skill == 540 || skill == 543 {
			first, last, duration = 2, 5, time.Second
		}
		total := time.Duration(0)
		for frame := first; frame <= last; frame++ {
			if spec.action != skillActorActionSkill || !spec.hasFrame || spec.frame != frame || spec.play {
				t.Fatalf("skill %d wrong ACT pose %+v", skill, spec)
			}
			total += spec.duration
			if frame < last {
				if spec.next == nil {
					t.Fatal("missing pose")
				}
				spec = *spec.next
			}
		}
		if total != duration || spec.next != nil {
			t.Fatalf("skill %d wrong animation lifetime", skill)
		}
	}
	ctx := archerEffectContext()
	m := &WorldMode{}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{ActorID: 2000000, StatusID: db.StatusNjBunsinjyutsu, Active: true})
	entry := sceneActorDrawEntry{actor: ctx.World.Player, isPlayer: true, worldX: 10.5, worldY: 20.5}
	entry.actor.BodyPal = 1 // Mirror Image also works on dyed bodies.
	tint := actorStateTint(entry.actor)
	if tint != (color.RGBA{155, 155, 255, 255}) {
		t.Fatalf("mirror tint=%v", tint)
	}
	billboard := &spriteBillboard{image: render.NewImage(10, 20)}
	projection := newSceneProjectionForTarget(800, 600, 10, 20, 0)
	start := time.Unix(100, 0)
	for tick := 0; tick <= 20; tick++ {
		m.drawActorBlurs(render.NewFrame(800, 600), projection, entry, billboard, 1, 1, 1, tint, start.Add(time.Duration(tick)*db.EffectFrameDuration))
	}
	if len(m.actorBlurs[100]) != 2 {
		t.Fatal("Mirror Image must emit every twenty ticks")
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{ActorID: 2000000, StatusID: db.StatusNjBunsinjyutsu})
	entry.actor = ctx.World.Player
	m.drawActorBlurs(render.NewFrame(800, 600), projection, entry, billboard, 1, 1, 1, tint, start.Add(44*db.EffectFrameDuration))
	if len(m.actorBlurs) != 0 {
		t.Fatal("Mirror Image survived status removal")
	}
}

func TestNinjaWindAndPetalsAreFrameRateIndependent(t *testing.T) {
	ctx := archerEffectContext()
	m := &WorldMode{}
	start := time.Unix(100, 0)
	m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: 540, HitCount: 3, SourceID: 2000000, TargetID: 300}, start)
	e := m.worldEffects[0]
	var stepped, jumped ninjaWind
	for tick := 0; tick < 16; tick++ {
		stepped.advance(ctx, e, start.Add(time.Duration(tick)*db.EffectFrameDuration))
	}
	jumped.advance(ctx, e, start.Add(15*db.EffectFrameDuration))
	if !reflect.DeepEqual(stepped, jumped) {
		t.Fatal("wind path depends on frame rate")
	}
	if stepped.heads[0].samples[0].position.z == stepped.heads[1].samples[0].position.z {
		t.Fatal("alternating wind blades must curve on different sides")
	}
	delete(ctx.World.Actors, 300)
	stepped.advance(ctx, e, start.Add(100*db.EffectFrameDuration))
	for _, h := range stepped.heads {
		for _, s := range h.samples {
			if s.alpha != 0 {
				t.Fatal("wind did not fade after losing its target")
			}
		}
	}
	ctx = archerEffectContext()
	m = &WorldMode{}
	m.addWorldEffectAt(ctx, effectKouenka, 300, start)
	e = m.worldEffects[0]
	var a, b ninjaPetals
	for tick := 0; tick < 30; tick++ {
		a.advance(ctx, e, start.Add(time.Duration(tick)*db.EffectFrameDuration))
	}
	b.advance(ctx, e, start.Add(29*db.EffectFrameDuration))
	if !reflect.DeepEqual(a, b) {
		t.Fatal("petals depend on frame rate")
	}
	if a.particles[0].position.y <= e.actorOrigin.y || a.particles[19].age != 11 {
		t.Fatal("petals must rise and appear one per tick")
	}
}

func TestNinjaNativeAssetsAndRendering(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	ctx.World.Player.Job, ctx.Session.Selected.Job = db.JobNinja, db.JobNinja
	ctx.World.Player.Weapon, ctx.Session.Selected.Weapon = 0, 0
	for _, sex := range []uint8{0, 1} {
		view, status := loadPlayerHumanoidSpriteView(ctx.Resources, ctx.Session.Selected, sex, false)
		if view == nil {
			t.Fatalf("missing Ninja sprite: %s", status)
		}
		for direction := 0; direction < 8; direction++ {
			_, action, ok := resolveSpriteAction(view.body.act, spriteActionPCSkill, direction)
			if !ok || len(action.Animations) < 6 {
				t.Fatalf("Ninja sex %d direction %d lacks native skill poses", sex, direction)
			}
		}
	}
	m := &WorldMode{}
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 10, 20, 0)
	for _, id := range []int{613, 614, 615, 616, 618, 619, 620, 621, 622, 624, 630, 631, 632, 633, 634, 635, 636} {
		m.worldEffects = nil
		if !m.addWorldEffectBetweenAt(ctx, id, 300, 2000000, start) {
			t.Fatalf("effect %d missing", id)
		}
		e := m.worldEffects[0]
		e.hitCount = 3
		spec, _ := worldEffectSpecForID(id)
		for _, c := range spec.components {
			if c.spriteFile != "" && m.effectSpriteView(ctx.Resources, c.spriteFile) == nil {
				t.Fatalf("missing SPR %s", c.spriteFile)
			}
			if c.textureFile != "" && m.effectFileTexture(ctx.Resources, c.textureFile) == nil {
				t.Fatalf("missing texture %s", c.textureFile)
			}
		}
		if id == 618 || id == 621 || id == 622 || id == 624 || id == 631 || id == 634 {
			frame := render.NewFrame(800, 600)
			for _, c := range spec.components {
				m.drawFuncEffect(frame, ctx, projection, e, c, 0, 20.5, 20.5, 0, 0, start.Add(10*db.EffectFrameDuration))
			}
			v := reflect.ValueOf(frame).Elem()
			if v.FieldByName("worldCommands").Len()+v.FieldByName("worldBillboards").Len() == 0 {
				t.Fatalf("effect %d produced no draw commands", id)
			}
		}
	}
}
