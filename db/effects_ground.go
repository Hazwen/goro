package db

import "time"

func groundWallsEffectSpec(effectID int) EffectSpec {
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

func groundTextureEffectSpec(texture string) EffectSpec {
	return EffectSpec{
		Duration: 1500 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "GroundTexture",
			TextureFile:    texture,
			Duration:       1500 * time.Millisecond,
			SizeStart:      1,
			SizeEnd:        1,
			PosZ:           0.05,
			BlendAdditive:  true,
			AttachedEntity: false,
		}},
	}
}

func propertyGroundEffectSpec(texture string) EffectSpec {
	spec := funcEffectSpec("PropertyGround", 29999*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/" + texture + ".tga"
	return spec
}
