package game

import (
	"math"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/world"
)

func knightEffectContext() client.Context {
	w := world.New()
	w.Player = world.Actor{ID: 100, X: 10, Y: 20, Job: db.JobKnight}
	w.Actors[300] = world.Actor{ID: 300, X: 30, Y: 20, Job: 1002}
	return client.Context{World: w, Resources: &res.Manager{}, Session: &session.Session{AccountID: 2000000, CharID: 100}}
}

func TestKnightAttackEffects(t *testing.T) {
	starts := time.Unix(100, 0)
	hitAt := starts.Add(300 * time.Millisecond)
	for _, tc := range []struct {
		skill uint16
		begin int
		hits  []int
	}{
		{db.SkillKNPierce, effectPierceSelf, []int{effectPierce}},
		{db.SkillKNSpearstab, effectSpearStabSelf, []int{effectPierce}},
		{db.SkillKNBowlingbash, effectBowlingSelf, []int{effectHit1, effectBowlingBash, effectHit1, effectHit1}},
		{db.SkillKNSpearboomerang, effectSpearBmrSelf, []int{effectHit4}},
		{db.SkillKNChargeatk, -1, []int{effectBashHit}},
	} {
		ctx := knightEffectContext()
		m := &WorldMode{}
		action := network.ActorActionNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300, Damage: 90, HitCount: 3, Action: 8}
		m.addSkillBeginEffect(ctx, action, starts)
		m.addSkillEffect(ctx, action, starts, hitAt)
		if tc.begin >= 0 {
			if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.begin || m.worldEffects[0].actorID != 2000000 || !m.worldEffects[0].starts.Equal(starts) {
				t.Fatalf("skill %d must start its caster STR before impact: %+v", tc.skill, m.worldEffects)
			}
		} else if len(m.worldEffects) != 0 {
			t.Fatalf("skill %d invented a startup flash", tc.skill)
		}
		before := len(m.worldEffects)
		m.addSkillHitEffect(ctx, action, hitAt)
		if len(m.worldEffects) != before+len(tc.hits) {
			t.Fatalf("skill %d replayed its impact for each damage number: %+v", tc.skill, m.worldEffects)
		}
		for i, id := range tc.hits {
			effect := m.worldEffects[before+i]
			wantAt := hitAt
			if tc.skill == db.SkillKNBowlingbash && i > 1 {
				wantAt = wantAt.Add(time.Duration(i-1) * multiHitDelay)
			}
			if effect.effectID != id || effect.actorID != 300 || !effect.starts.Equal(wantAt) {
				t.Fatalf("skill %d impact has wrong owner, visual or timing: %+v", tc.skill, effect)
			}
		}
		action.Damage = 0
		m = &WorldMode{}
		m.addSkillHitEffect(ctx, action, hitAt)
		if len(m.worldEffects) != 0 {
			t.Fatalf("skill %d played a hit on a miss", tc.skill)
		}
	}
}

func TestKnightSplashDoesNotReplayCasterOrEffects(t *testing.T) {
	for _, skill := range []uint16{db.SkillKNBrandishspear, db.SkillKNSpearstab, db.SkillKNBowlingbash} {
		ctx := knightEffectContext()
		ctx.World.Player.Dir = 2
		m := &WorldMode{}
		starts := time.Now()
		m.setCombatActorAction(ctx, 2000000, actorAnimation{actionFamily: spriteActionPCAttack1, started: starts, duration: time.Second})
		m.applyActorActionNotify(ctx, network.ActorActionNotify{
			SkillID: skill, SourceID: 2000000, TargetID: 300, Damage: 50, HitCount: 1, Action: 5,
		})
		if len(m.worldEffects) != 0 {
			t.Fatalf("splash skill %d replayed effects: %+v", skill, m.worldEffects)
		}
		if !m.actorAnims[2000000].started.Equal(starts) || ctx.World.Player.Dir != 2 {
			t.Fatalf("splash skill %d restarted or turned the caster", skill)
		}
	}
}

func TestKnightBrandishStartsBothSTRsImmediately(t *testing.T) {
	ctx := knightEffectContext()
	m := &WorldMode{}
	now := time.Now()
	action := network.ActorActionNotify{SkillID: db.SkillKNBrandishspear, SourceID: 2000000, TargetID: 300, Damage: 50}
	m.addSkillBeginEffect(ctx, action, now)
	m.addSkillEffect(ctx, action, now, now.Add(time.Second))
	if len(m.worldEffects) != 2 || len(m.scheduledSounds) != 0 {
		t.Fatalf("Brandish effects or extra audio: %+v, %+v", m.worldEffects, m.scheduledSounds)
	}
	for i, want := range []struct {
		id    int
		actor uint32
	}{{effectBrandishSpear2, 2000000}, {effectBrandishSpear, 300}} {
		effect := m.worldEffects[i]
		if effect.effectID != want.id || effect.actorID != want.actor || !effect.starts.Equal(now) {
			t.Fatalf("Brandish phase %d is delayed or misplaced: %+v", i, effect)
		}
	}
}

