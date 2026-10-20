package db

import "time"

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

func lordKnightBurstEffectSpec(sound string, delay time.Duration) EffectSpec {
	return EffectSpec{
		Duration: 51 * EffectFrameDuration,
		SFX:      []string{sound}, SFXDelays: []time.Duration{delay},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "Sacrifice", TextureFile: "effect/alpha_center.tga",
		}},
	}
}
