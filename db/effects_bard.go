package db

func pangVoiceEffectSpec() EffectSpec {
	spec := directedSpriteEffectSpec("fvoice")
	spec.SFX = []string{"amon_ra_die01.wav"}
	return spec
}
