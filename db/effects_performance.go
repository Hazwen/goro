package db

import (
	"image/color"
	"time"
)

// Musical ground effects are shared by the Bard and Dancer job families.
func melodyBillboardEffectSpec(texture, wav string, tint color.RGBA) EffectSpec {
	return EffectSpec{
		Duration: 100 * time.Millisecond,
		SFX:      []string{wav},
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			TextureFile:    texture,
			Color:          tint,
			Duration:       100 * time.Millisecond,
			AlphaMax:       0.6,
			AttachedEntity: true,
			SizeStart:      effectTableSize(50),
			SizeEnd:        effectTableSize(50),
			PosZ:           0.5,
			Repeat:         true,
		}},
	}
}

func secra2EffectSpec() EffectSpec {
	components := make([]EffectComponent, 0, 10)
	for i := 0; i < 10; i++ {
		sizeStart := 850 - float64(i)*50
		tint := effectTableColor(1, 0.55+float64(i)*0.05, 0.55+float64(i)*0.05)
		components = append(components, EffectComponent{
			Kind:           EffectComponent3D,
			TextureFile:    "effect/priest_spell.bmp",
			Color:          tint,
			Duration:       1500 * time.Millisecond,
			AlphaMax:       0.3,
			BlendMode:      2,
			BlendAdditive:  true,
			FadeIn:         true,
			FadeOut:        true,
			AttachedEntity: true,
			PosZ:           7,
			SizeStart:      effectTableSize(sizeStart),
			SizeEnd:        effectTableSize(100),
			SizeSmooth:     true,
		})
	}
	return EffectSpec{
		Duration:   1500 * time.Millisecond,
		SFX:        []string{"effect\\ab_ancilla.wav"},
		Components: components,
	}
}

func songMusicEffectSpec(texture string, variant int) EffectSpec {
	spec := funcEffectSpec("SongMusic", 29999*EffectFrameDuration, false)
	c := &spec.Components[0]
	c.TextureFile, c.Animation = texture, variant
	switch variant {
	case 2:
		c.TextureFiles = []string{"effect/melody_a.bmp", "effect/melody_b.bmp"}
	case 5:
		c.TextureFiles = []string{"effect/레드젬스톤.bmp", "effect/블루젬스톤.bmp", "effect/옐로우젬스톤.bmp"}
	case 8:
		c.TextureFiles = []string{"effect/spell_01.bmp", "effect/spell_02.bmp", "effect/spell_03.bmp", "effect/spell_04.bmp", "effect/spell_05.bmp", "effect/spell_06.bmp", "effect/spell_07.bmp", "effect/spell_08.bmp"}
	}
	return spec
}

func songVerticalEffectSpec(texture string, variant int) EffectSpec {
	spec := funcEffectSpec("SongVertical", 29999*EffectFrameDuration, false)
	spec.Components[0].TextureFile = texture
	spec.Components[0].Animation = variant
	return spec
}

func songLightEffectSpec(variant int) EffectSpec {
	spec := funcEffectSpec("SongLight", 29999*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/twirl.bmp"
	spec.Components[0].Animation = variant
	return spec
}

func songOutEffectSpec() EffectSpec {
	spec := funcEffectSpec("SongOut", 29999*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/safeline.bmp"
	return spec
}
