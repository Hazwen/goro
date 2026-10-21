package db

import (
	"time"
)

const EffectFrameDuration = time.Second / 60

// EffectHitRing holds the Prim3DCylinder motion in cells per original-client tick.
type EffectHitRing struct {
	Speed, HeightSpeed, HeightAcceleration float64
	InitialDistance                        float64
	FadeOutFrame                           int
	RandomImpactOffset                     bool
}

// EffectHitParticles uses the original random speed range, in hundredths of
// a native unit per tick. One cell is five native units.
type EffectHitParticles struct {
	SpeedMin, SpeedRange int
}

func hitRingComponent(texture string, frames int, inner, outer, height, speed float64, fadeOutFrame int) EffectComponent {
	return EffectComponent{
		Kind: EffectComponentFUNC, FuncName: "HitRing", TextureName: texture,
		Duration: time.Duration(frames) * EffectFrameDuration, FrameDelay: EffectFrameDuration,
		AlphaMax: 254.0 / 255, BottomSize: inner / 5, TopSize: outer / 5,
		Height: height / 5, PosZ: 2, AngleX: 90, CircleSides: 10, AttachedEntity: true,
		HitRing: &EffectHitRing{Speed: speed / 5, FadeOutFrame: fadeOutFrame},
	}
}

// CRagEffect::Hit3/Hit4: growing cylinders followed by three-segment SPR trails.
func legacyHitEffectSpec(arrow bool) EffectSpec {
	inner, count, speedMin, speedRange := 1.5, 8, 80, 120
	heightAccel, sound := 0.2, "effect\\ef_hit3.wav"
	if arrow {
		inner, count, speedMin, speedRange = 0.5, 5, 60, 90
		heightAccel, sound = 0.15, "effect\\ef_hit4.wav"
	}
	ring := hitRingComponent("lens2", 15, inner, 4, 0, 0.7, 10)
	ring.HitRing.HeightSpeed = 0.25 / 5
	ring.HitRing.HeightAcceleration = heightAccel / 5
	components := []EffectComponent{}
	if !arrow {
		core := hitRingComponent("lens2", 15, 1.5, 1.5, 0, 0.7, 10)
		core.HitRing.HeightSpeed = 0.5 / 5
		core.HitRing.HeightAcceleration = 0.2 / 5
		components = append(components, core)
	}
	components = append(components, ring, EffectComponent{
		Kind: EffectComponentFUNC, FuncName: "HitParticles", SpriteFile: "particle1",
		Duration: 30 * EffectFrameDuration, FrameDelay: EffectFrameDuration,
		Duplicate: count, PosZ: 2, AttachedEntity: true,
		HitParticles: &EffectHitParticles{SpeedMin: speedMin, SpeedRange: speedRange},
	})
	return EffectSpec{Duration: 30 * EffectFrameDuration, SFX: []string{sound}, Components: components}
}

func bash3DEffectSpec(funcName, wav string, duration, delay time.Duration, duplicate int) EffectSpec {
	return EffectSpec{
		Duration: duration,
		SFX:      []string{wav},
		Components: []EffectComponent{
			{
				Kind:           EffectComponentFUNC,
				FuncName:       funcName,
				AttachedEntity: true,
			},
			bash3DCylinderComponent(4.5, delay, duplicate),
			bash3DCylinderComponent(7.2, delay, duplicate),
		},
	}
}

func bash3DCylinderComponent(topSize float64, delay time.Duration, duplicate int) EffectComponent {
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		TextureName:      "alpha_center",
		Duration:         175 * time.Millisecond,
		Delay:            delay,
		Duplicate:        duplicate,
		AlphaMax:         0.6,
		Fade:             true,
		AngleX:           -90,
		AngleZRandom:     360,
		FixedPerspective: true,
		PosZ:             1.5,
		Height:           0,
		BottomSize:       0.01,
		TopSize:          topSize,
		Animation:        2,
		AttachedEntity:   true,
		TotalCircleSides: 30,
		CircleSides:      1,
	}
}

func bash3D6EffectSpec() EffectSpec {
	spec := bash3DEffectSpec("Bash3D6", "effect\\bash3d.wav", 500*time.Millisecond, 200*time.Millisecond, 5)
	tint := effectTableColor(0.3, 0.5, 1)
	for i := 1; i < len(spec.Components); i++ {
		spec.Components[i].Color = tint
	}
	return spec
}

func teiHit1TEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 350 * time.Millisecond,
		SFX:      []string{"effect\\mon_아수라 패황권.wav"},
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			TextureFile:      "effect/lens1.tga",
			Color:            effectTableColor(0.1, 0.1, 1),
			Duration:         250 * time.Millisecond,
			Delay:            100 * time.Millisecond,
			Duplicate:        24,
			DuplicateDelay:   0,
			AlphaMax:         0.8,
			BlendMode:        2,
			BlendAdditive:    true,
			FadeIn:           true,
			FadeOut:          true,
			AttachedEntity:   true,
			PosXEndRand:      40,
			PosYEndRand:      40,
			SizeStartX:       effectTableSize(10),
			SizeStartY:       effectTableSize(150),
			SizeEndX:         effectTableSize(10),
			SizeEndY:         effectTableSize(150),
			Overlay:          true,
			RotateToTarget:   true,
			RotateWithCamera: true,
		}},
	}
}

// The BASH3D ray burst is shared by Lord Knight attacks and Gunslinger Dust.
func lordKnightBurstEffectSpec(sound string, delay time.Duration) EffectSpec {
	return EffectSpec{
		Duration: 51 * EffectFrameDuration,
		SFX:      []string{sound}, SFXDelays: []time.Duration{delay},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "Sacrifice", TextureFile: "effect/alpha_center.tga",
		}},
	}
}
