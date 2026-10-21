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

// The native ring primitives have a height at each of their 21 vertices.
func (m *WorldMode) drawCrusaderDefenseEffect(screen *render.Frame, ctx client.Context, _ sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	alpha := min(n, 80) - 5*max(0, n-179)
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if alpha <= 0 || texture == nil {
		return
	}
	origin := effectWorldAnchor(ctx, effect, now)
	vertices, indices := []render.Vertex3D{}, []uint16{}
	for i := 0; i < 4; i++ {
		rotationSpeed, waveSpeed := 1+i%2, 1+i/2
		height := float64(40-i) / 5 * (1 + .3*math.Sin(degreesToRadians(float64(n*waveSpeed+i*90))))
		appendEffectBand(&vertices, &indices, texture, origin, (8-.1*float64(i))/5,
			degreesToRadians(float64(180+90*i+n*rotationSpeed)), 2*math.Pi, 0, 20, color.RGBA{255, 255, 255, uint8(alpha)},
			func(j int) float64 { return height * math.Sin(degreesToRadians(float64(j*9))) })
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(false, false))
}

func (m *WorldMode) drawProvidenceEffect(screen *render.Frame, ctx client.Context, _ sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	alpha := 40 - max(0, n-50)
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if alpha <= 0 || texture == nil {
		return
	}
	origin := effectWorldAnchor(ctx, effect, now)
	vertices, indices := []render.Vertex3D{}, []uint16{}
	for i := 0; i < 4; i++ {
		rise := degreesToRadians(float64(83 - i))
		height := 24 * math.Sin(degreesToRadians(float64(min(n, 90)))) * math.Sin(rise)
		appendEffectBand(&vertices, &indices, texture, origin, (6+.5*float64(i))/5,
			degreesToRadians(float64(90*i+n)), 2*math.Pi, 1/math.Tan(rise), 20, color.RGBA{255, 255, 255, uint8(alpha)},
			func(int) float64 { return height })
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
}

func (m *WorldMode) drawGrandCrossEffect(screen *render.Frame, ctx client.Context, _ sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	origin := effect.actorOrigin
	if texture := m.effectFileTexture(ctx.Resources, component.textureFile); texture != nil {
		alpha := min(n, 9)*10 - max(0, n-9)
		if alpha > 0 {
			vertices, indices := []render.Vertex3D{}, []uint16{}
			for i, offset := range []modelPoint3{{x: 4.6, z: 4.6}, {x: 4.6, z: -4.6}, {x: -4.6, z: -4.6}, {x: -4.6, z: 4.6}} {
				height := 24 * math.Sin(degreesToRadians(float64(n)))
				appendEffectBand(&vertices, &indices, texture, add3(origin, offset), (19.9-.01*float64(n))/5,
					degreesToRadians(float64(180-90*i)), degreesToRadians(80), 0, 8, color.RGBA{255, 175, 175, uint8(alpha)}, func(int) float64 { return height })
			}
			screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
		}
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFiles[0])
	alpha := min(n, 9)*9 - max(0, n-9)
	if texture == nil || alpha <= 0 {
		return
	}
	vertices, indices := []render.Vertex3D{}, []uint16{}
	floorVertices, floorIndices := []render.Vertex3D{}, []uint16{}
	length, width := (24+.005*float64(n))/5, (3.2+.005*float64(n))/5
	height := 12 * math.Sin(degreesToRadians(float64(n)))
	for arm := 0; arm < 2; arm++ {
		corners := [4]modelPoint3{{x: -length, z: -width}, {x: length, z: -width}, {x: length, z: width}, {x: -length, z: width}}
		for i := range corners {
			if arm == 1 {
				corners[i].x, corners[i].z = -corners[i].z, corners[i].x
			}
			corners[i] = add3(origin, corners[i])
			corners[i].y += .2
		}
		for repeat := 0; repeat < 2; repeat++ {
			appendEffectQuad(&floorVertices, &floorIndices, texture, corners, color.RGBA{255, 255, 255, uint8(alpha / 2)})
		}
		for i, bottom := range corners {
			next := corners[(i+1)%4]
			appendEffectQuad(&vertices, &indices, texture,
				[4]modelPoint3{add3(bottom, modelPoint3{y: height}), add3(next, modelPoint3{y: height}), next, bottom},
				color.RGBA{255, 255, 255, uint8(alpha)})
		}
	}
	screen.DrawTriangles3DOwned(floorVertices, floorIndices, texture, texturedEffectBillboardDrawOptions(true, false))
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(false, false))
}

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
			appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{
				add3(target, a), add3(target, b), add3(source, b), add3(source, a),
			}, tint)
		}
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
}
