package social_test

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/kivutar/goro/game/social"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	gameui "github.com/kivutar/goro/ui"
)

func TestFormatConsoleMessageUsesMsgStringTable(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "msgstringtable.txt"), []byte("ignored#\nYou got %d items.#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	got := social.FormatConsoleMessage(manager, network.ChatMessage{MessageID: 1, Value: 3})
	if got != "You got 3 items." {
		t.Fatalf("message = %q", got)
	}
}

func TestColoredConsoleMessageUsesPacketColor(t *testing.T) {
	console := &gameui.ChatConsole{}
	social.AddConsoleMessage(console, nil, network.ChatMessage{Text: "Experience Gained Base:1 (0.01%) Job:1 (0.01%)", Color: 0x00B5FFB5, HasColor: true})

	messages := console.Messages()
	wantColor := color.RGBA{R: 0xB5, G: 0xFF, B: 0xB5, A: 255}
	if len(messages) != 1 || messages[0].Text != "Experience Gained Base:1 (0.01%) Job:1 (0.01%)" || messages[0].Color != wantColor {
		t.Fatalf("messages = %+v, want color %+v", messages, wantColor)
	}
}
