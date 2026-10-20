package game

import (
	"math"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestLevel99AuraAddsNormalEffectsForLocalPlayer(t *testing.T) {
	mode := &WorldMode{}
	now := time.Unix(10, 0)
	ctx := level99AuraTestContext(99, false)

	mode.syncLevel99AuraEffects(ctx, now)

	if got := level99AuraEffectIDsForActor(mode.worldEffects, 2000000); !reflect.DeepEqual(got, level99AuraNormalEffects) {
		t.Fatalf("aura effects = %v, want %v", got, level99AuraNormalEffects)
	}
	for _, effect := range mode.worldEffects {
		if !effect.persistent {
			t.Fatalf("effect %d is not persistent", effect.effectID)
		}
		if effect.expires.Sub(now) < level99AuraLifetime {
			t.Fatalf("effect %d lifetime = %s, want at least %s", effect.effectID, effect.expires.Sub(now), level99AuraLifetime)
		}
	}
}

func TestLevel99AuraUsesSimpleEffectWhenLessEffectsEnabled(t *testing.T) {
	mode := &WorldMode{}
	now := time.Unix(10, 0)
	ctx := level99AuraTestContext(99, true)

	mode.syncLevel99AuraEffects(ctx, now)

	if got := level99AuraEffectIDsForActor(mode.worldEffects, 2000000); !reflect.DeepEqual(got, level99AuraSimpleEffects) {
		t.Fatalf("aura effects = %v, want %v", got, level99AuraSimpleEffects)
	}

	ctx.Session.LessEffects = false
	mode.syncLevel99AuraEffects(ctx, now.Add(time.Second))
	if got := level99AuraEffectIDsForActor(mode.worldEffects, 2000000); !reflect.DeepEqual(got, level99AuraNormalEffects) {
		t.Fatalf("normal aura effects after preference change = %v, want %v", got, level99AuraNormalEffects)
	}
}

func TestLevel99AuraRemovesEffectsBelowLevel(t *testing.T) {
	mode := &WorldMode{}
	now := time.Unix(10, 0)
	ctx := level99AuraTestContext(99, false)
	mode.syncLevel99AuraEffects(ctx, now)

	ctx.Session.Progress.BaseLevel = 98
	mode.syncLevel99AuraEffects(ctx, now.Add(time.Second))

	if got := level99AuraEffectIDsForActor(mode.worldEffects, 2000000); len(got) != 0 {
		t.Fatalf("aura effects = %v, want none", got)
	}
}

func TestLevel99AuraAddsEffectsForParsedRemoteActorLevel(t *testing.T) {
	mode := &WorldMode{}
	ctx := level99AuraTestContext(1, false)

	mode.upsertNetworkActor(ctx, network.ActorEntry{
		ID:       300,
		X:        12,
		Y:        34,
		Level:    99,
		HasLevel: true,
	})

	if got := level99AuraEffectIDsForActor(mode.worldEffects, 300); !reflect.DeepEqual(got, level99AuraNormalEffects) {
		t.Fatalf("remote aura effects = %v, want %v", got, level99AuraNormalEffects)
	}

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: actorVanishLogout})
	if got := level99AuraEffectIDsForActor(mode.worldEffects, 300); len(got) != 0 {
		t.Fatalf("remote aura effects after vanish = %v, want none", got)
	}
}

func TestLevel99AuraRemovedWhenDeadActorCleanupRemovesActor(t *testing.T) {
	mode := &WorldMode{}
	now := time.Unix(10, 0)
	ctx := level99AuraTestContext(1, false)
	ctx.World.Actors[300] = worldstate.Actor{
		ID:       300,
		X:        12,
		Y:        34,
		Level:    99,
		HasLevel: true,
	}
	mode.syncLevel99AuraEffects(ctx, now)
	mode.actorDeaths = map[uint32]time.Time{300: now.Add(-time.Second)}

	mode.cleanupDeadActors(ctx, now)

	if _, ok := ctx.World.Actors[300]; ok {
		t.Fatal("dead actor was not removed")
	}
	if got := level99AuraEffectIDsForActor(mode.worldEffects, 300); len(got) != 0 {
		t.Fatalf("remote aura effects after death cleanup = %v, want none", got)
	}
}

func TestLevel99BubbleSampleStartsHiddenThenRisesFromFeet(t *testing.T) {
	effect := worldEffect{effectID: effectLevel99Bubble, actorID: 2000000, starts: time.Unix(10, 0)}

	if sample := level99BubbleSample(effect, 0, 0, 0); sample.visible {
		t.Fatalf("fresh bubble sample is visible: %+v", sample)
	}

	firstVisibleFrame := -1.0
	for frame := 0.0; frame < 900; frame++ {
		if level99BubbleSample(effect, 0, 0, frame).visible {
			firstVisibleFrame = frame
			break
		}
	}
	if firstVisibleFrame < 0 {
		t.Fatal("bubble never reached its visible phase")
	}

	born := level99BubbleSample(effect, 0, 0, firstVisibleFrame)
	later := level99BubbleSample(effect, 0, 0, firstVisibleFrame+80)
	if !born.visible || !later.visible {
		t.Fatalf("visible samples missing: born=%+v later=%+v", born, later)
	}
	if born.height > level99BubbleRiseSpeed*level99BubbleGameToWorld+1e-9 {
		t.Fatalf("bubble starts at height %.4f, want near feet", born.height)
	}
	if later.height <= born.height {
		t.Fatalf("bubble height = %.4f after %.4f, want rising motion", later.height, born.height)
	}
	if math.Hypot(later.offsetX, later.offsetY) <= math.Hypot(born.offsetX, born.offsetY) {
		t.Fatalf("bubble drift = %.4f after %.4f, want wind spread while rising", math.Hypot(later.offsetX, later.offsetY), math.Hypot(born.offsetX, born.offsetY))
	}
}

func level99AuraTestContext(baseLevel int, lessEffects bool) client.Context {
	return client.Context{
		Session: &session.Session{
			AccountID:   2000000,
			CharID:      2000000,
			LessEffects: lessEffects,
			Progress: session.Progress{
				BaseLevel: baseLevel,
			},
			Selected: session.Character{
				ID:    2000000,
				Level: int16(baseLevel),
			},
		},
		World: &worldstate.World{
			Player: worldstate.Actor{ID: 2000000, X: 10, Y: 20},
			Actors: map[uint32]worldstate.Actor{},
		},
	}
}

func level99AuraEffectIDsForActor(effects []worldEffect, actorID uint32) []int {
	out := make([]int, 0, len(effects))
	for _, effect := range effects {
		if effect.actorID == actorID && isLevel99AuraEffect(effect.effectID) {
			out = append(out, effect.effectID)
		}
	}
	sort.Ints(out)
	return out
}

func isLevel99AuraEffect(effectID int) bool {
	for _, auraEffectID := range level99AuraNormalEffects {
		if effectID == auraEffectID {
			return true
		}
	}
	return false
}
