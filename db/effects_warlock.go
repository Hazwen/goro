package db

import (
	"image/color"
	"time"
)

func earthWallEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:    1 * time.Second,
		CameraShake: 200 * time.Millisecond,
		SFX:         []string{"effect\\wizard_earthspike.wav"},
		Components: []EffectComponent{
			quadHornEffectComponent(
				"effect/stone.bmp",
				time.Second,
				[2]float64{0.75, 1.2},
				[2]float64{0.2, 0.2},
				[2]float64{0.2, 0.2},
				-0.1,
				[2]float64{0.4, 0.9},
				[2]float64{1, 360},
				[2]float64{-8, 8},
				3,
				250*time.Millisecond,
				true,
				1,
			),
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "CameraQuake",
				AttachedEntity: true,
			},
		},
	}
}

func bottomBasilica2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 20 * time.Second,
		SFX:      []string{"effect\\wl_whiteimprison.wav"},
		Components: []EffectComponent{
			basilicaCylinderComponent(2.2, 3.0, 65.0/255.0, 0),
			basilicaCylinderComponent(2.25, 3.1, 65.0/255.0, 10),
			basilicaCylinderComponent(2.3, 3.0, 15.0/255.0, 0),
			basilicaCylinderComponent(2.3, 3.0, 15.0/255.0, 53.2),
		},
	}
}

func basilicaCylinderComponent(size, height, alpha, angleY float64) EffectComponent {
	component := robrCylinderBlendComponent("alpha_down", color.RGBA{}, 20*time.Second, alpha, 0, size, size, height, true, false, false, 2)
	component.TotalCircleSides = 4
	component.CircleSides = 4
	component.RotateWithCamera = true
	component.AngleY = angleY
	return component
}
