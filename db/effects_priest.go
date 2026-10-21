package db

import (
	"time"
)

func turnUndeadEffectSpec() EffectSpec {
	spec := funcEffectSpec("TurnUndead", 250*EffectFrameDuration, true)
	spec.SFX = []string{"effect\\ef_bash.wav"}
	spec.SFXDelays = []time.Duration{37 * EffectFrameDuration}
	spec.Components[0].TextureFile = "effect/alpha_down.tga"
	spec.Components[0].TextureFiles = []string{"effect/alpha_center.tga", "effect/ring_liner.tga"}
	return spec
}

func reviveEffectSpec() EffectSpec {
	spec := funcEffectSpec("Revive", 250*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/ring_yellow.tga"
	return spec
}
