package db

import (
	"time"
)

func mildWindEffectSpec(texture string) EffectSpec {
	return EffectSpec{Duration: 300 * EffectFrameDuration, Components: []EffectComponent{{
		Kind: EffectComponentFUNC, FuncName: "AsuraGlyphs", TextureFiles: []string{texture},
		AttachedEntity: true,
	}}}
}

func taekwonStormEffectSpec(kick bool) EffectSpec {
	spec := EffectSpec{Duration: 100 * EffectFrameDuration, Components: []EffectComponent{{
		Kind: EffectComponentFUNC, FuncName: "TaekwonStorm",
		TextureFiles:   []string{"effect/storm2.tga", "effect/cloud11.tga"},
		AttachedEntity: true,
	}}}
	if kick {
		spec.SFX = []string{"effect\\T_회오리차기.wav"}
		spec.SFXDelays = []time.Duration{5 * EffectFrameDuration}
	}
	return spec
}

func taekwonStopEffectSpec() EffectSpec {
	return EffectSpec{Duration: 100 * EffectFrameDuration,
		SFX: []string{"effect\\T_효과음1.wav"}, Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "TaekwonStop", TextureFile: "effect/magic_blue.tga", AttachedEntity: true,
		}}}
}

func taekwonDashEffectSpec() EffectSpec {
	return EffectSpec{Duration: 300 * EffectFrameDuration, Components: []EffectComponent{{
		Kind: EffectComponentFUNC, FuncName: "TaekwonDash", TextureFile: "effect/line3.tga",
	}}}
}

func taekwonJumpEffectSpec() EffectSpec {
	return EffectSpec{Duration: 300 * EffectFrameDuration, Components: []EffectComponent{{
		Kind: EffectComponentFUNC, FuncName: "TaekwonJump",
		TextureFiles: []string{"effect/whitelight.tga", "effect/peong1.tga", "effect/peong2.tga", "effect/peong3.tga"},
	}}}
}

func taekwonFootEffectSpec(texture string) EffectSpec {
	return EffectSpec{Duration: 340 * EffectFrameDuration, Components: []EffectComponent{{
		Kind: EffectComponentFUNC, FuncName: "TaekwonFoot", TextureFile: texture,
	}}}
}
