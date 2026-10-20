package db

import (
	"image/color"
	"time"
)

func blueCastingEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 60 * EffectFrameDuration, SFX: []string{"effect\\EF_BeginSpell.wav"},
		Components: []EffectComponent{{Kind: EffectComponentFUNC, FuncName: "BlueCasting", TextureFile: "effect/ring_blue.tga", AttachedEntity: true}},
	}
}

func coupleCastingEffectSpec() EffectSpec {
	spec := blueCastingEffectSpec()
	spec.Duration = 1000 * EffectFrameDuration
	spec.Components[0].TextureFile = "effect/ring_red.tga"
	return spec
}

func castAuraEffectSpec(texture string, tint color.RGBA, alphaMax, height, topSize float64, rotate bool) EffectSpec {
	return EffectSpec{
		Duration: 900 * time.Millisecond,
		SFX:      []string{"effect\\ef_beginspell.wav"},
		Components: []EffectComponent{{
			Kind:             EffectComponentCylinder,
			TextureName:      texture,
			AlphaMax:         alphaMax,
			Fade:             true,
			Rotate:           rotate,
			Animation:        2,
			BottomSize:       1,
			TopSize:          topSize,
			Height:           height,
			TotalCircleSides: 32,
			CircleSides:      32,
			Color:            tint,
		}},
	}
}

func elementalCastAuraEffectSpec(texture string, tint color.RGBA, alphaMax float64) EffectSpec {
	return EffectSpec{
		Duration: 900 * time.Millisecond,
		SFX:      []string{"effect\\ef_beginspell.wav"},
		Components: []EffectComponent{
			{
				Kind:             EffectComponentCylinder,
				TextureName:      texture,
				AlphaMax:         0.3,
				Fade:             true,
				Rotate:           true,
				Animation:        1,
				BottomSize:       1,
				TopSize:          1,
				Height:           30,
				TotalCircleSides: 32,
				CircleSides:      32,
				Color:            tint,
			},
			{
				Kind:             EffectComponentCylinder,
				TextureName:      texture,
				AlphaMax:         alphaMax,
				Fade:             true,
				Rotate:           true,
				Animation:        1,
				BottomSize:       1,
				TopSize:          1.3,
				Height:           2,
				TotalCircleSides: 32,
				CircleSides:      32,
				Color:            tint,
			},
			{
				Kind:             EffectComponentCylinder,
				TextureName:      texture,
				AlphaMax:         alphaMax,
				Fade:             true,
				Rotate:           true,
				Animation:        2,
				BottomSize:       1,
				TopSize:          4,
				Height:           3,
				TotalCircleSides: 32,
				CircleSides:      32,
				Color:            tint,
			},
		},
	}
}

func darkCastingEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 900 * time.Millisecond,
		SFX:      []string{"effect\\ef_beginspell.wav"},
		Components: []EffectComponent{{
			Kind:             EffectComponentCylinder,
			TextureName:      "ring_black",
			Color:            color.RGBA{R: 255, G: 255, B: 255, A: 255},
			AlphaMax:         0.8,
			Animation:        2,
			AttachedEntity:   true,
			BlendMode:        2,
			BlendAdditive:    true,
			BottomSize:       1,
			TopSize:          5,
			Height:           4,
			Fade:             true,
			Rotate:           true,
			TotalCircleSides: 32,
			CircleSides:      32,
		}},
	}
}
