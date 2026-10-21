package db

import (
	"image/color"
	"time"
)

func agilityEffectSpec(decrease bool) EffectSpec {
	spec := funcEffectSpec("Agility", 100*EffectFrameDuration, true)
	spec.SFX = []string{"effect\\ef_incagility.wav"}
	spec.Components[0].TextureFile = "effect/agi_up.bmp"
	spec.Components[0].TextureFiles = []string{"effect/ac_center2.tga"}
	if decrease {
		spec.SFX = []string{"effect\\ef_decagility.wav"}
		spec.Components[0].TextureFile = "effect/slow.bmp"
	}
	return spec
}

func healCylinderComponent(bottomSize, topSize, height float64) EffectComponent {
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		Color:            color.RGBA{R: 178, G: 255, B: 178, A: 255},
		TextureName:      "ring_white",
		Duration:         1500 * time.Millisecond,
		AlphaMax:         0.2,
		Fade:             true,
		Rotate:           true,
		Animation:        1,
		BottomSize:       bottomSize,
		TopSize:          topSize,
		Height:           height,
		TotalCircleSides: 32,
		CircleSides:      32,
		BlendAdditive:    true,
	}
}

func healOffensiveCylinderComponent(bottomSize, topSize, height float64) EffectComponent {
	component := healCylinderComponent(bottomSize, topSize, height)
	component.Duration = time.Second
	component.Color = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	component.BlendAdditive = true
	return component
}

func healParticleComponent(alpha float64, duration, delay, duplicateDelay time.Duration, duplicate int, posXRand, posYRand, posZStartRand, posZStartMiddle, posZEnd, posZEndRand, posZEndMiddle float64, sparkling bool) EffectComponent {
	component := EffectComponent{
		Kind:            EffectComponent3D,
		Color:           color.RGBA{R: 255, G: 255, B: 255, A: 255},
		TextureFile:     "effect/pok3.tga",
		Duration:        duration,
		Delay:           delay,
		DuplicateDelay:  duplicateDelay,
		AlphaMax:        alpha,
		Sparkling:       sparkling,
		FadeIn:          true,
		FadeOut:         true,
		PosXRand:        posXRand,
		PosYRand:        posYRand,
		PosZStartRand:   posZStartRand,
		PosZStartMiddle: posZStartMiddle,
		PosZEnd:         posZEnd,
		PosZEndRand:     posZEndRand,
		PosZEndMiddle:   posZEndMiddle,
		SizeStart:       9 * EffectPixelRatio,
		SizeEnd:         9 * EffectPixelRatio,
		SizeRand:        2 * EffectPixelRatio,
		Duplicate:       duplicate,
		BlendAdditive:   true,
	}
	if sparkling {
		component.SparkNumber = 2
	}
	return component
}

func incAgilityParticleComponent(alpha float64, delay time.Duration, duplicate int) EffectComponent {
	return EffectComponent{
		Kind:            EffectComponent3D,
		Color:           color.RGBA{R: 255, G: 255, B: 255, A: 255},
		TextureFile:     "effect/ac_center2.tga",
		Duration:        1000 * time.Millisecond,
		Delay:           delay,
		DuplicateDelay:  200 * time.Millisecond,
		AlphaMax:        alpha,
		FadeOut:         true,
		PosXRand:        1.5,
		PosYRand:        1,
		PosZStartRand:   1,
		PosZStartMiddle: 1,
		PosZEndRand:     1,
		PosZEndMiddle:   6,
		SizeStartX:      2.5 * EffectPixelRatio,
		SizeEndX:        2.5 * EffectPixelRatio,
		SizeRandY:       15 * EffectPixelRatio,
		SizeRandYMiddle: 45 * EffectPixelRatio,
		Duplicate:       duplicate,
	}
}

// Acolyte Heal selects these larger variants by heal amount; Arch Bishop also reuses them.
func heal2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1890 * time.Millisecond,
		SFX:      []string{"_heal_effect.wav"},
		Components: []EffectComponent{
			heal2CylinderComponent(1.1, 1.1, 15),
			heal2CylinderComponent(1, 1, 13),
			heal2CylinderComponent(1.1, 3, 2),
			healSparkParticle(0.8, 1500*time.Millisecond, 0, 10*time.Millisecond, 4, 1.2, 1.2, 0, 0, 0, 1, 8, 9, 2, 2, true),
			healSparkParticle(0.8, 1300*time.Millisecond, 400*time.Millisecond, 10*time.Millisecond, 20, 1.5, 1.5, 0, 0, 0, 3, 6, 9, 2, 2, true),
			healSparkParticle(0.8, 1100*time.Millisecond, 200*time.Millisecond, 50*time.Millisecond, 10, 1, 1, 1, 0, 6, 0, 0, 9, 2, 0, false),
		},
	}
}

func heal4EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 2 * time.Second,
		SFX:      []string{"_heal_effect.wav"},
		Components: []EffectComponent{
			heal2CylinderComponent(1.1, 1.1, 18),
			heal2CylinderComponent(1, 1, 15),
			heal2CylinderComponent(1.1, 3, 3),
			healSparkParticle(0.8, 1500*time.Millisecond, 0, 10*time.Millisecond, 7, 1.2, 1.2, 0, 0, 0, 1, 8, 9, 2, 3, true),
			healSparkParticle(0.8, 1300*time.Millisecond, 400*time.Millisecond, 10*time.Millisecond, 25, 1.5, 1.5, 0, 0, 0, 3, 6, 10, 5, 3, true),
			healSparkParticle(0.8, 1100*time.Millisecond, 200*time.Millisecond, 50*time.Millisecond, 15, 1, 1, 1, 0, 6, 0, 0, 11, 2, 0, false),
		},
	}
}

func heal2CylinderComponent(bottomSize, topSize, height float64) EffectComponent {
	return robrCylinderBlendComponent("ring_white", color.RGBA{R: 178, G: 255, B: 178, A: 255}, 1500*time.Millisecond, 0.3, 1, bottomSize, topSize, height, true, true, true, 2)
}

func healSparkParticle(alpha float64, duration, delay, duplicateDelay time.Duration, duplicate int, posXRand, posYRand, posZStartRand, posZStartMiddle, posZEnd, posZEndRand, posZEndMiddle, size, sizeRand float64, sparkNumber int, sparkling bool) EffectComponent {
	component := EffectComponent{
		Kind:            EffectComponent3D,
		TextureFile:     "effect/pok3.tga",
		Color:           color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Duration:        duration,
		Delay:           delay,
		Duplicate:       duplicate,
		DuplicateDelay:  duplicateDelay,
		AlphaMax:        alpha,
		FadeIn:          true,
		FadeOut:         true,
		PosXRand:        posXRand,
		PosYRand:        posYRand,
		PosZStartRand:   posZStartRand,
		PosZStartMiddle: posZStartMiddle,
		PosZEnd:         posZEnd,
		PosZEndRand:     posZEndRand,
		PosZEndMiddle:   posZEndMiddle,
		SizeStart:       effectTableSize(size),
		SizeEnd:         effectTableSize(size),
		SizeRand:        effectTableSize(sizeRand),
		BlendMode:       2,
		BlendAdditive:   true,
		AttachedEntity:  true,
		Sparkling:       sparkling,
		SparkNumber:     sparkNumber,
	}
	return component
}
