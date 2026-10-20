package game

import (
	"math"
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

func thiefEffectContext() client.Context {
	w := world.New()
	w.Player = world.Actor{ID: 100, X: 10, Y: 20, Job: db.JobThief}
	w.Actors[300] = world.Actor{ID: 300, X: 16, Y: 20, Job: 1002}
	return client.Context{World: w, Resources: &res.Manager{}, Session: &session.Session{AccountID: 2000000, CharID: 100}}
}

func TestThiefSkillPacketEffects(t *testing.T) {
	ctx := thiefEffectContext()
	for _, tc := range []struct {
		skill  uint16
		effect int
		sound  string
	}{
		{db.SkillTFSteal, effectSteal, "effect\\ef_steal.wav"},
		{db.SkillTFDetoxify, effectDetoxication, "effect\\ef_detoxication.wav"},
	} {
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: tc.skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects) != 0 || len(m.scheduledSounds) != 0 {
			t.Fatalf("failed skill %d produced an effect", tc.skill)
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != tc.effect || m.worldEffects[0].actorID != 300 {
			t.Fatalf("skill %d must affect its target once: %+v", tc.skill, m.worldEffects)
		}
		spec, _ := worldEffectSpecForID(tc.effect)
		if len(m.scheduledSounds) != 1 || len(spec.sfx) != 1 || spec.sfx[0] != tc.sound {
			t.Fatalf("skill %d has wrong audio: %+v", tc.skill, m.scheduledSounds)
		}
	}
	starts := time.Unix(100, 0)
	hitAt := starts.Add(500 * time.Millisecond)
	for _, skill := range []uint16{db.SkillTFPoison, db.SkillTFSprinklesand, db.SkillTFThrowstone} {
		m := &WorldMode{}
		action := network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Damage: 50}
		m.addSkillBeforeHitEffect(ctx, action, starts)
		m.addSkillEffect(ctx, action, starts, hitAt)
		m.addSkillHitEffect(ctx, action, hitAt)
		want, at, actor := effectBashHit, hitAt, uint32(300)
		if skill == db.SkillTFSprinklesand {
			want, at = effectSprinkleSand, starts
		} else if skill == db.SkillTFThrowstone {
			want, at, actor = effectThrowItem3, starts, 2000000
		}
		if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != want || m.worldEffects[0].actorID != actor || !m.worldEffects[0].starts.Equal(at) {
			t.Fatalf("skill %d has wrong effect or timing: %+v", skill, m.worldEffects)
		}
		if skill == db.SkillTFPoison {
			m.worldEffects = nil
			action.Damage = 0
			m.addSkillHitEffect(ctx, action, hitAt)
			if len(m.worldEffects) != 0 {
				t.Fatal("missed Envenom must not show an impact")
			}
		}
	}
}

func TestThiefParticleMotion(t *testing.T) {
	effect := worldEffect{effectID: effectSteal, starts: time.Unix(100, 0)}
	for i := 0; i < 10; i++ {
		first, middle, last := stealParticleAt(effect, i, 0), stealParticleAt(effect, i, 24), stealParticleAt(effect, i, 48)
		if middle.position.y <= first.position.y || last.position.y >= middle.position.y || last.size >= first.size || last.alpha >= first.alpha {
			t.Fatalf("Steal particle %d must rise, fall, shrink and fade: %+v / %+v / %+v", i, first, middle, last)
		}
		if p := stealParticleAt(effect, i, 49); p.alpha != 0 || p.size != 0 {
			t.Fatal("Steal must finish at fifty ticks")
		}
	}
	effect.effectID = effectDetoxication
	first, last := risingParticleAt(effect, 0, 0), risingParticleAt(effect, 0, 38)
	if first.alpha <= 0 || last.position.y <= first.position.y || last.angle <= first.angle || risingParticleAt(effect, 0, 39).alpha != 0 {
		t.Fatal("Detoxify particles must rise, rotate and fade over forty ticks")
	}
	effect.effectID = effectSprinkleSand
	for i := 0; i < 80; i++ {
		if sprinkleSandParticleAt(effect, i, 0).alpha != 0 || sprinkleSandParticleAt(effect, i, 20).alpha <= 0 || sprinkleSandParticleAt(effect, i, 65).alpha != 0 {
			t.Fatalf("Sand Attack particle %d has wrong delay or fade", i)
		}
	}
}

