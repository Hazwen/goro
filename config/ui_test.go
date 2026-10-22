package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUISettingsRoundTrip(t *testing.T) {
	cfg := defaultConfig()
	cfg.ConfigPath = filepath.Join(t.TempDir(), "goro.ini")
	if err := os.WriteFile(cfg.ConfigPath, []byte("; keep this comment\n[render]\nvsync = true\n[ui]\nskills_open = false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	want := UISettings{
		InventoryOpen: true, SkillsOpen: true, PetOpen: true,
		HomunculusSkillsOpen: true, MercenaryOpen: true, CartOpen: true,
		BasicInfoCompact: true, BasicMenuCollapsed: true, SkillTree: true,
		MinimapHidden: true, ShortcutRows: 3,
		ShortcutPosition: WindowPosition{X: 120, Y: 240, Set: true},
		ConsolePosition:  WindowPosition{X: 20, Y: 300, Set: true},
		StatsPosition:    WindowPosition{X: 300, Y: 80, Set: true},
		CameraZoom:       142.75,
	}
	if _, err := cfg.SaveUISettings(want); err != nil {
		t.Fatal(err)
	}
	if _, err := cfg.SaveUserSettings(UserSettings{VSync: true}); err != nil {
		t.Fatal(err)
	}
	got, err := cfg.LoadUISettings()
	if err != nil || got != want {
		t.Fatalf("reload = %+v, %v; want %+v", got, err, want)
	}
	data, err := os.ReadFile(cfg.ConfigPath)
	if err != nil || !strings.Contains(string(data), "; keep this comment") || !strings.Contains(string(data), "vsync = true") {
		t.Fatalf("other settings lost: %s, %v", data, err)
	}
	loaded := defaultConfig()
	if err := applyINI(&loaded, strings.NewReader(string(data))); err != nil || loaded.UI != want {
		t.Fatalf("startup load = %+v, %v", loaded.UI, err)
	}
	want.SkillsOpen, want.SkillTree, want.ShortcutRows = false, false, 1
	want.ConsolePosition = WindowPosition{}
	want.StatsPosition = WindowPosition{}
	if _, err := cfg.SaveUISettings(want); err != nil {
		t.Fatal(err)
	}
	if got, err = cfg.LoadUISettings(); err != nil || got != want {
		t.Fatalf("second reload = %+v, %v; want %+v", got, err, want)
	}
}

func TestUISettingsValidation(t *testing.T) {
	for _, entry := range []string{
		"shortcut_rows = 0", "shortcut_rows = 4", "shortcut_rows = three", "skills_open = sometimes", "bogus = true",
		"shortcut_position = 10", "console_position = x,20", "console_position = 10,20,30",
		"stats_position = 10,no",
		"camera_zoom = -1", "camera_zoom = NaN", "camera_zoom = +Inf", "camera_zoom = far",
	} {
		cfg := defaultConfig()
		if err := applyINI(&cfg, strings.NewReader("[ui]\n"+entry)); err == nil {
			t.Errorf("accepted %s", entry)
		}
	}
	cfg := defaultConfig()
	if err := applyINI(&cfg, strings.NewReader("[ui]\nshortcut_rows = 2\nskill_tree = yes")); err != nil || cfg.UI.ShortcutRows != 2 || !cfg.UI.SkillTree {
		t.Fatalf("valid settings = %+v, %v", cfg.UI, err)
	}
}

func TestUIWindowPositionParsing(t *testing.T) {
	cfg := defaultConfig()
	if err := applyINI(&cfg, strings.NewReader("[ui]\nconsole_position = -20, 800\nshortcut_position =")); err != nil {
		t.Fatal(err)
	}
	if cfg.UI.ConsolePosition != (WindowPosition{X: -20, Y: 800, Set: true}) || cfg.UI.ShortcutPosition.Set {
		t.Fatalf("positions = %+v", cfg.UI)
	}
}
