package actors_test

import (
	"math"
	"testing"
	"time"

	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func TestAppendActorDrawEntryUsesPathRenderDirection(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	actor := worldstate.Actor{
		X:            2,
		Y:            1,
		Dir:          4,
		Moving:       true,
		FromX:        0,
		FromY:        0,
		ToX:          2,
		ToY:          1,
		MoveStarted:  now.Add(-225 * time.Millisecond),
		MoveDuration: 450 * time.Millisecond,
		MovePath: []worldstate.WalkStep{
			{X: 0, Y: 0},
			{X: 0, Y: 1},
			{X: 1, Y: 1},
			{X: 2, Y: 1},
		},
	}
	projection := testutil.Projection(800, 600, 0.5, 1.5, 0)

	entries := actors.AppendActorDrawEntry(nil, world, projection, actor, false, now, 800, 600)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if got, want := entries[0].Actor.Dir, actors.DirectionFromDelta(0, 1, 1, 1, 4); got != want {
		t.Fatalf("entry direction = %d, want %d", got, want)
	}
}

func TestActorBillboardSortDepthUsesTopInCameraProjection(t *testing.T) {
	projection := testutil.Projection(800, 600, 10.5, 20.5, 0)
	footDepth := projection.Depth(10.5, 20.5, 0)
	topDepth := projection.Depth(10.5, 20.5, scene.ActorBillboardWorldHeightUnit)
	got := actors.ActorBillboardSortDepth(projection, 10.5, 20.5, 0)
	want := math.Min(footDepth, topDepth)
	if got != want {
		t.Fatalf("billboard depth = %.4f, want closer of foot %.4f and top %.4f", got, footDepth, topDepth)
	}
	if got >= footDepth {
		t.Fatalf("billboard depth = %.4f, want closer than foot depth %.4f", got, footDepth)
	}
}

func TestActorShadowFactorAveragesGNDLightmapAlpha(t *testing.T) {
	var lightmap res.GNDLightmap
	for y := range lightmap.Alpha {
		for x := range lightmap.Alpha[y] {
			lightmap.Alpha[y][x] = 128
		}
	}
	gnd := &res.GND{
		Width:     4,
		Height:    4,
		Lightmaps: []res.GNDLightmap{lightmap},
		Surfaces:  []res.GNDSurface{{LightmapID: 0}},
		Cells:     make([]res.GNDCell, 16),
	}
	for i := range gnd.Cells {
		gnd.Cells[i].Top = 0
	}

	got := actors.ActorShadowFactor(&worldstate.World{GND: gnd}, 3, 3)
	want := float64(128) / 255
	if math.Abs(got-want) > 0.0001 {
		t.Fatalf("shadow factor = %.4f, want %.4f", got, want)
	}
}

func TestActorShadowFactorIgnoresGroundShadowBelowElevatedGAT(t *testing.T) {
	var lightmap res.GNDLightmap
	for y := range lightmap.Alpha {
		for x := range lightmap.Alpha[y] {
			lightmap.Alpha[y][x] = 32
		}
	}
	gnd := &res.GND{
		Width:     4,
		Height:    4,
		Lightmaps: []res.GNDLightmap{lightmap},
		Surfaces:  []res.GNDSurface{{LightmapID: 0}},
		Cells:     make([]res.GNDCell, 16),
	}
	for i := range gnd.Cells {
		gnd.Cells[i].Top = 0
	}
	gat := &res.GAT{
		Width:  8,
		Height: 8,
		Cells:  make([]res.GATCell, 64),
	}
	for i := range gat.Cells {
		gat.Cells[i].Heights = [4]float32{2, 2, 2, 2}
	}

	got := actors.ActorShadowFactor(&worldstate.World{GAT: gat, GND: gnd}, 3, 3)
	if got != 1 {
		t.Fatalf("elevated GAT shadow factor = %.4f, want 1", got)
	}
}

func TestActorShadowFactorDefaultsToLitWithoutGroundLightmap(t *testing.T) {
	if got := actors.ActorShadowFactor(nil, 3, 3); got != 1 {
		t.Fatalf("nil world shadow = %.2f, want 1", got)
	}
	gnd := &res.GND{
		Width:    1,
		Height:   1,
		Surfaces: []res.GNDSurface{{LightmapID: 0}},
		Cells:    []res.GNDCell{{Top: -1}},
	}
	if got := actors.ActorShadowFactor(&worldstate.World{GND: gnd}, 0, 0); got != 1 {
		t.Fatalf("missing top surface shadow = %.2f, want 1", got)
	}
}

func TestWarpActorUsesCenteredWorldAnchor(t *testing.T) {
	actor := worldstate.Actor{Job: scene.ActorJobWarpPortal}
	x, y := actors.ActorWorldAnchor(actor, 30, 40)
	if x != 30.5 || y != 40.5 {
		t.Fatalf("warp anchor = %.1f, %.1f; want 30.5, 40.5", x, y)
	}
}

func TestNormalActorUsesCenteredWorldAnchor(t *testing.T) {
	actor := worldstate.Actor{Job: 1002}
	x, y := actors.ActorWorldAnchor(actor, 30, 40)
	if x != 30.5 || y != 40.5 {
		t.Fatalf("normal actor anchor = %.1f, %.1f; want 30.5, 40.5", x, y)
	}
}

func TestActorOverlayLifeBarIsBelowNameLabel(t *testing.T) {
	nameY := actors.ActorNameLabelY(100, 1.2)
	barY := actors.ActorLifeBarY(100, 1.2)
	if barY <= nameY+10 {
		t.Fatalf("bar y = %.1f, name y = %.1f; want bar below name", barY, nameY)
	}
}

func TestActorOverlayCastBarIsBelowSpeechBubble(t *testing.T) {
	bubbleBottomY := actors.ActorSpeechBubbleBottomY(100, 1.2)
	barY := actors.ActorCastBarY(100, 1.2)
	if barY <= bubbleBottomY {
		t.Fatalf("cast bar y = %.1f, bubble bottom y = %.1f; want cast bar below bubble", barY, bubbleBottomY)
	}
}
