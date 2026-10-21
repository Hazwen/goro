package db

func priestGroundEffectSpec(effectID int) EffectSpec {
	spec := funcEffectSpec("PriestGround", 9999*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/alpha_down.tga"
	if effectID == effectBottomMagnus {
		spec.Components[0].TextureFile = "effect/ring_red.tga"
	}
	if effectID == effectBottomBasilica {
		spec.Duration = 29999 * EffectFrameDuration
	}
	return spec
}
