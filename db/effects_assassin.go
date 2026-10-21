package db

import (
	"time"
)

func poisonParticlesEffectSpec(enchant bool) EffectSpec {
	spec := funcEffectSpec("RisingParticles", 10*EffectFrameDuration, false)
	spec.Components[0].SpriteFile = "particle3"
	if enchant {
		spec.Duration = 250 * EffectFrameDuration
		spec.Components[0].Duration = spec.Duration
		spec.SFX = []string{"effect\\assasin_enchantpoison.wav", "effect\\ef_poisonattack.wav"}
	}
	return spec
}

func venomSplasherEffectSpec() EffectSpec {
	spec := strEffectSpecAttached("venomsplasher", "effect\\assasin_venomsplasher.wav", false)
	spec.SFXDelays = []time.Duration{10 * EffectFrameDuration}
	return spec
}

func sonicBlowEffectSpec() EffectSpec {
	spec := funcEffectSpec("SonicBlow", 40*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/alpha_center.tga"
	spec.SFX = []string{"effect\\ef_stonecurse.wav"}
	return spec
}

func sonicBlowHitEffectSpec() EffectSpec {
	ring := hitRingComponent("magic_red", 12, 4, 7, 3.5, 0.4, 6)
	ring.PosZ = 11.0 / 5
	ring.HitRing.InitialDistance = 2.0 / 5
	ring.HitRing.RandomImpactOffset = true
	return EffectSpec{
		Duration:   ring.Duration,
		SFX:        []string{"effect\\assasin_sonicblow.wav"},
		Components: []EffectComponent{ring},
	}
}

func grimtoothEffectSpec(impact bool) EffectSpec {
	spec := funcEffectSpec("Grimtooth", 250*EffectFrameDuration, false)
	if impact {
		spec.Duration = 1000 * EffectFrameDuration
		spec.Components[0].Duration = spec.Duration
	} else {
		spec.SFX = []string{"effect\\ef_frostdiver.wav"}
	}
	spec.Components[0].TextureFile = "effect/stone.bmp"
	return spec
}
