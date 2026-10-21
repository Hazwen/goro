package game

import (
	"image/color"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestGypsySkillAnimationsWithClientAssets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	ctx.World.Player.Job, ctx.World.Player.Weapon, ctx.World.Player.Sex = db.JobDancerH, 1950, 0 // Rope
	ctx.Session.Selected.Job, ctx.Session.Selected.Weapon = db.JobDancerH, 1950
	view, status := loadPlayerHumanoidSpriteView(ctx.Resources, ctx.Session.Selected, 0, false)
	if view == nil || !strings.Contains(view.body.actSource, res.PlayerJobToken(db.JobDancerH)) {
		t.Fatalf("must load the Gypsy body, without a Novice fallback: %s", status)
	}
	m := &WorldMode{playerView: view}
	for direction := 0; direction < 8; direction++ {
		ctx.World.Player.Dir = direction
		a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillCGArrowvulcan, SourceSpeed: 500, HitCount: 9, Damage: 100}
		m.applyActorActionNotify(ctx, a)
		anim := m.actorAnims[a.SourceID]
		if anim.actionFamily != spriteActionPCAttack3 || anim.next == nil || anim.next.actionFamily != spriteActionPCAttack2 {
			t.Fatal("Gypsy must shoot with ACT 88, then use her whip attack for additional hits")
		}
		for _, step := range []actorAnimation{anim, *anim.next, *anim.next.next} {
			index, action, ok := resolveSpriteAction(view.body.act, step.actionFamily, direction)
			if !ok || index/8 != step.actionFamily || step.hasFixedMotion && step.fixedMotion >= len(action.Animations) {
				t.Fatal("Gypsy attack must use its requested action and frame without fallback")
			}
			state := spriteState{direction: direction}
			applyActorAnimationToSpriteState(&state, step, true)
			billboard, ok := humanoidBillboardForState(view, state, step.started)
			if !ok || billboard == nil || billboard.image == nil {
				t.Fatalf("Gypsy attack failed to render for direction %d", direction)
			}
		}
		m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillCGHermode, SourceID: a.SourceID})
		dance := m.actorAnims[a.SourceID]
		_, action, ok := resolveSpriteAction(view.body.act, spriteActionPCSkill, direction)
		if !ok || len(action.Animations) < 4 || !dance.pingPong {
			t.Fatal("Gypsy must have the complete three-frame dance, without falling back to another action")
		}
		state := spriteState{direction: direction}
		applyActorAnimationToSpriteState(&state, dance, true)
		for i, want := range []int{1, 2, 3, 2, 1} {
			at := dance.started.Add(time.Duration(i) * dance.speed)
			if got := bodyMotionForState(action, state, dance.started, at); got != want {
				t.Fatalf("Gypsy dance frame %d: got %d, want %d", i, got, want)
			}
		}
	}
}

func TestClownMoonlitStatusOwnership(t *testing.T) {
	ctx := archerEffectContext()
	m := &WorldMode{}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusMoon, ActorID: 100, Active: true})
	if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectMoonlit || !m.worldEffects[0].expires.IsZero() {
		t.Fatal("Moonlit status must create a persistent sphere, including on entry into view")
	}
	start := m.worldEffects[0].starts
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillCGMoonlit, SourceID: 2000000, TargetID: 2000000, Result: 1})
	m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: 181, X: 10, Y: 20, Visible: true})
	if len(m.worldEffects) != 1 || !m.worldEffects[0].starts.Equal(start) || len(m.scheduledSounds) != 1 {
		t.Fatal("activation and ground units must not duplicate or restart the sphere")
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusMoon, ActorID: 2000000})
	if len(m.worldEffects) != 0 || ctx.World.Player.Opt3State&db.Opt3Moonlit != 0 {
		t.Fatal("Moonlit sphere survived its status ending")
	}
}

func TestClownMarionetteLinkLifetime(t *testing.T) {
	for _, end := range []uint16{db.StatusMarionetteMaster, db.StatusMarionette} {
		ctx := archerEffectContext()
		m := &WorldMode{}
		m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusMarionetteMaster, ActorID: 2000000, Active: true})
		m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusMarionette, ActorID: 300, Active: true})
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillCGMarionette, SourceID: 100, TargetID: 300, Result: 1})
		if len(m.worldEffects) != 3 || len(m.scheduledSounds) != 1 {
			t.Fatal("Marionette needs one link, two body glows, and one sound")
		}
		link := m.worldEffects[2]
		if link.effectID != effectMarionetteLink || link.actorID != 2000000 || link.targetID != 300 || !link.expires.IsZero() {
			t.Fatal("link must connect the caster's canonical ID to the recipient")
		}
		id := uint32(100)
		if end == db.StatusMarionette {
			id = 300
		}
		m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: end, ActorID: id})
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectPinkBody {
			t.Fatal("ending either endpoint must remove the link and its own body glow")
		}
	}
	ctx := archerEffectContext()
	m := &WorldMode{}
	n := network.SkillNoDamageNotify{SkillID: db.SkillCGMarionette, SourceID: 2000000, TargetID: 300, Result: 1}
	m.applySkillNoDamageNotify(ctx, n)
	m.removeWorldEffectsForActor(300)
	if len(m.worldEffects) != 1 || m.worldEffects[0].actorID != n.SourceID {
		t.Fatal("recipient leaving view must remove the link as well as its body effect")
	}
	m = &WorldMode{}
	n.TargetID = 999
	m.applySkillNoDamageNotify(ctx, n)
	if len(m.worldEffects) != 0 {
		t.Fatal("missing recipient must not leave an immortal link")
	}
}

