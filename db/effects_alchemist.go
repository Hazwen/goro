package db

func chemicalProtectionEffectSpec() EffectSpec {
	spec := funcEffectSpec("ChemicalProtection", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/shockwave_c.bmp"
	spec.SFX = []string{"apocalips_attack.wav"}
	return spec
}

func demonstrationEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 99999 * EffectFrameDuration,
		Components: []EffectComponent{{
			Kind: EffectComponent3D, SpriteFile: "데몬스트레이션",
			SpriteRepeat: true, SpriteDelay: 4 * EffectFrameDuration,
			WorldSizedSprite: true, PosZ: .2, AlphaMax: 200.0 / 255,
			SizeStart: 120 * EffectPixelRatio, SizeEnd: 120 * EffectPixelRatio,
		}},
	}
}

func twilightEffectSpec(variant int) EffectSpec {
	textures := [][]string{
		{"유저인터페이스/item/하얀허브.bmp"},
		{"유저인터페이스/item/화이트슬림포션.bmp"},
		{"유저인터페이스/item/화염병.bmp", "유저인터페이스/item/염산병.bmp", "유저인터페이스/item/알코올.bmp"},
	}
	return EffectSpec{
		Duration:   300 * EffectFrameDuration,
		Components: []EffectComponent{{Kind: EffectComponentFUNC, FuncName: "Twilight", TextureFiles: textures[variant]}},
	}
}
