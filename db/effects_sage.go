package db

import "time"

func sageGroundCastEffectSpec(texture string, ticks int) EffectSpec {
	spec := funcEffectSpec("SageGroundCast", time.Duration(ticks)*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/" + texture + ".tga"
	return spec
}

func propertyGroundEffectSpec(texture string) EffectSpec {
	spec := funcEffectSpec("PropertyGround", 29999*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/" + texture + ".tga"
	return spec
}

func landProtectorGroundEffectSpec() EffectSpec {
	spec := funcEffectSpec("LandProtectorGround", 29999*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/aaa copy.bmp"
	return spec
}

func sageEndowEffectSpec(file string) EffectSpec {
	spec := strEffectSpecAttached(file, "_enemy_hit_wind1.wav", false)
	spec.SFXDelays = []time.Duration{10 * EffectFrameDuration}
	return spec
}