func TestClownArrowVulcanTiming(t *testing.T) {
	ctx := archerEffectContext()
	m := &WorldMode{playerView: humanoidSoundView(spriteActionPCAttack3, 6, 3, []string{"atk"})}
	a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillCGArrowvulcan, SourceSpeed: 500, HitCount: 9, Damage: 100}
	m.applyActorActionNotify(ctx, a)
	start := m.actorAnims[a.SourceID].started
	if anim := m.actorAnims[a.SourceID]; anim.actionFamily != spriteActionPCAttack3 || anim.speed != 500*time.Millisecond/6 {
		t.Fatal("Arrow Vulcan must release with ACT 88 at the server's attack speed")
	}
	counts := map[int]int{}
	var hitAt time.Time
	for _, e := range m.worldEffects {
		counts[e.effectID]++
		switch e.effectID {
		case effectTripleAttack3:
			if e.actorID != a.SourceID || e.targetID != a.TargetID || !e.starts.Equal(start) || e.projectileDirection.x != 10 {
				t.Fatal("rays must aim from the caster through the victim at action start")
			}
		case effectArrowShot:
			if e.starts.Sub(start) != 250*time.Millisecond || e.expires.Sub(e.starts) != referenceBowArrowFlightDuration {
				t.Fatal("arrow must leave at the ACT marker and fly for eight native ticks")
			}
			hitAt = e.expires
		}
	}
	if !reflect.DeepEqual(counts, map[int]int{effectTripleAttack3: 1, effectArrowShot: 1, effectHit1: 9, effectBashHit: 1}) {
		t.Fatalf("unexpected Arrow Vulcan effects: %v", counts)
	}
	for i := 1; i < 9; i++ {
		at := hitAt.Add(time.Duration(i) * multiHitDelay)
		anim, _ := m.actorAnimation(a.SourceID, at)
		if anim.actionFamily != attackActionFamilyForActorWithResources(nil, ctx.World.Player) || !anim.hasFixedMotion || anim.fixedMotion != 4 {
			t.Fatalf("additional hit %d must restart the native attack", i)
		}
	}
	before := len(m.worldEffects)
	a.Action = 5
	m.applyActorActionNotify(ctx, a)
	if len(m.worldEffects) != before {
		t.Fatal("secondary splash packet replayed Arrow Vulcan")
	}
	for _, skill := range []uint16{db.SkillCGArrowvulcan, db.SkillCGTarotcard} {
		m = &WorldMode{}
		m.startSkillSourceCastAnimation(ctx, a.SourceID, skill, time.Second, start)
		m.addSkillCastEffects(ctx, skill, 5, a.SourceID, a.TargetID, 0, 0, time.Second, start, "test")
		if m.worldEffects[2].effectID != effectBeginSpell6 {
			t.Fatal("Clown casts must use the white aura")
		}
		if skill == db.SkillCGArrowvulcan && (!m.actorAnims[a.SourceID].hasFixedMotion || m.actorAnims[a.SourceID].actionFamily != spriteActionPCAttack3) {
			t.Fatal("Vulcan cast must hold the ready-shot pose")
		}
	}
}

