package game

import (
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func TestBlueCastingFlaresDuringCastAndEndsWithIt(t *testing.T) {
	ctx := soulLinkerEffectContext()
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	start := time.Unix(100, 0)
	for _, duration := range []int{6, 60, 180} {
		m.worldEffects = nil
		m.addWorldEffectBetweenAtDuration(ctx, effectBlueCasting, 300, 0, start, time.Duration(duration)*db.EffectFrameDuration)
		e := m.worldEffects[0]
		duration = max(duration, 56)
		if e.expires.Sub(e.starts) != time.Duration(duration)*db.EffectFrameDuration {
			t.Fatal("short casts need the native minimum flare duration")
		}
		spec, _ := worldEffectSpecForID(effectBlueCasting)
		frame := render.NewFrame(800, 600)
		m.drawBlueCastingEffect(frame, ctx, spec.components[0], e, 18.5, 20.5, 0, start.Add(15*db.EffectFrameDuration))
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() != 1 {
			t.Fatal("missing cast rings")
		}
		one, many := &blueCasting{}, &blueCasting{}
		one.advance(duration, duration)
		for i := 0; i <= duration; i++ {
			many.advance(i, duration)
		}
		if !reflect.DeepEqual(one, many) {
			t.Fatal("cast rings depend on render rate")
		}
		frame = render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, newSceneProjectionForTarget(800, 600, 18.5, 20.5, 0), start.Add(time.Duration(duration+1)*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() != 0 || len(m.worldEffects) != 0 {
			t.Fatal("rings survived the cast")
		}
	}
}

func TestCastRingEffectSpecUsesMagicRingCylinder(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectCastRing)
	if !ok {
		t.Fatal("cast ring effect missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentFUNC || component.funcAdapter != effectFuncCastRing || component.funcName != "CastRing" || component.textureName != "ring_yellow" {
		t.Fatalf("component = %+v", component)
	}
	if component.bottomSize != 0.8 || component.topSize != 2.45 || component.height != 2.8 {
		t.Fatalf("magic ring dimensions = bottom %.2f top %.2f height %.2f", component.bottomSize, component.topSize, component.height)
	}
	if component.duration != 0 {
		t.Fatalf("cast ring component duration = %s, want inherited cast duration", component.duration)
	}
}

func TestBeginSpellEffectsInheritCastDuration(t *testing.T) {
	for _, effectID := range []int{effectBeginSpell, effectBeginSpell2, effectBeginSpell3, effectBeginSpell4, effectBeginSpell5, effectBeginSpell6, effectBeginSpell7} {
		spec, ok := worldEffectSpecForID(effectID)
		if !ok {
			t.Fatalf("begin spell effect %d missing", effectID)
		}
		for i, component := range spec.components {
			if component.duration != 0 {
				t.Fatalf("effect %d component %d duration = %s, want inherited cast duration", effectID, i, component.duration)
			}
		}
	}
}
