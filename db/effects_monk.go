package db

import (
	"image/color"
	"strconv"
	"time"
)

func teiHitEffectSpec(asura bool) EffectSpec {
	spec := funcEffectSpec("MonkRays", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/alpha_center.tga"
	spec.Components[0].Duplicate = 80
	if asura {
		spec.Components[0].TextureFile = "effect/lens1.tga"
		spec.Components[0].Duplicate = 96
	}
	return spec
}

func teiHit2EffectSpec() EffectSpec {
	spec := funcEffectSpec("ImpactParticles", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/thunder_center.bmp"
	spec.Components[0].Duplicate = 20
	return spec
}

func tanjiEffectSpec() EffectSpec {
	spec := funcEffectSpec("Tanji", 500*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/blue_ivy.bmp"
	spec.SFX = []string{"effect\\mon_탄지신통.wav"}
	return spec
}

func tripleAttackEffectSpec() EffectSpec {
	spec := funcEffectSpec("TripleAttack", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/cloud11.tga"
	spec.SFX = []string{"effect\\ef_hit2.wav", "effect\\ef_hit4.wav", "effect\\ef_hit2.wav"}
	spec.SFXDelays = []time.Duration{10 * EffectFrameDuration, 20 * EffectFrameDuration, 30 * EffectFrameDuration}
	return spec
}

func hitLineEffectSpec() EffectSpec {
	spec := funcEffectSpec("HitLine", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/white02.bmp"
	return spec
}

func gumgangEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 99999 * EffectFrameDuration,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "Gumgang",
			TextureFiles:   []string{"effect/super1.bmp", "effect/super2.bmp", "effect/super3.bmp", "effect/super4.bmp", "effect/super5.bmp"},
			Color:          color.RGBA{R: 100, G: 100, B: 255, A: 255},
			AttachedEntity: true,
		}},
	}
}

func chimtoEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 16 * EffectFrameDuration,
		SFX:      []string{"effect\\mon_침투경.wav"},
		Components: []EffectComponent{{
			Kind:        EffectComponent3D,
			SpriteFile:  "침투경_", // misc/chimto in the original source's resource aliases
			SpriteDelay: 2 * EffectFrameDuration,
			Duration:    16 * EffectFrameDuration,
			AlphaMax:    200.0 / 255,
			SizeStart:   effectTableSize(250),
			SizeEnd:     effectTableSize(250),
		}},
	}
}

func absorbSpiritsEffectSpec() EffectSpec {
	tint := color.RGBA{R: 77, G: 77, B: 255, A: 255}
	return EffectSpec{
		Duration: 1890 * time.Millisecond,
		SFX:      []string{"effect\\흡기.wav"},
		Components: []EffectComponent{
			robrCylinderBlendComponent("ring_blue", tint, 1500*time.Millisecond, 0.3, 1, 1.1, 1.1, 15, true, true, true, 2),
			robrCylinderBlendComponent("ring_blue", tint, 1500*time.Millisecond, 0.3, 1, 1, 1, 13, true, true, true, 2),
			robrCylinderBlendComponent("ring_blue", tint, 1500*time.Millisecond, 0.3, 1, 1.1, 3, 2, true, true, true, 2),
			absorbSpiritParticle(tint, 1500*time.Millisecond, 0, 10*time.Millisecond, 4, 1.2, 1.2, 0, 0, 0, 1, 8, true),
			absorbSpiritParticle(tint, 1300*time.Millisecond, 400*time.Millisecond, 10*time.Millisecond, 20, 1.5, 1.5, 0, 0, 0, 3, 6, true),
			absorbSpiritParticle(tint, 1100*time.Millisecond, 200*time.Millisecond, 50*time.Millisecond, 10, 1, 1, 1, 0, 6, 0, 0, false),
		},
	}
}

func absorbSpiritParticle(tint color.RGBA, duration, delay, duplicateDelay time.Duration, duplicate int, posXRand, posYRand, posZStartRand, posZStartMiddle, posZEnd, posZEndRand, posZEndMiddle float64, sparkling bool) EffectComponent {
	component := EffectComponent{
		Kind:             EffectComponent3D,
		TextureFile:      "effect/pok3.tga",
		Color:            tint,
		Duration:         duration,
		Delay:            delay,
		Duplicate:        duplicate,
		DuplicateDelay:   duplicateDelay,
		AlphaMax:         0.8,
		FadeIn:           true,
		FadeOut:          true,
		PosXRand:         posXRand,
		PosYRand:         posYRand,
		PosZStartRand:    posZStartRand,
		PosZStartMiddle:  posZStartMiddle,
		PosZEnd:          posZEnd,
		PosZEndRand:      posZEndRand,
		PosZEndMiddle:    posZEndMiddle,
		SizeStart:        effectTableSize(9),
		SizeEnd:          effectTableSize(9),
		SizeRand:         effectTableSize(2),
		BlendMode:        2,
		BlendAdditive:    true,
		AttachedEntity:   true,
		Sparkling:        sparkling,
		RotateWithCamera: false,
	}
	if sparkling {
		component.SparkNumber = 2
	}
	return component
}

