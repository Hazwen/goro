package game

import (
	"testing"

	"github.com/kivutar/goro/db"
)

func TestNinjaSkillEffectsMatchRobrowser(t *testing.T) {
	expectEffectIDs(t, "NJ_SYURIKEN", skillBeforeHitEffectIDs(db.SkillNJSyuriken), effectThrowItem7)
	expectEffectIDs(t, "NJ_KUNAI", skillBeforeHitEffectIDs(db.SkillNJKunai), effectThrowItem8)
	expectEffectIDs(t, "NJ_HUUMA", skillBeforeHitEffectIDs(db.SkillNJHuuma), effectThrowItem9)
	expectEffectIDs(t, "NJ_ZENYNAGE", skillBeforeHitEffectIDs(db.SkillNJZenynage), effectThrowItem10)
	expectEffectIDs(t, "NJ_TATAMIGAESHI", skillGroundEffectIDs(db.SkillNJTatamigaeshi), effectTatami)
	expectEffectIDs(t, "NJ_KAENSIN", skillGroundEffectIDs(db.SkillNJKaensin), effectKaen)
	expectEffectIDs(t, "NJ_SUITON", skillGroundEffectIDs(db.SkillNJSuiton), 620)
	expectEffectIDs(t, "NJ_ISSEN", skillEffectIDs(db.SkillNJIssen), effectIssen)
}

func TestNinjaTatamiActionMatchesRobrowser(t *testing.T) {
	action := skillAction(db.SkillNJTatamigaeshi)
	if !action.defined || action.action != skillActorActionPickup || !action.hasFrame || action.frame != 1 || action.play || action.repeat || action.next != nil {
		t.Fatalf("Tatami action = %+v, want held PICKUP frame 1", action)
	}
}

func TestNinjaMirrorImageStatusUsesRobrowserOpt3State(t *testing.T) {
	if got := db.StatusOpt3State[db.StatusNjBunsinjyutsu]; got != db.Opt3Bunsin {
		t.Fatalf("Mirror Image Opt3 = 0x%08X, want 0x%08X", got, db.Opt3Bunsin)
	}
	r, g, b, a := actorOpt3StateTint(db.Opt3Bunsin)
	if r != 0.5 || g != 0.5 || b != 0.85 || a != 1 {
		t.Fatalf("Mirror Image tint = %.2f,%.2f,%.2f,%.2f", r, g, b, a)
	}
}
