package scene

import (
	"github.com/kivutar/goro/client"
)

func LocalSkillTarget(ctx client.Context) uint32 {
	if ctx.Session == nil {
		return 0
	}
	if ctx.Session.AccountID != 0 {
		return ctx.Session.AccountID
	}
	return ctx.Session.CharID
}