func gumgangRingEffectSpec(duration time.Duration, alphaMax, bottomSize, topSize float64, wav string) EffectSpec {
	component := robrCylinderBlendComponent("ring_yellow", color.RGBA{}, duration, alphaMax, 4, bottomSize, topSize, 2, true, true, true, 8)
	component.Duplicate = 4
	component.DuplicateDelay = 100 * time.Millisecond
	spec := EffectSpec{
		Duration:   duration + 300*time.Millisecond,
		Components: []EffectComponent{component},
	}
	if wav != "" {
		spec.SFX = []string{wav}
	}
	return spec
}

func beginAsuraEffectSpec() EffectSpec {
	components := []EffectComponent{
		robrCylinderBlendComponent("ring_white", color.RGBA{}, 800*time.Millisecond, 0, 2, 1, 4.5, -4, true, false, true, 2),
		robrCylinderBlendComponent("ring_white", color.RGBA{}, 800*time.Millisecond, 0, 2, 1, 2.5, -4, true, false, true, 2),
	}
	positions := []float64{-6, -3.6, -1.2, 1.2, 3.6, 6}
	for i, posX := range positions {
		components = append(components,
			asuraGlyphComponent(i+1, posX, time.Duration(i)*100*time.Millisecond, 1200*time.Millisecond, 250, 120, true, false),
			asuraGlyphComponent(i+1, posX, time.Duration(1200+i*100)*time.Millisecond, 400*time.Millisecond, 120, 200, false, true),
		)
	}
	return EffectSpec{
		Duration:   2100 * time.Millisecond,
		Components: components,
	}
}

func beginAsura11EffectSpec() EffectSpec {
	components := []EffectComponent{
		robrCylinderBlendComponent("ring_white", color.RGBA{}, 800*time.Millisecond, 0, 2, 1, 4.5, -4, true, false, true, 2),
		robrCylinderBlendComponent("ring_white", color.RGBA{}, 800*time.Millisecond, 0, 2, 1, 2.5, -4, true, false, true, 2),
	}
	positions := []float64{-8, -4.8, -1.6, 1.6, 4.8, 8}
	for i, posX := range positions {
		components = append(components,
			asuraGlyphComponent(i+11, posX, time.Duration(i)*100*time.Millisecond, 1200*time.Millisecond, 300, 150, true, false),
			asuraGlyphComponent(i+11, posX, time.Duration(1200+i*100)*time.Millisecond, 400*time.Millisecond, 150, 250, false, true),
		)
	}
	return EffectSpec{
		Duration:   2100 * time.Millisecond,
		Components: components,
	}
}

func asuraGlyphComponent(index int, posX float64, delay, duration time.Duration, sizeStart, sizeEnd float64, duplicate, fadeOut bool) EffectComponent {
	component := EffectComponent{
		Kind:           EffectComponent3D,
		TextureFile:    "effect/asura" + strconv.Itoa(index) + ".tga",
		Color:          color.RGBA{R: 26, G: 26, B: 26, A: 255},
		Duration:       duration,
		Delay:          delay,
		AlphaMax:       1,
		AttachedEntity: true,
		FadeIn:         !fadeOut,
		FadeOut:        fadeOut,
		SizeStart:      effectTableSize(sizeStart),
		SizeEnd:        effectTableSize(sizeEnd),
		SizeSmooth:     true,
		PosX:           posX,
		PosZ:           4,
		Overlay:        true,
	}
	if duplicate {
		component.Duplicate = 3
		component.DuplicateDelay = 150 * time.Millisecond
		component.AlphaMaxDelta = -0.25
	}
	return component
}

func steelBodyEffectSpec() EffectSpec {
	spec := gumgangEffectSpec()
	// A packet effect is finite; the Steel Body status keeps its aura alive.
	spec.Duration = 2 * time.Second
	spec.Components[0].Duplicate = 4
	spec.SFX = []string{"effect\\mon_금강불괴.wav"}
	return spec
}
