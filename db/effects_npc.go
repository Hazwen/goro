package db

import (
	"image/color"
	"time"
)

func suiExplosionEffectSpec() EffectSpec {
	return EffectSpec{
		CameraShake: 200 * time.Millisecond,
		SFX:         []string{"effect\\ef_hit2.wav"},
		Components: []EffectComponent{
			{
				Kind:           EffectComponentSTR,
				STRFile:        "sui_explosion",
				AttachedEntity: true,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "CameraQuake",
				AttachedEntity: true,
			},
		},
	}
}

func drainEffectSpec(tint color.RGBA, withBodyColor bool) EffectSpec {
	spec := EffectSpec{
		Duration: 900 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			SpriteFile:       "data/sprite/이팩트/particle1",
			SpriteRepeat:     true,
			Duration:         600 * time.Millisecond,
			Duplicate:        5,
			DuplicateDelay:   0,
			ToSrc:            true,
			RotateToTarget:   true,
			RotateWithCamera: true,
			AlphaMax:         1,
			Color:            tint,
			SizeStart:        effectTableSize(150),
			SizeEnd:          effectTableSize(180),
			PosZ:             5,
			Arc:              3,
			Retreat:          3,
		}},
	}
	if withBodyColor {
		spec.Components = append(spec.Components,
			EffectComponent{Kind: EffectComponentFUNC, FuncName: "EnergyDrainOwnerBodyColor", AttachedEntity: true, Delay: 500 * time.Millisecond, Duration: 400 * time.Millisecond},
			EffectComponent{Kind: EffectComponentFUNC, FuncName: "EnergyDrainTargetBodyColor", AttachedEntity: true, Duration: 400 * time.Millisecond},
		)
	}
	return spec
}

func npcEarthquakeEffectSpec() EffectSpec {
	return EffectSpec{
		CameraShake: 650 * time.Millisecond,
		SFX:         []string{"effect\\earth_quake.wav"},
		Components: []EffectComponent{
			{
				Kind:           EffectComponentSPR,
				SpriteFile:     "어스퀘이크",
				AttachedEntity: true,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "CameraQuake",
				AttachedEntity: true,
				Duplicate:      3,
				DuplicateDelay: 35 * time.Millisecond,
			},
		},
	}
}

func dragonFearEffectSpec() EffectSpec {
	return EffectSpec{
		CameraShake: 650 * time.Millisecond,
		SFX:         []string{"effect\\dragonfear.wav"},
		Components: []EffectComponent{
			{
				Kind:           EffectComponentSTR,
				STRFile:        "dragon_h",
				AttachedEntity: true,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "CameraQuake",
				AttachedEntity: true,
			},
		},
	}
}

func evilLandEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1500 * time.Millisecond,
		Components: []EffectComponent{
			{
				Kind:      EffectComponentFUNC,
				FuncName:  "FlatColorTile",
				Color:     color.RGBA{R: 160, G: 160, B: 160, A: 51},
				SizeStart: 1,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "GroundTexture",
				TextureFile:    "effect/curse.bmp",
				Duration:       1500 * time.Millisecond,
				SizeStart:      1,
				SizeEnd:        1,
				AlphaMax:       0.7,
				PosZ:           0.4,
				BlendAdditive:  true,
				AttachedEntity: false,
			},
		},
	}
}

func darkSoulStrikeEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 450 * time.Millisecond,
		SFX:      []string{"effect\\ef_soulstrike.wav"},
		Components: []EffectComponent{
			{
				Kind:            EffectComponent3D,
				TextureFile:     "effect/pok3.tga",
				Color:           color.RGBA{R: 255, G: 255, B: 255, A: 255},
				Duration:        200 * time.Millisecond,
				Delay:           250 * time.Millisecond,
				DuplicateDelay:  150 * time.Millisecond,
				AlphaMax:        1,
				FadeIn:          true,
				FadeOut:         true,
				ToSrc:           true,
				PosZEnd:         1,
				PosZSmooth:      true,
				PosZStartRand:   5,
				PosZStartMiddle: 6,
				SizeStart:       effectTableSize(50),
				SizeEnd:         effectTableSize(50),
				AttachedEntity:  true,
			},
			{
				Kind:           EffectComponent3D,
				SpriteFile:     "data/sprite/이팩트/particle5",
				SpriteRepeat:   true,
				Duration:       250 * time.Millisecond,
				Duplicate:      5,
				DuplicateDelay: 20 * time.Millisecond,
				ToSrc:          true,
				RotateToTarget: true,
				SizeStart:      effectTableSize(100),
				SizeEnd:        effectTableSize(500),
				PosZ:           3,
				Arc:            4,
				Retreat:        4,
			},
		},
	}
}

func darkJupitelHitEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 300 * time.Millisecond,
		Components: []EffectComponent{
			{
				Kind:           EffectComponent3D,
				TextureFile:    "effect/pokjuk_d.bmp",
				Duration:       100 * time.Millisecond,
				SizeStart:      0,
				SizeEnd:        effectTableSize(25),
				BlendMode:      2,
				BlendAdditive:  true,
				RotateToTarget: true,
				FadeOut:        true,
				Overlay:        true,
				AttachedEntity: true,
			},
			{
				Kind: EffectComponent3D,
				TextureFiles: []string{
					"effect/twirl_soft.bmp",
					"effect/thunder_ball_b.bmp",
					"effect/twirl_soft.bmp",
					"effect/thunder_ball_c.bmp",
					"effect/twirl_soft.bmp",
				},
				FrameDelay:     10 * time.Millisecond,
				Duration:       300 * time.Millisecond,
				SizeStart:      effectTableSize(75),
				SizeEnd:        effectTableSize(75),
				BlendMode:      2,
				BlendAdditive:  true,
				Overlay:        true,
				AttachedEntity: true,
			},
		},
	}
}
