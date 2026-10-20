package game

import (
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

func TestHunterTrapPlacementAndTrigger(t *testing.T) {
	for _, tc := range []struct {
		skill, unit uint16
		effect      int
	}{
		{db.SkillHTLandmine, 147, effectFirePillarBomb},
		{db.SkillHTAnklesnare, 145, 0}, {db.SkillHTShockwave, 148, 0},
		{db.SkillHTSandman, 149, effectSandman}, {db.SkillHTFlasher, 150, effectFlasher},
		{db.SkillHTFreezingtrap, 151, effectFreezingTrap}, {db.SkillHTBlastmine, 143, effectBlastMineBomb},
		{db.SkillHTClaymoretrap, 152, effectClaymore}, {db.SkillHTTalkiebox, 153, 0},
	} {
		t.Run(db.SkillResourceName[tc.skill], func(t *testing.T) {
			ctx := archerEffectContext()
			m := &WorldMode{}
			m.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SourceID: 2000000, SkillID: tc.skill, X: 15, Y: 20})
			if len(m.worldEffects) != 0 || m.actorAnims[2000000].actionFamily != spriteActionPickup {
				t.Fatal("placing a trap must play pickup without detonating it")
			}
			m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: tc.unit, X: 15, Y: 20, Visible: true})
			if tc.unit == 148 && (len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectShockwave || !m.worldEffects[0].persistent) {
				t.Fatal("Shock Wave must loop on the trap")
			}
			before := len(m.worldEffects)
			m.applyActorActionNotify(ctx, network.ActorActionNotify{SourceID: 500, TargetID: 300, SkillID: tc.skill, Damage: 100, HitCount: 3, Action: 5})
			hits := 0
			for _, e := range m.worldEffects[before:] {
				if e.effectID != effectHit1 || e.actorID != 300 {
					t.Fatal("victim damage must only create the small impact, not the trap explosion")
				}
				hits++
			}
			if (tc.unit == 147 || tc.unit == 151 || tc.unit == 143 || tc.unit == 152) && hits != 3 {
				t.Fatal("missing trap damage impacts")
			}
			look := uint32(db.SkillUnitUsedTraps)
			if tc.unit == 147 {
				look = 141
			}
			if !m.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 500, Value: look}) {
				t.Fatal("unhandled trap look")
			}
			m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 500})
			if tc.effect == 0 {
				if len(m.worldEffects) != 0 {
					t.Fatal("expired trap retained its aura")
				}
				return
			}
			if len(m.worldEffects) != hits+1 {
				t.Fatalf("got %d explosions", len(m.worldEffects)-hits)
			}
			e := m.worldEffects[hits]
			if e.effectID != tc.effect || e.actorID != 0 || e.persistent || e.x != 15 || e.y != 20 {
				t.Fatalf("incorrect trap burst: %+v", e)
			}
			if tc.unit == 143 {
				found := false
				for _, sound := range m.scheduledSounds {
					if sound.paths[0] == "effect\\hunter_blastmine.wav" {
						found = true
						if !sound.at.Equal(e.starts.Add(20 * db.EffectFrameDuration)) {
							t.Fatal("Blast Mine sound must wait twenty ticks")
						}
					}
				}
				if !found {
					t.Fatal("missing Blast Mine sound")
				}
			}
		})
	}
}

func TestHunterTrapUtilityTargetsTrapAndSurvivesRemoval(t *testing.T) {
	for _, skill := range []uint16{db.SkillHTRemovetrap, db.SkillHTSpringtrap} {
		ctx := archerEffectContext()
		ctx.World.Player.Job = db.JobHunter
		ctx.Session.Selected.Job = db.JobHunter
		ctx.Session.Selected.Option = db.EffectStateFalcon
		m := &WorldMode{}
		m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: 149, X: 15, Y: 25, Visible: true})
		n := network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 500, SkillID: skill}
		m.applySkillNoDamageNotify(ctx, n)
		if len(m.worldEffects) != 0 {
			t.Fatal("failed trap skill emitted effect")
		}
		n.Result = 1
		m.applySkillNoDamageNotify(ctx, n)
		m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 500})
		if len(m.worldEffects) != 1 || m.worldEffects[0].actorID != 0 || m.worldEffects[0].x != 15 || m.worldEffects[0].y != 25 {
			t.Fatal("effect must stay at removed trap")
		}
		if skill == db.SkillHTSpringtrap && (m.falcons[100] == nil || !m.falcons[100].attacking) {
			t.Fatal("Spring Trap did not send the falcon")
		}
	}
}

func TestHunterBlitzBeatFirstHitOnly(t *testing.T) {
	ctx := archerEffectContext()
	m := &WorldMode{}
	a := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillHTBlitzbeat, Damage: 500, HitCount: 5, SourceSpeed: 500}
	m.applyActorActionNotify(ctx, a)
	if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectBlitzBeat || m.worldEffects[0].actorID != 300 {
		t.Fatal("Blitz Beat needs one victim effect for the whole attack")
	}
	e := m.worldEffects[0]
	for i, delay := range []time.Duration{5 * db.EffectFrameDuration, 30 * db.EffectFrameDuration} {
		found := false
		for _, sound := range m.scheduledSounds {
			if sound.at.Equal(e.starts.Add(delay)) && sound.paths[0] == db.EffectSpecs[effectBlitzBeat].SFX[i] {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing Blitz Beat sound at %s", delay)
		}
	}
	a.Action = 5
	m.applyActorActionNotify(ctx, a)
	if len(m.worldEffects) != 1 {
		t.Fatal("splash victim replayed Blitz Beat")
	}
}

