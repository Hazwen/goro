package db

import (
	"image/color"
	"strconv"
	"time"
)

func rollingCutterCounterEffectSpec(index int) EffectSpec {
	texture := "effect/회전카운터" + strconv.Itoa(index) + ".tga"
	return EffectSpec{
		Duration: time.Second,
		Components: []EffectComponent{
			rollingCutterCounterComponent(texture, 1.0, 1.0, 1.0, 1.0, 200, 1),
			rollingCutterCounterComponent(texture, 0.2, 0.7, 0.7, 1.0, 220, 2),
			rollingCutterCounterComponent(texture, 0.2, 0.5, 0.5, 1.0, 240, 2),
			rollingCutterCounterComponent(texture, 0.2, 0.3, 0.3, 1.0, 260, 2),
			rollingCutterCounterComponent(texture, 0.2, 0.1, 0.1, 1.0, 280, 2),
		},
	}
}

func rollingCutterCounterComponent(texture string, alphaMax, red, green, blue, sizeStart float64, blendMode int) EffectComponent {
	return EffectComponent{
		Kind:           EffectComponent3D,
		TextureFile:    texture,
		Color:          color.RGBA{R: uint8(red * 255), G: uint8(green * 255), B: uint8(blue * 255), A: 255},
		Duration:       time.Second,
		AlphaMax:       alphaMax,
		FadeIn:         true,
		FadeOut:        true,
		PosZ:           4,
		SizeStart:      effectTableSize(sizeStart),
		SizeEnd:        effectTableSize(20),
		SizeSmooth:     true,
		BlendMode:      blendMode,
		BlendAdditive:  blendMode == 2,
		AttachedEntity: true,
	}
}
