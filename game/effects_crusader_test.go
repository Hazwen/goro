package game

import (
	"testing"
	"time"

	"github.com/kivutar/goro/network"
)

func TestDevotionSnapshotAndRemoval(t *testing.T) {
	ctx := knightEffectContext()
	m := &WorldMode{}
	m.addWorldEffectAt(ctx, effectDevotion, 2000000, time.Now())
	notify := network.ActorDevotion{SourceID: 2000000, Targets: [5]uint32{300, 300}}
	m.applyActorDevotion(ctx, notify)
	m.applyActorDevotion(ctx, notify)
	if len(m.worldEffects) != 2 || !m.worldEffects[1].expires.IsZero() || m.worldEffects[1].targetID != 300 {
		t.Fatalf("snapshots must replace links without duplicating them: %+v", m.worldEffects)
	}
	m.removeWorldEffectsForActor(300)
	if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectDevotion {
		t.Fatal("vanishing target left a link or removed the cast animation")
	}
	m.applyActorDevotion(ctx, notify)
	m.applyActorDevotion(ctx, network.ActorDevotion{SourceID: 2000000})
	if len(m.worldEffects) != 1 {
		t.Fatal("empty snapshot did not clear the caster's links")
	}
	m.applyActorDevotion(ctx, notify)
	m.removeWorldEffectsForActor(2000000)
	if len(m.worldEffects) != 0 {
		t.Fatal("vanishing caster left Devotion effects")
	}
}
