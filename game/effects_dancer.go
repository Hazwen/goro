package game

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
)

func isDancerPerformance(skillID uint16) bool {
	return skillID == db.SkillDCUglydance || skillID >= db.SkillDCHumming && skillID <= db.SkillDCServiceforyou
}

func dancerSkillActionSpec(skillID uint16) (skillActionSpec, bool) {
	if skillID == db.SkillDCScream {
		return skillActionSpec{defined: true, action: skillActorActionSkill, hasFrame: true, frame: 0, duration: 1200 * time.Millisecond}, true
	}
	return skillActionSpec{}, false
}

func (m *WorldMode) applyWinkCharm(ctx client.Context, notify network.SkillNoDamageNotify) {
	if notify.SkillID != db.SkillDCWinkcharm {
		return
	}
	if _, ok, _ := actorForCombatID(ctx, notify.SourceID); !ok {
		return
	}
	if _, ok, _ := actorForCombatID(ctx, notify.TargetID); !ok {
		return
	}
	m.applyEmotionNotify(ctx, network.EmotionNotify{GID: notify.SourceID, Type: 30}) // ET_CHUP
}