func TestClownSuccessPacketsAndTarotSelection(t *testing.T) {
	ctx := archerEffectContext()
	for _, skill := range []uint16{db.SkillCGLongingfreedom, db.SkillCGTarotcard, db.SkillCGHermode} {
		m := &WorldMode{}
		n := network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatal("failed skill must not play effects or sounds")
		}
		n.Result = 1
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.scheduledSounds) != 1 {
			t.Fatal("successful skill must play exactly one sound")
		}
		if skill == db.SkillCGHermode {
			if len(m.worldEffects) != 0 {
				t.Fatal("Hermode activation must not create an extra ground cube on the caster")
			}
			continue
		}
		id := n.TargetID
		if skill == db.SkillCGLongingfreedom {
			id = n.SourceID
		}
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectChemicalBody || m.worldEffects[0].actorID != id {
			t.Fatal("success packet must flash the correct body without inventing a card")
		}
		if skill == db.SkillCGTarotcard {
			m.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 2000000, EffectID: effectTarotCard6})
			if len(m.scheduledSounds) != 1 || len(m.worldEffects) != 2 || m.worldEffects[1].effectID != effectTarotCard6 || !isLocalActor(ctx, m.worldEffects[1].actorID) {
				t.Fatal("Lovers card must use the server's card and actor, without repeating the sound")
			}
		}
	}
	for _, tc := range []struct{ frame, alpha int }{{0, 10}, {21, 220}, {198, 220}, {199, 215}, {242, 0}} {
		if alpha := tarotCardAlpha(tc.frame); alpha != tc.alpha {
			t.Fatalf("Tarot frame %d: alpha %d, want %d", tc.frame, alpha, tc.alpha)
		}
	}
}

func TestClownHermodeGroundUnitOwnership(t *testing.T) {
	ctx := archerEffectContext()
	m := &WorldMode{}
	m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillCGHermode, SourceID: 2000000})
	if len(m.worldEffects) != 0 || len(m.scheduledSounds) != 1 || !m.actorAnims[2000000].loop {
		t.Fatal("Hermode ground activation must only begin the dance and sound")
	}
	unit := network.SkillUnitEntry{ID: 500, UnitID: 185, X: 15, Y: 20, Visible: true}
	m.applySkillUnitEntry(ctx, unit)
	if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectBottomHermode || len(m.scheduledSounds) != 1 {
		t.Fatal("Hermode's unit must own one silent cube")
	}
	start := m.worldEffects[0].starts
	m.applySkillUnitEntry(ctx, unit)
	if !m.worldEffects[0].starts.Equal(start) {
		t.Fatal("refresh must not restart the cube")
	}
	m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: unit.ID})
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusBdplaying, ActorID: 100})
	if len(m.worldEffects)+len(m.actorAnims) != 0 {
		t.Fatal("Hermode ground removal and status ending must stop the cube and dance")
	}
}

func TestClownEffectsWithClientAssets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 15, 20, 0)
	ids := []int{effectTripleAttack3, effectMoonlit, effectMarionetteLink, effectBottomHermode}
	for id := effectTarotCard1; id <= effectTarotCard14; id++ {
		ids = append(ids, id)
	}
	for _, id := range ids {
		m := &WorldMode{}
		m.addWorldEffectBetweenAt(ctx, id, 300, 2000000, start)
		for _, sound := range m.scheduledSounds {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound.paths[0]); err != nil {
				t.Fatal(err)
			}
			if id == effectTripleAttack3 && sound.at.Sub(start) != 30*db.EffectFrameDuration {
				t.Fatal("Arrow Vulcan sound must wait 30 ticks")
			}
		}
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(65*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		billboards := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if commands.Len()+billboards.Len() == 0 {
			t.Fatalf("Clown effect %d drew nothing with actual client assets", id)
		}
		if id >= effectTarotCard1 && id <= effectTarotCard14 {
			cmd := billboards.Index(0)
			if math.Abs(cmd.FieldByName("Center").Index(1).Float()-6.07) > .001 || len(m.scheduledSounds) != 0 {
				t.Fatal("Tarot card must hover at native height and remain silent")
			}
		}
	}
	for _, skill := range []uint16{db.SkillCGMoonlit, db.SkillCGHermode, db.SkillCGLongingfreedom, db.SkillCGTarotcard} {
		for _, sound := range db.SkillEffects[skill].SFX {
			if _, err := ctx.Resources.ReadFile("data/wav/" + sound); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestClownBodyEffectRendering(t *testing.T) {
	ctx := archerEffectContext()
	start := time.Unix(100, 0)
	for _, tc := range []struct{ id, frame, copies int }{{effectPinkBody, 1000, 2}, {effectChemicalBody, 30, 2}, {effectChemicalBody, 31, 1}} {
		m := &WorldMode{}
		m.addWorldEffectAt(ctx, tc.id, 300, start)
		frame := render.NewFrame(800, 600)
		billboard := &spriteBillboard{image: render.NewImage(10, 20), anchorX: 5, anchorY: 20}
		entry := sceneActorDrawEntry{actor: ctx.World.Actors[300], worldX: 20.5, worldY: 20.5}
		m.drawActorSpriteWithEffects3D(ctx, frame, newSceneProjectionForTarget(800, 600, 15, 20, 0), entry, billboard, 1, 1, 1,
			color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Duration(tc.frame)*db.EffectFrameDuration))
		if count := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); count != tc.copies {
			t.Fatalf("body effect %d frame %d: %d layers, want %d", tc.id, tc.frame, count, tc.copies)
		}
	}
}
