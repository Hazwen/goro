package db

import (
	"time"
)

const effectPierceBody = 502

func auraBladeEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 60 * EffectFrameDuration,
		SFX:      []string{"effect\\오라 블레이드.wav"},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "BlueCasting",
			TextureFile: "effect/ring_white.tga", TextureFiles: []string{"effect/ring_yellow.tga"},
			AttachedEntity: true,
		}},
	}
}

func magnum2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration:  100 * EffectFrameDuration,
		SFX:       []string{"permeter_attack.wav", "effect\\ef_magnumbreak.wav"},
		SFXDelays: []time.Duration{20 * EffectFrameDuration, 30 * EffectFrameDuration},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "SpiralPierce", TextureFile: "effect/ring_yellow.tga",
			AttachedEntity: true,
		}},
	}
}
