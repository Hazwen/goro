package game

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func TestSkillInfoUpdatePreservesExistingTargetMetadata(t *testing.T) {
	sessionState := &session.Session{
		Skills: session.Skills{
			List: []session.Skill{
				{
					ID:       db.SkillWECallpartner,
					Type:     skills.SkillTargetFriend,
					Level:    1,
					MaxLevel: 1,
					SPCost:   1,
					Range:    1,
					Name:     "Romantic Rendeavous!!",
				},
			},
		},
	}
	ctx := client.Context{Session: sessionState}

	applySkillInfoUpdate(ctx, network.SkillInfoUpdate{Skill: network.SkillInfo{
		ID:     db.SkillWECallpartner,
		Level:  1,
		SPCost: 2,
		Range:  1,
	}})

	got := sessionState.Skills.List[0]
	if got.Type != skills.SkillTargetFriend {
		t.Fatalf("skill type = %d, want existing target type %d", got.Type, skills.SkillTargetFriend)
	}
	if got.Name != "Romantic Rendeavous!!" {
		t.Fatalf("skill name = %q, want existing name", got.Name)
	}
	if got.MaxLevel != 1 {
		t.Fatalf("skill max level = %d, want existing max level", got.MaxLevel)
	}
	if got.SPCost != 2 {
		t.Fatalf("skill sp cost = %d, want updated cost", got.SPCost)
	}
}