func TestKnightQuickenAndAutoCounter(t *testing.T) {
	ctx := knightEffectContext()
	for _, skill := range []uint16{db.SkillKNTwohandquicken, db.SkillKNOnehand} {
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects) != 0 {
			t.Fatal("failed Quicken played an effect")
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects) != 1 || m.worldEffects[0].actorID != 2000000 || m.worldEffects[0].effectID != effectTwoHandQuicken || len(m.scheduledSounds) != 1 {
			t.Fatal("Quicken must play once on the caster")
		}
	}
	ctx.Session.Selected = session.Character{ID: 100, Job: db.JobKnight, Weapon: db.WeaponSword}
	for _, state := range []string{"ready", "expired", "cancelled"} {
		m := &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillKNAutocounter, SourceID: 2000000, TargetID: 2000000, Result: 1})
		anim := m.actorAnims[2000000]
		if !anim.autoCounter || anim.actionFamily != spriteActionPCAttack2 || !anim.hasFixedMotion || anim.fixedMotion != 0 || anim.duration != 1500*time.Millisecond || len(m.worldEffects) != 0 || len(m.scheduledSounds) != 0 {
			t.Fatalf("Auto Counter must silently hold the first attack frame: %+v", anim)
		}
		if state == "expired" {
			anim.started = time.Now().Add(-2 * time.Second)
			m.setCombatActorAction(ctx, 2000000, anim)
		} else if state == "cancelled" {
			m.applyActorActionNotify(ctx, network.ActorActionNotify{SourceID: 2000000, Action: network.ActionSitDown})
		}
		action := network.ActorActionNotify{SourceID: 2000000, TargetID: 300, Action: 10, Damage: 50, HitCount: 1}
		m.applyActorActionNotify(ctx, action)
		count := 0
		for _, effect := range m.worldEffects {
			if effect.effectID == effectAutoCounter {
				count++
			}
		}
		want := 0
		if state == "ready" {
			want = 1
		}
		if count != want {
			t.Fatalf("state=%s counter effects=%d, want %d", state, count, want)
		}
		before := len(m.worldEffects)
		m.applyActorActionNotify(ctx, action)
		for _, effect := range m.worldEffects[before:] {
			if effect.effectID == effectAutoCounter {
				t.Fatal("a normal attack replayed an already consumed counter effect")
			}
		}
	}
}

func TestKnightChargeAttackCast(t *testing.T) {
	ctx := knightEffectContext()
	m := &WorldMode{}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: db.SkillKNChargeatk, SourceID: 2000000, TargetID: 300, DelayTime: 1000})
	if m.actorAnims[2000000].actionFamily != spriteActionIdle {
		t.Fatal("Charge Attack must charge while standing")
	}
	found := false
	for _, effect := range m.worldEffects {
		if effect.effectID == effectBeginSpell6 {
			found = true
		}
	}
	if !found {
		t.Fatal("Charge Attack is missing its native casting aura")
	}
}

func TestKnightSpearFlightAndSounds(t *testing.T) {
	ctx := knightEffectContext()
	m := &WorldMode{}
	starts := time.Unix(100, 0)
	m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: db.SkillKNSpearboomerang, SourceID: 2000000, TargetID: 300, HitCount: 3}, starts)
	if len(m.worldEffects) != 1 || len(m.scheduledSounds) != 4 {
		t.Fatalf("Spear Boomerang must launch one four-image trail: %+v", m.worldEffects)
	}
	effect := m.worldEffects[0]
	if effect.actorID != 2000000 || effect.targetID != 300 {
		t.Fatal("spear travels in the wrong direction")
	}
	for i, sound := range m.scheduledSounds {
		if !sound.at.Equal(starts.Add(time.Duration(i*4) * db.EffectFrameDuration)) {
			t.Fatal("spear sounds are not spaced four native ticks apart")
		}
	}
	for _, age := range []int{0, 6, 13} {
		point, ok := spearBoomerangPosition(effect, age)
		if !ok || math.Abs(point.x-(10.5+20*float64(age+1)/15)) > .001 || math.Abs(point.y-3.07) > .001 {
			t.Fatalf("unexpected spear position at tick %d: %+v", age, point)
		}
	}
	for _, age := range []int{-1, 14, 15, 16} {
		if _, ok := spearBoomerangPosition(effect, age); ok {
			t.Fatalf("spear visible before launch or after reaching target: %d", age)
		}
	}
}

