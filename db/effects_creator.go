package db

import "time"

func slimPitcherEffectSpec(texture string) EffectSpec {
	spec := funcEffectSpec("SlimPitcher", 250*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "유저인터페이스/" + texture + ".bmp"
	spec.Components[0].TextureFiles = []string{"effect/bbbb.bmp"}
	return spec
}

func acidBottlesEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 300 * EffectFrameDuration,
		Components: []EffectComponent{
			{Kind: EffectComponentFUNC, FuncName: "ThrowItem", TextureFile: "유저인터페이스/item/화염병.bmp", Delay: 5 * EffectFrameDuration},
			{Kind: EffectComponentFUNC, FuncName: "ThrowItem", TextureFile: "유저인터페이스/item/염산병.bmp", Delay: 10 * EffectFrameDuration},
		},
	}
}

func acidDemonEffectSpec() EffectSpec {
	spec := funcEffectSpec("AcidDemon", 200*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/ring_black.tga"
	spec.SFX = []string{"effect\\EF_FireWall.wav"}
	spec.CameraShake = 200 * time.Millisecond
	spec.CameraShakeDelay = 5 * EffectFrameDuration
	return spec
}
