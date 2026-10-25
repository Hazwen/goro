package ui

import (
	"fmt"
	"testing"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/core/dropdown"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/session"
)

func TestGuildPositionDropdownSelectsLastRank(t *testing.T) {
	for _, y := range []int{50, 600} {
		t.Run(fmt.Sprint(y), func(t *testing.T) {
			ctx, _, app := newWindowInstanceTest()
			app.Window().SetRenderMode(uiapp.RenderModeFrameworkManaged)
			app.Window().HandleResize(ctx.ScreenW, ctx.ScreenH)
			ctx.Session = &session.Session{Guild: session.Guild{
				IsMaster: true, MenuAccess: 0xff,
				Members: []session.GuildMember{{AccountID: 2, CharID: 20, PositionID: 1, CharName: "Alice"}},
			}}
			for i := uint32(0); i < 20; i++ {
				ctx.Session.Guild.Positions = append(ctx.Session.Guild.Positions, session.GuildPosition{PositionID: i, PosName: fmt.Sprintf("Rank %02d", i)})
			}
			w := &GuildWindow{tab: guildWindowTabMembers}
			w.OpenWindow(ctx)
			w.setPosition(ctx, 100, y)
			w.Publish(ctx)
			app.Frame()
			app.Window().DrawTo(&uitest.MockCanvas{})
			var find func(widget.Widget) *dropdown.Widget
			find = func(root widget.Widget) *dropdown.Widget {
				if d, ok := root.(*dropdown.Widget); ok {
					return d
				}
				for _, c := range root.Children() {
					if d := find(c); d != nil {
						return d
					}
				}
				return nil
			}
			d := find(w.content)
			if d == nil {
				t.Fatal("no dropdown")
			}
			p := d.ScreenBounds().Center()
			app.Window().HandleEvent(uitest.Click(p.X, p.Y))
			app.Window().HandleEvent(uitest.Release(p.X, p.Y))
			app.Frame()
			app.Window().DrawTo(&uitest.MockCanvas{})
			overlays := app.Window().OverlayContentWidgets()
			if len(overlays) != 1 {
				t.Fatalf("overlays: %d", len(overlays))
			}
			bounds := overlays[0].(interface{ Bounds() geometry.Rect }).Bounds()
			if bounds.Height() != 8*22 {
				t.Fatalf("menu height = %v, want eight compact rows", bounds.Height())
			}
			if bounds.Min.Y < 0 || bounds.Max.Y > float32(ctx.ScreenH) {
				t.Fatalf("menu %v extends outside the viewport", bounds)
			}
			p = bounds.Center()
			for i := 0; i < 12; i++ {
				app.Window().HandleEvent(uitest.WheelScroll(p.X, p.Y, -1))
				app.Frame()
				canvas := &uitest.MockCanvas{}
				if !app.Window().HasDirtyOverlays() {
					t.Fatal("scroll did not request an overlay repaint")
				}
				if !app.Window().DrawTo(canvas) {
					t.Fatal("scroll repaint was skipped")
				}
				if !app.Window().LastDirtyUnion().ContainsRect(bounds) {
					t.Fatalf("repaint bounds %v do not cover menu %v", app.Window().LastDirtyUnion(), bounds)
				}
				if i == 11 {
					found := false
					for _, text := range canvas.StyledTexts {
						if text.Text == "Rank 19" {
							found = true
						}
					}
					if !found {
						t.Fatal("last rank was not painted after scrolling")
					}
				}
			}
			if app.Window().HasDirtyOverlays() || app.Window().DrawTo(&uitest.MockCanvas{}) {
				t.Fatal("unchanged menu kept requesting redraws")
			}
			p = geometry.Pt(bounds.Min.X+10, bounds.Max.Y-10)
			app.Window().HandleEvent(uitest.Click(p.X, p.Y))
			app.Window().HandleEvent(uitest.Release(p.X, p.Y))
			changes := w.memberPositionChanges(ctx)
			if len(changes) != 1 || changes[0].PositionID != 19 {
				t.Fatalf("last rank selection=%+v", changes)
			}
		})
	}
}
