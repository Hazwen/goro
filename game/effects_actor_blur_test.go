package game

import (
	"image/color"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

func TestOverthrustAfterimages(t *testing.T) {
	ctx := archerEffectContext()
	start := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 10, 20, 0)
	billboard := &spriteBillboard{image: render.NewImage(20, 40)}
	billboard.image.Fill(color.RGBA{255, 255, 255, 255})
	for _, status := range []uint16{db.StatusOverthrust, db.StatusOverthrustmax} {
		m := &WorldMode{}
		m.applyStatusEffectChange(ctx, network.StatusEffectChange{ActorID: 2000000, StatusID: status, Active: true})
		entry := sceneActorDrawEntry{actor: ctx.World.Player, isPlayer: true, worldX: 10.5, worldY: 20.5, scale: 1}
		tint := actorStateTint(entry.actor)
		if tint != (color.RGBA{250, 150, 150, 255}) {
			t.Fatalf("incorrect native Overthrust tint: %v", tint)
		}
		for tick := 0; tick <= 100; tick++ {
			frame := render.NewFrame(800, 600)
			entry.worldX = 10.5 + float64(tick)/10
			m.drawActorSpriteWithEffects3D(ctx, frame, projection, entry, billboard, 1, 1, 1, tint, render.BlendSourceOver, start.Add(time.Duration(tick)*db.EffectFrameDuration))
			if len(m.actorBlurs[entry.actor.ID]) > 5 {
				t.Fatal("afterimages must remain bounded during long movement")
			}
			if tick == 5 {
				commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
				if commands.Len() != 3 {
					t.Fatal("expected two frozen poses and the current actor")
				}
				old := commands.Index(0)
				if old.FieldByName("Center").Index(0).Float() != 10.5 || math.Abs(old.FieldByName("ColorA").Float()-140.0/255) > .005 {
					t.Fatal("afterimage must keep its captured position and fade by eight per tick")
				}
				if old.FieldByName("Options").FieldByName("DepthWrite").Bool() {
					t.Fatal("fading afterimages must not occlude the actor or other sprites")
				}
			}
		}
		m.applyStatusEffectChange(ctx, network.StatusEffectChange{ActorID: 2000000, StatusID: status})
		entry.actor = ctx.World.Player
		m.drawActorBlurs(render.NewFrame(800, 600), projection, entry, billboard, 1, 1, 1, tint, start.Add(124*db.EffectFrameDuration))
		if len(m.actorBlurs) != 0 {
			t.Fatal("afterimages survived the end of the buff and fade")
		}
	}
}

func TestOverthrustAfterimageVisibilityAndRemoval(t *testing.T) {
	ctx := archerEffectContext()
	entry := sceneActorDrawEntry{actor: ctx.World.Player, isPlayer: true, worldX: 10.5, worldY: 20.5, scale: 1}
	entry.actor.Opt3State = db.Opt3Overthrust
	projection := newSceneProjectionForTarget(800, 600, 10, 20, 0)
	billboard := &spriteBillboard{image: render.NewImage(20, 40)}
	billboard.image.Fill(color.RGBA{255, 255, 255, 255})
	for _, kind := range []string{"dyed", "monster", "hidden", "removed", "remote"} {
		m := &WorldMode{}
		e := entry
		switch kind {
		case "dyed":
			e.actor.BodyPal = 1
		case "monster":
			e.isPlayer = false
			e.actor = ctx.World.Actors[300]
			e.actor.Opt3State = db.Opt3Overthrust
		case "hidden":
			m.drawActorBlurs(render.NewFrame(800, 600), projection, e, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, time.Now())
			e.actor.EffectState = db.EffectStateHide
		case "remote":
			e.isPlayer, e.actor.Appearance = false, true
		}
		m.drawActorBlurs(render.NewFrame(800, 600), projection, e, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, time.Now())
		if kind == "removed" || kind == "remote" {
			if len(m.actorBlurs) != 1 {
				t.Fatal("visible player did not produce an afterimage")
			}
			m.clearRemovedActorState(e.actor.ID)
		}
		if len(m.actorBlurs) != 0 {
			t.Fatalf("%s actor retained afterimages", kind)
		}
	}
}
