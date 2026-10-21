package game

import (
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

func merchantEffectContext() client.Context {
	w := world.New()
	w.Player = world.Actor{ID: 100, Job: db.JobMerchant, X: 10, Y: 20}
	w.Actors[300] = world.Actor{ID: 300, Job: 1002, X: 11, Y: 20}
	return client.Context{World: w, Resources: &res.Manager{}, Session: &session.Session{AccountID: 2000000, CharID: 100}}
}

func TestMerchantAttackEffects(t *testing.T) {
	ctx := merchantEffectContext()
	start := time.Unix(100, 0)
	hit := start.Add(500 * time.Millisecond)
	for _, skill := range []uint16{db.SkillMCMammonite, db.SkillMCCartrevolution} {
		for _, damage := range []int32{0, 100} {
			m := &WorldMode{}
			action := network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Damage: damage, Action: network.ActorActionSkill}
			m.addSkillBeginEffect(ctx, action, start)
			m.addSkillBeforeHitEffect(ctx, action, start)
			m.addSkillEffect(ctx, action, start, hit)
			m.addSkillHitEffect(ctx, action, hit)
			begin, impact := effectBashBegin, effectMammonite
			if skill == db.SkillMCCartrevolution {
				begin, impact = effectCartRevolution, effectCartRevolution
			}
			want := 1
			if damage > 0 {
				want += 2
			}
			if len(m.worldEffects) != want || m.worldEffects[0].effectID != begin || m.worldEffects[0].actorID != 2000000 || !m.worldEffects[0].starts.Equal(start) {
				t.Fatalf("skill %d damage %d: wrong caster effects: %+v", skill, damage, m.worldEffects)
			}
			if damage > 0 {
				if e := m.worldEffects[2]; e.effectID != impact || e.actorID != 300 || !e.starts.Equal(hit) {
					t.Fatalf("skill %d: impact must appear on the victim at hit time: %+v", skill, e)
				}
			}
			if skill == db.SkillMCCartrevolution {
				for i, effect := range []worldEffect{m.worldEffects[0], m.worldEffects[len(m.worldEffects)-1]} {
					if damage == 0 && i == 1 {
						break
					}
					for j, tick := range []int{7, 20} {
						if sound := m.scheduledSounds[i*2+j]; !sound.at.Equal(effect.starts.Add(time.Duration(tick) * db.EffectFrameDuration)) {
							t.Fatalf("Cart Revolution sound must coincide with burst %d: %+v", tick, sound)
						}
					}
				}
			}
			before, sounds := len(m.worldEffects), len(m.scheduledSounds)
			action.Action = 5 // Secondary splash notification.
			m.addSkillBeginEffect(ctx, action, start)
			m.addSkillBeforeHitEffect(ctx, action, start)
			m.addSkillEffect(ctx, action, start, hit)
			m.addSkillHitEffect(ctx, action, hit)
			if len(m.worldEffects) != before || len(m.scheduledSounds) != sounds {
				t.Fatal("splash victims must not replay the skill")
			}
		}
	}
	for _, weapon := range []int16{0, 1, 6} {
		actor := ctx.World.Player
		actor.Weapon = weapon
		if got := skillAction(db.SkillMCMammonite).actionFamilyForActorWithResources(nil, actor); got != attackActionFamilyForActorWithResources(nil, actor) {
			t.Fatal("Mammonite must use the equipped weapon's normal attack animation")
		}
	}
}

func TestMerchantUtilityEffects(t *testing.T) {
	ctx := merchantEffectContext()
	m := &WorldMode{}
	notify := network.SkillNoDamageNotify{SkillID: db.SkillMCLoud, SourceID: 2000000, TargetID: 300}
	m.applySkillNoDamageNotify(ctx, notify)
	if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
		t.Fatal("failed Crazy Uproar must be silent")
	}
	notify.Result = 1
	m.applySkillNoDamageNotify(ctx, notify)
	if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectLoud || m.worldEffects[0].actorID != 2000000 || len(m.scheduledSounds) != 1 {
		t.Fatalf("Crazy Uproar must show only Loud on its caster: %+v", m.worldEffects)
	}
	for _, skill := range []uint16{db.SkillMCInccarry, db.SkillMCDiscount, db.SkillMCOvercharge, db.SkillMCPushcart, db.SkillMCIdentify, db.SkillMCVending, db.SkillMCChangecart} {
		m := &WorldMode{}
		notify.SkillID = skill
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects)+len(m.scheduledSounds) != 0 {
			t.Fatalf("utility/passive skill %d must not invent an effect", skill)
		}
	}
}

