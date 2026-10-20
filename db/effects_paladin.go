package db

import (
	"image/color"
	"time"
)

const (
	effectShieldBoomerang3 = 520
	effectShrink           = 599
)

func shieldBoomerangEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 499 * EffectFrameDuration,
		SFX:      []string{"effect\\cru_shield boomerang.wav"},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "ShieldBoomerang", TextureFile: "effect/shield_boomerang.bmp",
		}},
	}
}

func shieldChainEffectSpec() EffectSpec {
	spec := shieldBoomerangEffectSpec()
	spec.Duration = 55 * EffectFrameDuration
	spec.Components[0].FuncName = "ShieldChain"
	return spec
}

func pressureEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 76 * EffectFrameDuration,
		SFX:      []string{"effect\\프레셔.wav"}, SFXDelays: []time.Duration{3 * EffectFrameDuration},
		CameraShake: 200 * time.Millisecond, CameraShakeDelay: 32 * EffectFrameDuration,
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "Pressure", TextureFile: "effect/cross_old.bmp",
			TextureFiles: []string{"effect/explosive_1_128.bmp"},
		}},
	}
}

func sacrificeEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 51 * EffectFrameDuration,
		SFX:      []string{"effect\\세크리파이스.wav"}, SFXDelays: []time.Duration{22 * EffectFrameDuration},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "Sacrifice", TextureFile: "effect/alpha_center.tga",
		}},
	}
}

func grandCrossEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 100 * EffectFrameDuration,
		SFX:      []string{"effect\\cru_grand cross.wav"},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "GrandCross",
			TextureFile: "effect/ring_white.tga", TextureFiles: []string{"effect/ring_yellow.tga"},
		}},
	}
}

func crusaderDefenseEffectSpec(texture string) EffectSpec {
	return EffectSpec{
		Duration: 200 * EffectFrameDuration,
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "CrusaderDefense", TextureFile: "effect/" + texture + ".tga",
		}},
	}
}

func providenceEffectSpec() EffectSpec {
	spec := strEffectSpecAttached("providence", "", false)
	spec.Duration = 90 * EffectFrameDuration
	spec.Components = append(spec.Components, EffectComponent{
		Kind: EffectComponentFUNC, FuncName: "Providence", TextureFile: "effect/ring_yellow.tga", Duration: spec.Duration,
	})
	return spec
}

func shrinkEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 26 * EffectFrameDuration,
		SFX:      []string{"effect\\EF_BeginSpell.wav"}, SFXDelays: []time.Duration{5 * EffectFrameDuration},
	}
}

func devotionLinkEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: time.Second,
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "DevotionLink", TextureFile: "effect/cloud11.tga", Repeat: true,
		}},
	}
}

func devotionEffectSpec() EffectSpec {
	spec := strEffectSpecAttached("devotion", "", false)
	// ProcessEZ2STR uses -20 pixels for Devotion, versus the usual -80.
	spec.Components[0].SpriteYOffset = 60
	return spec
}

func guardEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 44 * EffectFrameDuration,
		SFX:      []string{"effect\\kyrie_guard.wav"},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "Guard",
			TextureFile: "effect/guardk.tga", TextureFiles: []string{"effect/guardk2.tga"},
		}},
	}
}

func gospelGroundEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1500 * time.Millisecond,
		Components: []EffectComponent{
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "FlatColorTile",
				Color:          color.RGBA{R: 255, G: 255, B: 255, A: 13},
				SizeStart:      1,
				RenderBefore:   true,
				AttachedEntity: false,
			},
			{
				Kind: EffectComponentFUNC, FuncName: "GospelCross", TextureFile: "effect/cross_old.bmp", Repeat: true,
			},
		},
	}
}
