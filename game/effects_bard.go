package game

import (
	"time"

	"github.com/kivutar/goro/db"
)

func bardSkillActionSpec(skillID uint16) (skillActionSpec, bool) {
	if skillID == db.SkillBaFrostjoke || skillID == db.SkillBaPangvoice {
		spec := newSkillActionSpec(skillActorActionSkill, true, &idleSkillActionSpec)
		spec.frame, spec.hasFrame, spec.length = 4, true, 2
		spec.speed, spec.duration = 10*db.EffectFrameDuration, 1800*time.Millisecond
		return spec, true
	}
	if skillID == db.SkillBaDissonance || skillID >= db.SkillBaWhistle && skillID <= db.SkillBaAppleidun {
		spec := danceSkillActionSpec()
		spec.pingPong = false // ST_SING loops 1,2,3; ST_DANCE plays 1,2,3,2.
		return spec, true
	}
	return skillActionSpec{}, false
}

func isBardPerformance(skillID uint16) bool {
	return skillID >= db.SkillBDLullaby && skillID <= db.SkillBDSiegfried ||
		skillID == db.SkillBaDissonance || skillID >= db.SkillBaWhistle && skillID <= db.SkillBaAppleidun
}
