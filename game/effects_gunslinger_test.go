package game

import (
	"testing"

	"github.com/kivutar/goro/db"
)

func TestGunslingerCoinFlipEffectMatchesRobrowser(t *testing.T) {
	expectEffectIDs(t, "GS_GLITTERING", skillEffectIDs(db.SkillGSGlittering), effectGunslingerCoinSound)
	spec, ok := worldEffectSpecForID(effectGunslingerCoinSound)
	if !ok {
		t.Fatal("gunslinger_coin effect spec is missing")
	}
	if len(spec.components) != 0 || len(spec.sfx) != 1 || spec.sfx[0] != "effect\\플립.wav" {
		t.Fatalf("gunslinger_coin spec = %+v, want sound-only effect\\플립.wav", spec)
	}
}

func TestGunslingerCombatActionsMatchRobrowser(t *testing.T) {
	for _, skillID := range []uint16{
		db.SkillGSTripleaction,
		db.SkillGSBullseye,
		db.SkillGSTracking,
		db.SkillGSDisarm,
		db.SkillGSPiercingshot,
		db.SkillGSRapidshower,
		db.SkillGSDesperado,
		db.SkillGSDust,
		db.SkillGSFullbuster,
		db.SkillGSSpreadattack,
		db.SkillGSGrounddrift,
	} {
		action := skillAction(skillID)
		if !action.defined || action.action != skillActorActionAttack {
			t.Fatalf("skill %d action = %+v, want ATTACK", skillID, action)
		}
	}
}
