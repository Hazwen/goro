package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) applyActorDevotion(ctx client.Context, notify network.ActorDevotion) {
	active := m.worldEffects[:0]
	for _, effect := range m.worldEffects {
		if effect.effectID != effectLineLink || effect.actorID != notify.SourceID {
			active = append(active, effect)
		}
	}
	m.worldEffects = active
	now := time.Now()
	for i, targetID := range notify.Targets {
		if targetID == 0 || targetID == notify.SourceID {
			continue
		}
		duplicate := false
		for _, previous := range notify.Targets[:i] {
			duplicate = duplicate || previous == targetID
		}
		if duplicate {
			continue
		}
		if m.addWorldEffectBetweenAt(ctx, effectLineLink, notify.SourceID, targetID, now) {
			m.worldEffects[len(m.worldEffects)-1].expires = time.Time{}
		}
	}
}

func (m *WorldMode) drawDevotionLinkEffect(screen *render.Frame, ctx client.Context, _ sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	if _, ok, _ := actorForCombatID(ctx, effect.targetID); !ok {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	source := effectWorldAnchor(ctx, effect, now)
	target := effectWorldAnchor(ctx, worldEffect{actorID: effect.targetID}, now)
	source.y += 1.6
	target.y += 1.6
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	target = add3(source, mul3(sub3(target, source), math.Sin(degreesToRadians(float64(min(n, 15)*6)))))
	angle := math.Atan2(target.z-source.z, target.x-source.x) + math.Pi/2
	pulse := n % 10
	alpha := 45 + 2*min(pulse, 4) - 2*max(0, pulse-4)
	vertices, indices := []render.Vertex3D{}, []uint16{}
	for layer := 0; layer < 2; layer++ {
		radius := .08 * float64(layer+1)
		tint := color.RGBA{255, 255, 0, uint8(alpha)}
		if layer == 1 {
			tint = color.RGBA{0, 0, 255, uint8(alpha * 2)}
		}
		offset := func(j int) modelPoint3 {
			a := float64(j) * math.Pi / 5
			r := radius * math.Cos(a)
			return modelPoint3{x: r * math.Cos(angle), y: radius * math.Sin(a), z: r * math.Sin(angle)}
		}
		for j := 0; j < 10; j++ {
			a, b := offset(j), offset(j+1)
			appendSoulLinkerQuad(&vertices, &indices, texture, [4]modelPoint3{
				add3(target, a), add3(target, b), add3(source, b), add3(source, a),
			}, tint)
		}
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
}
