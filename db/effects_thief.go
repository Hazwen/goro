package db

func stealEffectSpec() EffectSpec {
	spec := funcEffectSpec("Steal", 50*EffectFrameDuration, false)
	spec.Components[0].SpriteFile = "particle7"
	spec.SFX = []string{"effect\\ef_steal.wav"}
	return spec
}

func detoxicationEffectSpec() EffectSpec {
	spec := funcEffectSpec("RisingParticles", 100*EffectFrameDuration, false)
	spec.Components[0].SpriteFile = "particle2"
	spec.SFX = []string{"effect\\ef_detoxication.wav"}
	return spec
}

func sprinkleSandEffectSpec() EffectSpec {
	spec := funcEffectSpec("SprinkleSand", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/thunder_center.bmp"
	return spec
}
