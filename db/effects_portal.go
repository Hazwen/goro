package db

import (
	"image/color"
	"time"
)

func exit2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1500 * time.Millisecond,
		SFX:      []string{"effect\\ef_teleportation.wav"},
		Components: []EffectComponent{
			robrCylinderBlendComponent("ring_blue", color.RGBA{R: 128, G: 128, B: 255, A: 255}, 1500*time.Millisecond, 0.3, 1, 0.3, 0.3, 35, true, true, true, 2),
			robrCylinderBlendComponent("ring_blue", color.RGBA{R: 128, G: 128, B: 255, A: 255}, 1500*time.Millisecond, 0.3, 1, 0.4, 0.6, 23, true, true, true, 2),
			robrCylinderBlendComponent("ring_blue", color.RGBA{R: 128, G: 128, B: 255, A: 255}, 1500*time.Millisecond, 0.3, 1, 0.5, 0.7, 5, true, true, true, 2),
		},
	}
}

func warpZone2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 7 * time.Second,
		Components: []EffectComponent{
			warpZone2Cylinder(1.15, 1.9),
			warpZone2Cylinder(1.05, 1.8),
			{
				Kind:           EffectComponent3D,
				TextureFile:    "effect/pok1.tga",
				Color:          color.RGBA{R: 230, G: 255, B: 230, A: 255},
				Duration:       time.Second,
				Duplicate:      5,
				DuplicateDelay: 300 * time.Millisecond,
				AlphaMax:       1,
				FadeIn:         true,
				FadeOut:        true,
				PosXStartRand:  1.45,
				PosYStartRand:  1.45,
				PosZEndRand:    1.5,
				PosZEndMiddle:  1.4,
				SizeStart:      effectTableSize(32),
				SizeEnd:        effectTableSize(32),
				BlendMode:      2,
				BlendAdditive:  true,
				AttachedEntity: true,
				Repeat:         true,
			},
		},
	}
}

func warpZone2Cylinder(bottomSize, topSize float64) EffectComponent {
	component := robrCylinderBlendComponent("ring_blue", color.RGBA{R: 128, G: 128, B: 255, A: 255}, 4*time.Second, 0.48, 3, bottomSize, topSize, 1.35, true, false, true, 2)
	component.Duplicate = 4
	component.DuplicateDelay = time.Second
	component.Repeat = true
	return component
}

func entry2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1500 * time.Millisecond,
		SFX:      []string{"effect\\ef_portal.wav"},
		Components: []EffectComponent{
			teleportCylinderComponent(0.3, 0.3, 35),
			teleportCylinderComponent(0.6, 0.8, 25),
			teleportCylinderComponent(0.8, 1.0, 13),
			teleportCylinderComponent(1.0, 1.3, 5),
		},
	}
}

func warpZoneEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 2800 * time.Millisecond,
		Components: []EffectComponent{
			robrCylinderComponent("ring_blue", color.RGBA{R: 128, G: 128, B: 255, A: 255}, 2800*time.Millisecond, 0.3, 3, 2, 3.3, 1.1, true, true, true, true),
			robrCylinderComponent("ring_blue", color.RGBA{R: 128, G: 128, B: 255, A: 255}, 2800*time.Millisecond, 0.3, 3, 1.9, 3.2, 1.1, true, true, true, true),
			{
				Kind:           EffectComponent3D,
				TextureFile:    "effect/pok1.tga",
				Duration:       time.Second,
				Duplicate:      3,
				AlphaMax:       1,
				FadeIn:         true,
				FadeOut:        true,
				PosXStartRand:  3,
				PosYStartRand:  3,
				PosZ:           0,
				PosZEndRand:    2,
				PosZEndMiddle:  2,
				SizeStart:      effectTableSize(100),
				SizeEnd:        effectTableSize(100),
				SizeRand:       effectTableSize(17),
				AttachedEntity: true,
				Color:          color.RGBA{R: 255, G: 255, B: 255, A: 255},
			},
		},
	}
}

func teleportCylinderComponent(bottomSize, topSize, height float64) EffectComponent {
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		Color:            color.RGBA{R: 153, G: 153, B: 255, A: 255},
		TextureName:      "ring_blue",
		Duration:         1500 * time.Millisecond,
		AlphaMax:         0.5,
		Fade:             true,
		Rotate:           true,
		Animation:        5,
		BottomSize:       bottomSize,
		TopSize:          topSize,
		Height:           height,
		AttachedEntity:   true,
		TotalCircleSides: 32,
		CircleSides:      32,
		BlendMode:        2,
		BlendAdditive:    true,
	}
}

func readyPortalCylinderComponent() EffectComponent {
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		Color:            color.RGBA{R: 153, G: 153, B: 255, A: 255},
		TextureName:      "ring_blue",
		Duration:         500 * time.Millisecond,
		Repeat:           true,
		RepeatDelay:      -300 * time.Millisecond,
		AlphaMax:         0.4,
		FadeOut:          true,
		Rotate:           true,
		Animation:        4,
		BottomSize:       2.4,
		TopSize:          3.9,
		Height:           0.1,
		PosZ:             0.1,
		AttachedEntity:   true,
		TotalCircleSides: 32,
		CircleSides:      32,
		BlendMode:        2,
		BlendAdditive:    true,
	}
}

func portalCylinderComponent(bottomSize, topSize, height, posZ float64, textureName string, alphaMax float64) EffectComponent {
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		Color:            color.RGBA{R: 153, G: 153, B: 255, A: 255},
		TextureName:      textureName,
		Duration:         25000 * time.Millisecond,
		AlphaMax:         alphaMax,
		Fade:             true,
		Rotate:           true,
		Animation:        0,
		BottomSize:       bottomSize,
		TopSize:          topSize,
		Height:           height,
		PosZ:             posZ,
		AttachedEntity:   true,
		TotalCircleSides: 32,
		CircleSides:      32,
		BlendMode:        2,
		BlendAdditive:    true,
	}
}