func TestKnightBowlingRings(t *testing.T) {
	ctx := knightEffectContext()
	starts := time.Unix(100, 0)
	spec, _ := worldEffectSpecForID(effectBowlingBash)
	component := spec.components[0]
	m := &WorldMode{textures: map[string]*render.Image{}}
	for _, path := range []string{component.textureFile, component.textureFiles[0]} {
		m.textures["__effectfile_"+path+"|blackkey=true"] = render.NewImage(8, 8)
	}
	m.addWorldEffectAt(ctx, effectBowlingBash, 300, starts)
	for _, tc := range []struct{ tick, draws int }{{0, 2}, {5, 3}, {20, 1}, {50, 0}} {
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, sceneProjection{}, starts.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if commands.Len() != tc.draws {
			t.Fatalf("tick %d: %d draws, want %d", tc.tick, commands.Len(), tc.draws)
		}
		for i := 0; i < commands.Len(); i++ {
			command := commands.Index(i)
			options := command.FieldByName("Options")
			if options.FieldByName("DepthWrite").Bool() || options.FieldByName("Blend").Int() != int64(render.BlendSourceOver) {
				t.Fatal("Bowling Bash must alpha blend without writing depth")
			}
			if options.FieldByName("DepthTest").Bool() != (i < commands.Len()-1) {
				t.Fatal("only the yellow ground ring bypasses depth testing")
			}
			vertices := command.FieldByName("Vertices")
			if vertices.Len() != 22 || command.FieldByName("Indices").Len() != 60 {
				t.Fatal("Bowling Bash rings must have ten faces")
			}
			var center modelPoint3
			for j := 0; j < 10; j++ {
				vertex := vertices.Index(j * 2)
				center.x += vertex.FieldByName("X").Float() / 10
				center.z += vertex.FieldByName("Z").Float() / 10
			}
			if math.Abs(center.x-30.5) > .001 || math.Abs(center.z-20.5) > .001 {
				t.Fatal("Bowling Bash followed the victim after impact")
			}
		}
		actor := ctx.World.Actors[300]
		actor.X += 10
		actor.Dir = (actor.Dir + 2) % 8
		ctx.World.Actors[300] = actor
	}
}

func TestKnightRealAssetsAndRendering(t *testing.T) {
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		t.Skip("set GORO_DATA_DIR to check real Knight assets")
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
	m := &WorldMode{}
	for _, id := range []int{effectPierce, effectPierceSelf, effectBrandishSpear, effectBrandishSpear2, effectBowlingBash, effectBowlingSelf, effectSpearStabSelf, effectSpearBmrSelf, effectTwoHandQuicken, effectAutoCounter, effectSpearBoomerang} {
		spec, _ := worldEffectSpecForID(id)
		for _, component := range spec.components {
			for _, path := range append([]string{component.textureFile}, component.textureFiles...) {
				if path != "" && m.effectFileTexture(manager, path) == nil {
					t.Fatalf("effect %d missing texture %s", id, path)
				}
			}
			if component.textureName != "" && m.effectTexture(manager, component.textureName) == nil {
				t.Fatalf("effect %d missing texture %s", id, component.textureName)
			}
			if component.strFile != "" {
				str := m.loadWorldEffectSTR(manager, component.strFile, component.texturePath)
				if str == nil {
					t.Fatalf("missing STR %s", component.strFile)
				}
				for _, layer := range str.Layers {
					for _, path := range layer.Textures {
						if m.strEffectTexture(manager, path) == nil {
							t.Fatalf("STR %s missing %s", component.strFile, path)
						}
					}
				}
			}
		}
		for _, sound := range spec.sfx {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Fatal(err)
			}
		}
	}
	starts := time.Unix(100, 0)
	m.addWorldEffectBetweenAt(ctx, effectSpearBoomerang, 300, 2000000, starts)
	projection := newSceneProjectionForTarget(800, 600, 20, 20, 0)
	for i, tick := range []int{0, 4, 8, 12} {
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, starts.Add(time.Duration(tick)*db.EffectFrameDuration), false)
		if count := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); count != i+1 {
			t.Fatalf("tick %d drew %d spears, want %d", tick, count, i+1)
		}
	}
}