func TestThiefFindStoneAnimation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		frames   int
		duration time.Duration
	}{
		{"unloaded sprite", 0, 300 * time.Millisecond},
		{"three frames", 3, 300 * time.Millisecond},
		{"four frames", 4, 400 * time.Millisecond},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := thiefEffectContext()
			m := &WorldMode{}
			if tc.frames > 0 {
				m.playerView = humanoidTimingView(spriteActionPickup, tc.frames)
				for i := range m.playerView.body.act.Actions {
					m.playerView.body.act.Actions[i].DelayMS = 100
				}
			}
			notify := network.SkillCastNotify{SkillID: db.SkillTFPickstone, SourceID: 2000000, TargetID: 2000000, DelayTime: 500}
			m.applySkillCastNotify(ctx, notify)
			for _, id := range []uint32{2000000, 100} {
				anim := m.actorAnims[id]
				if anim.actionFamily != spriteActionPickup || anim.duration != tc.duration || anim.hasSpeed {
					t.Fatalf("Find Stone must play pickup once at the sprite's own speed: %+v", anim)
				}
				end := anim.started.Add(tc.duration)
				if _, active := m.actorAnimation(id, end.Add(-time.Millisecond)); !active {
					t.Fatal("pickup ended before its last frame finished")
				}
				if _, active := m.actorAnimation(id, end); active {
					t.Fatal("pickup must end before the cast finishes")
				}
				bar := m.actorCastBars[id]
				if _, active := actorCastBarProgress(bar, end); !active || bar.duration != 500*time.Millisecond {
					t.Fatalf("cast bar must retain the server duration: %+v", bar)
				}
			}
			// A fast cast can finish before the pickup animation does.
			notify.DelayTime = 100
			m.applySkillCastNotify(ctx, notify)
			m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillTFPickstone, SourceID: 2000000, TargetID: 2000000, Result: 1})
			for _, id := range []uint32{2000000, 100} {
				if anim := m.actorAnims[id]; anim.actionFamily != spriteActionIdle {
					t.Fatalf("Find Stone must return to standing on success: %+v", anim)
				}
			}
		})
	}
}

func TestStoneFlingFlight(t *testing.T) {
	ctx := thiefEffectContext()
	m := &WorldMode{}
	starts := time.Unix(100, 0)
	m.addSkillBeforeHitEffect(ctx, network.ActorActionNotify{SkillID: db.SkillTFThrowstone, SourceID: 2000000, TargetID: 300}, starts)
	effect := m.worldEffects[0]
	first, angle, alpha := throwItemAt(effect, 1)
	if math.Abs(first.x-10.9) > 1e-9 || angle != degreesToRadians(10) || alpha != 255 {
		t.Fatalf("stone must advance two native units and rotate ten degrees per tick: %+v, %g, %d", first, angle, alpha)
	}
	middle, _, _ := throwItemAt(effect, 8)
	if math.Abs(middle.y-effect.actorOrigin.y-2) > 1e-9 {
		t.Fatal("stone's arc must rise ten native units above its launch height")
	}
	near := effect
	near.projectileDirection.x = 1.2
	_, _, nearAlpha := throwItemAt(near, 10)
	_, _, farAlpha := throwItemAt(effect, 10)
	if nearAlpha >= farAlpha {
		t.Fatal("stone flight duration must depend on distance")
	}
	m.textures = map[string]*render.Image{"__effectfile_유저인터페이스/item/돌.bmp|blackkey=true": render.NewImage(8, 8)}
	frame := render.NewFrame(800, 600)
	m.drawWorldEffects(frame, ctx, newSceneProjectionForTarget(800, 600, 14, 20, 0), starts.Add(6*db.EffectFrameDuration), false)
	if n := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); n != 4 {
		t.Fatalf("stone and three afterimages: got %d", n)
	}
}

func TestThiefEffectsWithClientAssets(t *testing.T) {
	manager := realDataManager(t)
	ctx := thiefEffectContext()
	ctx.Resources = manager
	m := &WorldMode{}
	starts := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 14, 20, 0)
	for _, tc := range []struct{ id, frame, count int }{
		{effectSteal, 20, 10}, {effectDetoxication, 20, 5}, {effectDetoxication, 98, 1},
		{effectSprinkleSand, 20, 80}, {effectThrowItem3, 6, 4},
	} {
		m.worldEffects = nil
		m.addWorldEffectBetweenAt(ctx, tc.id, 300, 2000000, starts)
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, starts.Add(time.Duration(tc.frame)*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if commands.Len() != tc.count {
			t.Fatalf("effect %d tick %d: want %d particles, got %d", tc.id, tc.frame, tc.count, commands.Len())
		}
		spec, _ := worldEffectSpecForID(tc.id)
		for _, sound := range spec.sfx {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Error(err)
			}
		}
	}
}
