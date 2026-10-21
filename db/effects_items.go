package db

import (
	"image/color"
	"time"
)

func berserkPotionEffectSpec() EffectSpec {
	spec := strEffectSpecAttached("버서크", "effect\\ac_concentration.wav", false)
	spec.Duration = 151 * EffectFrameDuration
	spec.SFXDelays = []time.Duration{18 * EffectFrameDuration}
	spec.CameraShake = 200 * time.Millisecond
	spec.CameraShakeDelay = 18 * EffectFrameDuration
	spec.Components = append(spec.Components, EffectComponent{
		Kind: EffectComponentFUNC, FuncName: "BerserkPotion", AttachedEntity: true,
		TextureFile: "effect/alpha_down.tga", TextureFiles: []string{"effect/ac_center2.tga"},
	})
	return spec
}

func potionEffectSpec(file string, c color.RGBA) EffectSpec {
	return EffectSpec{
		Duration: 850 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponentSTR,
			Color:          c,
			STRFile:        file,
			AttachedEntity: true,
		}},
	}
}

func healingPotionEffectSpec(file string, c color.RGBA) EffectSpec {
	spec := potionEffectSpec(file, c)
	spec.SFX = []string{"_heal_effect.wav"}
	return spec
}

func bluePotionEffectSpec() EffectSpec {
	spec := potionEffectSpec("파란포션", color.RGBA{R: 92, G: 150, B: 255, A: 255})
	spec.SFX = []string{"effect\\흡기.wav"}
	return spec
}
