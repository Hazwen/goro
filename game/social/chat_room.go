package social

import (
	"fmt"
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func (m *System) UpdateChatRoomWindows(ctx client.Context) bool {
	createConsumed := m.bindings.Windows.ChatRoomCreate.Update(ctx)
	if action := m.bindings.Windows.ChatRoomCreate.PopAction(); action.Title != "" {
		m.createChatRoomFromWindow(ctx, action)
		return true
	}
	if createConsumed {
		return true
	}
	roomConsumed := m.bindings.Windows.ChatRoom.Update(ctx)
	if action := m.bindings.Windows.ChatRoom.PopAction(); action.Leave || action.Message != "" {
		m.HandleChatRoomAction(ctx, action)
		return true
	}
	return roomConsumed
}

func (m *System) createChatRoomFromWindow(ctx client.Context, action gameui.ChatRoomCreateWindowAction) {
	room := network.ChatRoomCreate{
		Title:    strings.TrimSpace(action.Title),
		Password: strings.TrimSpace(action.Password),
		Limit:    action.Limit,
		Public:   action.Public,
	}
	if room.Title == "" {
		return
	}
	if ctx.Network == nil {
		m.bindings.Console.AddErrorMessage("Create chat room failed: not connected.")
		return
	}
	if err := ctx.Network.SendCreateChatRoom(room); err != nil {
		m.bindings.Console.AddErrorMessage("Create chat room failed.")
		glog.Warnf("chat room create failed title=%q limit=%d public=%t: %v", room.Title, room.Limit, room.Public, err)
		return
	}
	m.PendingChatRoom = room
}

func (m *System) HandleChatRoomCreateAck(ctx client.Context, ack network.ChatRoomCreateAck) {
	switch ack.Result {
	case 0:
		room := m.PendingChatRoom
		if room.Title == "" {
			room = network.ChatRoomCreate{Title: "Chat Room", Limit: 20, Public: true}
		}
		m.PendingChatRoom = network.ChatRoomCreate{}
		member := scene.SelectedCharacterName(ctx.Session)
		m.bindings.Windows.ChatRoom.Open(ctx, room.Title, room.Limit, room.Public, []string{member})
		m.bindings.Console.AddBlueMessage("%s", chatRoomCreateAckMessage(ctx, ack))
	case 1, 2:
		m.PendingChatRoom = network.ChatRoomCreate{}
		m.bindings.Console.AddErrorMessage("%s", chatRoomCreateAckMessage(ctx, ack))
	default:
		m.PendingChatRoom = network.ChatRoomCreate{}
		m.bindings.Console.AddErrorMessage("Create chat room failed.")
	}
}

func (m *System) ApplyChatRoomBoard(ctx client.Context, board network.ChatRoomBoard) {
	ApplyChatRoomBoardToWorld(ctx, board)
	glog.Debugf("chat room board actor=%d room=%d title=%q count=%d limit=%d public=%t", board.OwnerID, board.RoomID, board.Title, board.Count, board.Limit, board.Public)
}

func ApplyChatRoomBoardToWorld(ctx client.Context, board network.ChatRoomBoard) {
	if ctx.World == nil {
		return
	}
	if scene.IsLocalActor(ctx, board.OwnerID) {
		ctx.World.Player.ChatRoom = true
		ctx.World.Player.ChatRoomID = board.RoomID
		ctx.World.Player.ChatRoomTitle = board.Title
		ctx.World.Player.ChatRoomCount = board.Count
		ctx.World.Player.ChatRoomLimit = board.Limit
		ctx.World.Player.ChatRoomPublic = board.Public
		return
	}
	actor, ok := ctx.World.Actors[board.OwnerID]
	if !ok {
		actor = worldstate.Actor{ID: board.OwnerID}
	}
	actor.ChatRoom = true
	actor.ChatRoomID = board.RoomID
	actor.ChatRoomTitle = board.Title
	actor.ChatRoomCount = board.Count
	actor.ChatRoomLimit = board.Limit
	actor.ChatRoomPublic = board.Public
	ctx.World.Actors[board.OwnerID] = actor
}

func (m *System) ApplyChatRoomDestroy(ctx client.Context, destroy network.ChatRoomDestroy) {
	ApplyChatRoomDestroyToWorld(ctx, destroy)
	glog.Debugf("chat room board removed room=%d", destroy.RoomID)
}

func ApplyChatRoomDestroyToWorld(ctx client.Context, destroy network.ChatRoomDestroy) {
	if ctx.World == nil {
		return
	}
	if ctx.World.Player.ChatRoomID == destroy.RoomID {
		clearActorChatRoom(&ctx.World.Player)
	}
	for id, actor := range ctx.World.Actors {
		if actor.ChatRoomID != destroy.RoomID {
			continue
		}
		clearActorChatRoom(&actor)
		ctx.World.Actors[id] = actor
		return
	}
}

func clearActorChatRoom(actor *worldstate.Actor) {
	actor.ChatRoom = false
	actor.ChatRoomID = 0
	actor.ChatRoomTitle = ""
	actor.ChatRoomCount = 0
	actor.ChatRoomLimit = 0
	actor.ChatRoomPublic = false
}

func ActorHasChatRoom(actor worldstate.Actor) bool {
	return actor.ChatRoom && actor.ChatRoomID != 0 && actor.ChatRoomTitle != ""
}

func (m *System) HandleChatRoomAction(ctx client.Context, action gameui.ChatRoomWindowAction) {
	if action.Leave {
		if ctx.Network == nil {
			m.bindings.Console.AddErrorMessage("Leave chat room failed: not connected.")
			return
		}
		if err := ctx.Network.SendExitChatRoom(); err != nil {
			m.bindings.Console.AddErrorMessage("Leave chat room failed.")
			glog.Warnf("chat room leave failed: %v", err)
		}
		return
	}
	message := strings.TrimSpace(action.Message)
	if message == "" {
		return
	}
	name := scene.SelectedCharacterName(ctx.Session)
	if name == "" {
		m.bindings.Windows.ChatRoom.AddError(ctx, "send failed: missing character name")
		m.bindings.Console.AddErrorMessage("Chat room send failed: missing character name.")
		return
	}
	if ctx.Network == nil {
		m.bindings.Windows.ChatRoom.AddError(ctx, "send failed: not connected")
		m.bindings.Console.AddErrorMessage("Chat room send failed: not connected.")
		return
	}
	if err := client.SendGlobalChat(ctx, message); err != nil {
		m.bindings.Windows.ChatRoom.AddError(ctx, "send failed: "+err.Error())
		m.bindings.Console.AddErrorMessage("Chat room send failed.")
		glog.Warnf("chat room send failed: %v", err)
	}
}

func (m *System) RequestChatRoomEnter(ctx client.Context, actor worldstate.Actor) {
	if actor.ChatRoomID == 0 {
		return
	}
	if !actor.ChatRoomPublic {
		m.bindings.Console.AddErrorMessage("Private chat rooms need a password.")
		return
	}
	if ctx.Network == nil {
		m.bindings.Console.AddErrorMessage("Enter chat room failed: not connected.")
		return
	}
	if err := ctx.Network.SendEnterChatRoom(actor.ChatRoomID, ""); err != nil {
		m.bindings.Console.AddErrorMessage("Enter chat room failed.")
		glog.Warnf("chat room enter failed room=%d title=%q: %v", actor.ChatRoomID, actor.ChatRoomTitle, err)
		return
	}
	m.PendingChatRoom = network.ChatRoomCreate{
		Title:  actor.ChatRoomTitle,
		Limit:  actor.ChatRoomLimit,
		Public: actor.ChatRoomPublic,
	}
}

func (m *System) HandleChatRoomEnter(ctx client.Context, enter network.ChatRoomEnter) {
	members := make([]string, 0, len(enter.Members))
	owner := ""
	for _, member := range enter.Members {
		name := strings.TrimSpace(member.Name)
		if name == "" {
			continue
		}
		members = append(members, name)
		if member.Owner {
			owner = name
		}
	}
	if !m.bindings.Windows.ChatRoom.IsOpen() {
		room := m.PendingChatRoom
		if room.Title == "" {
			room = network.ChatRoomCreate{Title: "Chat Room", Limit: uint16(len(members)), Public: true}
		}
		m.bindings.Windows.ChatRoom.Open(ctx, room.Title, room.Limit, room.Public, members)
	}
	m.bindings.Windows.ChatRoom.SetMembers(ctx, members, owner)
	m.PendingChatRoom = network.ChatRoomCreate{}
}

func (m *System) HandleChatRoomMemberJoin(ctx client.Context, join network.ChatRoomMemberJoin) {
	if !m.bindings.Windows.ChatRoom.IsOpen() {
		return
	}
	name := strings.TrimSpace(join.Name)
	m.bindings.Windows.ChatRoom.AddMember(ctx, name, join.Count)
	m.bindings.Windows.ChatRoom.AddSystem(ctx, chatRoomMemberMessage(ctx, 179, "%s entered the chat room.", name))
}

func (m *System) HandleChatRoomMemberLeave(ctx client.Context, leave network.ChatRoomMemberLeave) {
	if !m.bindings.Windows.ChatRoom.IsOpen() {
		return
	}
	name := strings.TrimSpace(leave.Name)
	m.bindings.Windows.ChatRoom.RemoveMember(ctx, name, leave.Count)
	messageID := 180
	fallback := "%s left the chat room."
	if leave.Kicked {
		messageID = 181
		fallback = "%s has been kicked out of the chat room."
	}
	m.bindings.Windows.ChatRoom.AddSystem(ctx, chatRoomMemberMessage(ctx, messageID, fallback, name))
}

func (m *System) HandleChatRoomChange(ctx client.Context, change network.ChatRoomChange) {
	if !m.bindings.Windows.ChatRoom.IsOpen() {
		return
	}
	m.bindings.Windows.ChatRoom.UpdateRoom(ctx, change.Title, change.Limit, change.Count, change.Public)
}

func (m *System) HandleChatRoomRoleChange(ctx client.Context, role network.ChatRoomRoleChange) {
	if !m.bindings.Windows.ChatRoom.IsOpen() || !role.Owner {
		return
	}
	m.bindings.Windows.ChatRoom.SetOwner(ctx, role.Name)
}

func (m *System) AddChatRoomMessage(ctx client.Context, chat network.ChatMessage) {
	if !m.bindings.Windows.ChatRoom.IsOpen() {
		return
	}
	text := FormatConsoleMessage(ctx.Resources, chat)
	if text == "" {
		return
	}
	m.bindings.Windows.ChatRoom.AddMessage(ctx, text)
}

func chatRoomMemberMessage(ctx client.Context, messageID int, fallback string, name string) string {
	name = strings.TrimSpace(name)
	message := ""
	if ctx.Resources != nil {
		message, _ = ctx.Resources.MsgString(messageID)
	}
	if message == "" {
		message = fallback
	}
	if strings.Contains(message, "%s") {
		return strings.Replace(message, "%s", name, 1)
	}
	if name != "" {
		return strings.TrimSpace(message + " " + name)
	}
	return message
}

func chatRoomCreateAckMessage(ctx client.Context, ack network.ChatRoomCreateAck) string {
	if ctx.Resources != nil {
		if text, ok := ctx.Resources.MsgString(64 + int(ack.Result)); ok && strings.TrimSpace(text) != "" {
			return text
		}
	}
	switch ack.Result {
	case 0:
		return "Chat room has been created."
	case 1:
		return "Chat room limit exceeded."
	case 2:
		return "Same chat room already exists."
	default:
		return fmt.Sprintf("Create chat room failed (%d).", ack.Result)
	}
}
