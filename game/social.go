package game

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/social"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func (m *WorldMode) handleChatMessage(ctx client.Context, chat network.ChatMessage, now time.Time) {
	if chat.Announcement {
		text := social.FormatConsoleMessage(ctx.Resources, chat)
		if text == "" {
			return
		}
		m.ui.announcement.Show(text, social.ROClientColor(chat.Color), gameui.AnnouncementStyle{
			Y:        int(chat.FontY),
			FontSize: int(chat.FontSize),
			Bold:     chat.FontType >= 700,
		}, now)
		social.AddConsoleMessage(&m.ui.console, ctx.Resources, chat)
		return
	}
	if m.ui.social.ChatRoom.IsOpen() && !chat.HasColor {
		m.social().AddChatRoomMessage(ctx, chat)
		return
	}
	m.applySpeechBubble(ctx, chat, now)
	social.AddConsoleMessage(&m.ui.console, ctx.Resources, chat)
}

func (m *WorldMode) openPlayerContextFromInput(ctx client.Context, now time.Time) bool {
	if ctx.Input == nil || !ctx.Input.MouseJustPressed(input.MouseButtonRight) || m.mapPointerBlocked(ctx) {
		return false
	}
	screenW, screenH := ctx.ScreenSize()
	projection := m.sceneProjection(ctx, screenW, screenH, now)
	actor, ok := clickedPlayerTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths)
	if !ok {
		return false
	}
	canManageGuildRelations := social.GuildCanManageRelations(ctx.Session, actor.GuildID)
	m.ui.social.PlayerContext.Open(ctx, ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, gameui.PlayerContextOptions{
		CanAddFriend:       !social.FriendNameInSession(ctx.Session, actor.Name),
		CanInviteParty:     social.PartyCanInvite(ctx.Session),
		CanInviteGuild:     social.GuildCanInvitePlayer(ctx.Session, actor.GuildID),
		CanAdopt:           social.CanRequestAdoption(ctx.Session, actor),
		CanRequestAlliance: canManageGuildRelations,
		CanDeclareHostile:  canManageGuildRelations,
	})
	return true
}

const siegeGuildEmblemSize = 24

func (m *WorldMode) drawSiegeGuildEmblems(screen *render.Frame, ctx client.Context, projection scene.Projection, now time.Time, entries []scene.ActorDrawEntry) {
	if screen == nil || ctx.World == nil || !ctx.World.MapProperty.IsSiege() {
		return
	}
	for _, entry := range entries {
		if !siegeActorShowsGuildEmblem(entry) {
			continue
		}
		emblem := m.social().ActorGuildEmblem(ctx, entry.Actor, entry.IsPlayer)
		if emblem == nil {
			continue
		}
		bounds := emblem.Bounds()
		if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
			continue
		}
		centerX, topY := m.siegeGuildEmblemAnchor(ctx, projection, now, entry)
		x, y := siegeGuildEmblemPosition(centerX, topY, siegeGuildEmblemSize)
		x, y = render.SnapScreenPoint(screen, x, y)
		var opts render.DrawImageOptions
		opts.Filter = render.FilterLinear
		opts.GeoM.Scale(float64(siegeGuildEmblemSize)/float64(bounds.Dx()), float64(siegeGuildEmblemSize)/float64(bounds.Dy()))
		opts.GeoM.Translate(x, y)
		screen.DrawImage(emblem, &opts)
	}
}

func siegeActorShowsGuildEmblem(entry scene.ActorDrawEntry) bool {
	return !scene.ActorHasStealth(entry.Actor) && entry.Actor.GuildID != 0 && entry.Actor.EmblemVersion != 0
}

func (m *WorldMode) siegeGuildEmblemAnchor(ctx client.Context, projection scene.Projection, now time.Time, entry scene.ActorDrawEntry) (float64, float64) {
	centerX := entry.ScreenX
	topY := actorSpriteTopY(entry.ScreenY, entry.Scale)
	if entry.IsPlayer || !m.nonPCActorHasGR2Model(ctx, entry.Actor) {
		return centerX, topY
	}

	view := m.nonPCGR2ModelView(ctx, entry.Actor)
	if view == nil || view.geometry == nil {
		return centerX, topY
	}
	scale := gr2ModelWorldScale * m.actorBodySizeMultiplier(entry.Actor.ID, now)
	top := scene.Point3{
		X: float64(view.geometry.Center[0]),
		Y: float64(view.geometry.Center[1]),
		Z: float64(view.geometry.Center[2] + view.geometry.Size[2]/2),
	}
	matrix := gr2ActorModelMatrix(entry.WorldX, entry.WorldY, entry.WorldZ, entry.Actor.Dir, scale)
	worldTop := scene.Mat4TransformPoint(matrix, top)
	projected := projection.Project(worldTop.X, worldTop.Z, worldTop.Y)
	if !scene.IsFinite(float64(projected.X)) || !scene.IsFinite(float64(projected.Y)) {
		return centerX, topY
	}
	return float64(projected.X), float64(projected.Y)
}

func siegeGuildEmblemPosition(centerX, topY float64, size int) (float64, float64) {
	if size <= 0 {
		size = siegeGuildEmblemSize
	}
	x := centerX - float64(size)/2
	y := topY - float64(size) - 4
	return x, y
}

func partyMemberLifeForDisplay(ctx client.Context, actor worldstate.Actor) (actorLife, bool) {
	if ctx.Session == nil || actor.ID == 0 || !actor.HasObjectType || actor.ObjectType != scene.ActorObjectTypePC {
		return actorLife{}, false
	}
	for _, member := range ctx.Session.Party.Members {
		if member.AccountID != actor.ID || member.MaxHP <= 0 {
			continue
		}
		return actorLife{
			hp:     clampGameInt(member.HP, 0, member.MaxHP),
			maxHP:  member.MaxHP,
			player: true,
		}, true
	}
	return actorLife{}, false
}

func (m *WorldMode) social() *social.System {
	m.socialSystem.Bind(social.Bindings{Windows: &m.ui.social, Console: &m.ui.console, Minimap: &m.ui.minimap})
	return &m.socialSystem
}