func TestHunterPhantasmicArrowRelease(t *testing.T) {
	ctx := archerEffectContext()
	family := attackActionFamilyForActorWithResources(nil, ctx.World.Player)
	m := &WorldMode{playerView: humanoidSoundView(family, 4, 2, []string{"atk"})}
	m.applyActorActionNotify(ctx, network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillHTPhantasmic, Damage: 100, HitCount: 1, SourceSpeed: 500})
	if m.actorAnims[2000000].actionFamily != family || len(m.worldEffects) != 4 {
		t.Fatal("Phantasmic Arrow must use the normal bow pose, flash, arrow, and two impacts")
	}
	arrow := m.worldEffects[1]
	if arrow.effectID != effectArrowShot || arrow.starts.Sub(m.actorAnims[2000000].started) != 250*time.Millisecond || arrow.expires.Sub(arrow.starts) != referenceBowArrowFlightDuration {
		t.Fatalf("wrong arrow timing: %+v", arrow)
	}
	for _, e := range m.worldEffects[2:] {
		if !e.starts.Equal(arrow.expires) {
			t.Fatal("impact precedes arrow arrival")
		}
	}
}

func TestHunterAnkleSnareClosesOnce(t *testing.T) {
	ctx := archerEffectContext()
	m := &WorldMode{}
	m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: 145, X: 15, Y: 20, Visible: false})
	m.applySkillUnitUpdate(network.SkillUnitUpdate{ID: 500})
	if !m.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 500, Value: 145}) {
		t.Fatal("Ankle Snare close was ignored")
	}
	u := m.skillUnitModels[500]
	rsm := &res.RSM{AnimLength: 1000}
	for _, tc := range []struct {
		age    time.Duration
		frame  int
		moving bool
	}{{0, 3, true}, {db.EffectFrameDuration, 103, true}, {time.Second, 903, false}} {
		frame, moving := skillUnitRSMFrame(rsm, res.RSWModel{}, u, u.animationStarted.Add(tc.age))
		if frame != tc.frame || moving != tc.moving {
			t.Fatalf("age %s: frame=%d moving=%t", tc.age, frame, moving)
		}
	}
}

func TestHunterTrapTargetAndTalkieBox(t *testing.T) {
	ctx := archerEffectContext()
	m := &WorldMode{}
	m.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 500, UnitID: 153, X: 15, Y: 20, Visible: true})
	projection := newSceneProjectionForTarget(800, 600, 15, 20, 0)
	p := projection.Project(15.5, 20.5, 0)
	skill := session.Skill{ID: db.SkillHTRemovetrap, Type: skillTargetTrap, Level: 1, Range: 2}
	actor, ok := m.clickedSkillTarget(ctx, projection, skill, int(p.x), int(p.y), time.Now())
	if !ok || actor.ID != 500 {
		t.Fatal("trap could not be selected")
	}
	c := skillController{mode: m}
	pending := pendingSkillTarget{skill: skill, targetID: 500}
	if x, y, ok := c.targetCell(ctx, pending, time.Now()); !ok || x != 15 || y != 20 {
		t.Fatal("trap chase lost target")
	}
	m.applySpeechBubble(ctx, network.ChatMessage{GID: 500, Text: "Beware : trap!"}, time.Now())
	m.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 500, Value: 140})
	b := m.speechBubbles[500]
	if b.ground == nil || *b.ground != [2]int{15, 20} || b.text != "Beware : trap!" {
		t.Fatalf("wrong Talkie Box bubble: %+v", b)
	}
	if _, _, ok := c.targetCell(ctx, pending, time.Now()); ok {
		t.Fatal("chase retained removed trap")
	}
	if _, ok := m.clickedSkillTarget(ctx, projection, skill, int(p.x), int(p.y), time.Now()); ok {
		t.Fatal("used trap still selectable")
	}
	m.addWorldEffectAt(ctx, effectBlitzBeat, 300, time.Now())
	m.applySpeechBubble(ctx, network.ChatMessage{GID: 300, Text: "Actor : hello"}, time.Now())
	if b := m.speechBubbles[300]; b.ground != nil || b.text != "hello" {
		t.Fatal("an actor's visual effects must not turn its speech into a ground bubble")
	}
}

func TestHunterEffectsWithClientAssets(t *testing.T) {
	ctx := archerEffectContext()
	ctx.Resources = realDataManager(t)
	projection := newSceneProjectionForTarget(800, 600, 20, 20, 0)
	start := time.Unix(100, 0)
	for _, tc := range []struct{ id, frame int }{{effectBlitzBeat, 10}, {effectBlitzBeat, 40}, {effectFlasher, 30}, {effectDetecting, 20}, {effectRemoveTrap, 20}, {effectShockwave, 20}} {
		m := &WorldMode{whitePixel: render.NewImage(1, 1)}
		m.addWorldEffectAt(ctx, tc.id, 300, start)
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(time.Duration(tc.frame)*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len()+reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() == 0 {
			t.Fatalf("effect %d frame %d drew nothing", tc.id, tc.frame)
		}
		for _, sound := range m.scheduledSounds {
			for _, path := range sound.paths {
				if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
					t.Error(err)
				}
			}
		}
	}
}
