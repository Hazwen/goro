package db

func agilityEffectSpec(decrease bool) EffectSpec {
	spec := funcEffectSpec("Agility", 100*EffectFrameDuration, true)
	spec.SFX = []string{"effect\\ef_incagility.wav"}
	spec.Components[0].TextureFile = "effect/agi_up.bmp"
	spec.Components[0].TextureFiles = []string{"effect/ac_center2.tga"}
	if decrease {
		spec.SFX = []string{"effect\\ef_decagility.wav"}
		spec.Components[0].TextureFile = "effect/slow.bmp"
	}
	return spec
}
