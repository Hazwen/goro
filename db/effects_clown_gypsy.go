package db

import "time"

func arrowVulcanEffectSpec() EffectSpec {
	spec := funcEffectSpec("ArrowVulcan", 200*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/alpha_center.tga"
	spec.SFX = []string{"effect\\애로우 발칸.wav"}
	spec.SFXDelays = []time.Duration{30 * EffectFrameDuration}
	return spec
}

func moonlitEffectSpec() EffectSpec {
	spec := funcEffectSpec("Moonlit", 99999*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/fire_ivy.bmp"
	return spec
}

func marionetteEffectSpec() EffectSpec {
	spec := funcEffectSpec("Marionette", 99999999*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/alpha_center.tga"
	spec.SFX = []string{"effect\\소울 체인지.wav"}
	return spec
}

func hermodeEffectSpec() EffectSpec {
	spec := funcEffectSpec("Hermode", 29999*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/white02.bmp"
	return spec
}

func tarotCardEffectSpec(index int) EffectSpec {
	spec := funcEffectSpec("TarotCard", 300*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/tarot" + twoDigitString(index) + ".tga"
	return spec
}