func TestMammoniteCoinBounces(t *testing.T) {
	ctx := merchantEffectContext()
	ctx.World.GAT = &res.GAT{Width: 30, Height: 30, Cells: make([]res.GATCell, 900)}
	for i := range ctx.World.GAT.Cells {
		ctx.World.GAT.Cells[i].Heights = [4]float32{4, 4, 4, 4}
	}
	m := &WorldMode{}
	start := time.Unix(100, 0)
	m.addWorldEffectAt(ctx, effectMammonite, 300, start)
	effect := m.worldEffects[0]
	if effect.coinTrails == nil {
		t.Fatal("missing five coin trajectories")
	}
	bounces := 0
	for _, trail := range effect.coinTrails {
		if len(trail) < 40 || len(trail) > 69 || trail[0].y <= 6 || trail[5].y <= trail[0].y {
			t.Fatal("coins must rise from ten native units above the victim's ground height")
		}
		for age := 1; age < len(trail); age++ {
			if trail[age].y < 4 && trail[age].y < trail[age-1].y {
				bounces++
				if age+1 < len(trail) && trail[age+1].y <= trail[age].y {
					t.Fatal("coin failed to bounce off terrain")
				}
			}
		}
	}
	if bounces < 5 || len(m.scheduledSounds) != bounces {
		t.Fatalf("one sound per bounce: bounces=%d sounds=%d", bounces, len(m.scheduledSounds))
	}
	for _, sound := range m.scheduledSounds {
		if !sound.at.After(start.Add(20 * db.EffectFrameDuration)) {
			t.Fatal("coin audio must wait until a coin hits the ground")
		}
	}
	ctx.Session.LessEffects = true
	m = &WorldMode{}
	m.addWorldEffectAt(ctx, effectMammonite, 300, start)
	if m.worldEffects[0].coinTrails != nil || len(m.scheduledSounds) != 0 {
		t.Fatal("minimal Mammonite must omit coins and bounce sounds")
	}
}

func TestMerchantEffectsWithClientAssets(t *testing.T) {
	ctx := merchantEffectContext()
	ctx.Resources = realDataManager(t)
	m := &WorldMode{}
	start := time.Unix(100, 0)
	for _, id := range []int{effectMammonite, effectCartRevolution, effectLoud} {
		m.addWorldEffectAt(ctx, id, 300, start)
		spec, _ := worldEffectSpecForID(id)
		for _, component := range spec.components {
			if component.strFile != "" && m.loadWorldEffectSTR(ctx.Resources, component.strFile, component.texturePath) == nil {
				t.Fatalf("missing STR %q", component.strFile)
			}
		}
	}
	for _, sound := range m.scheduledSounds {
		for _, path := range sound.paths {
			if _, err := ctx.Resources.ReadFile("data/wav/" + path); err != nil {
				t.Error(err)
			}
		}
	}
	projection := newSceneProjectionForTarget(800, 600, 11, 20, 0)
	coin, _ := worldEffectSpecForID(effectMammonite)
	for _, tc := range []struct{ tick, count int }{{19, 0}, {20, 5}, {21, 10}, {35, 10}, {90, 0}} {
		frame := render.NewFrame(800, 600)
		m.drawMammoniteCoins(frame, ctx, projection, coin.components[1], m.worldEffects[0], start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		if count := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); count != tc.count {
			t.Fatalf("tick %d: got %d coins/afterimages, want %d", tc.tick, count, tc.count)
		}
	}
	cart, _ := worldEffectSpecForID(effectCartRevolution)
	for _, tc := range []struct{ tick, count int }{{6, 0}, {8, 2}, {21, 4}, {30, 2}, {40, 0}} {
		frame := render.NewFrame(800, 600)
		m.drawCartRevolutionEffect(frame, ctx, cart.components[1], m.worldEffects[1], start.Add(time.Duration(tc.tick)*db.EffectFrameDuration))
		if count := reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len(); count != tc.count {
			t.Fatalf("tick %d: got %d burst meshes, want %d", tc.tick, count, tc.count)
		}
	}
}
