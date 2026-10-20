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

func rogueBurstParticle(effect worldEffect, index, frame int) ([4]modelPoint3, color.RGBA) {
	salt := index * 5
	delay, rise, tint := 70, 15, color.RGBA{100, 150, 255, 0}
	if effect.effectID == effectRogueCoin {
		delay, rise, tint = 18, 25, color.RGBA{250, 250, 155, 0}
	}
	pitch := degreesToRadians(float64(hitEffectRandom(effect, salt+1, 360)))
	yaw := degreesToRadians(float64(hitEffectRandom(effect, salt+2, 360)))
	var corners [4]modelPoint3
	if effect.effectID == effectTeiHit3 {
		age := frame + 1 - 20 - hitEffectRandom(effect, salt, 10)
		alpha := 15*clampInt(age-10, 0, 10) - 5*max(0, age-20)
		if alpha <= 0 {
			return corners, tint
		}
		tint = color.RGBA{200, 200, 255, uint8(alpha)}
		distance := float64(-10 + age*2)
		for j, p := range [4][2]float64{{16, -1}, {16, 1}, {-16, 1}, {-16, -1}} {
			corners[j] = effectParticlePoint(distance+p[0], p[1], pitch, yaw)
		}
		return corners, tint
	}
	age := frame + 1 - delay - hitEffectRandom(effect, salt, 30)
	alpha := rise*clampInt(age, 0, 10) - 7*max(0, age-25)
	if alpha <= 0 {
		return corners, tint
	}
	tint.A = uint8(alpha)
	center := effectParticlePoint(12+1.5*float64(age), 0, pitch, yaw)
	roll := degreesToRadians(float64(hitEffectRandom(effect, salt+3, 360)))
	spin := degreesToRadians(float64(hitEffectRandom(effect, salt+4, 360) + 5*(frame+1)))
	for j := range corners {
		corners[j] = add3(center, effectParticlePoint(3.5, 0, roll+float64(j)*math.Pi/2, spin))
	}
	return corners, tint
}

func (m *WorldMode) drawRogueBurst(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	count := 80
	if effect.effectID == effectRogueCoin {
		count = 120
	}
	vertices := make([]render.Vertex3D, 0, count*4)
	indices := make([]uint16, 0, count*6)
	for i := 0; i < count; i++ {
		corners, tint := rogueBurstParticle(effect, i, frame)
		if tint.A == 0 {
			continue
		}
		// RenderTeiRect maps its first edge to the bottom of the texture.
		for j := range corners {
			corners[j] = add3(corners[j], effect.actorOrigin)
		}
		appendEffectQuad(&vertices, &indices, texture,
			[4]modelPoint3{corners[3], corners[2], corners[1], corners[0]}, tint)
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(effect.effectID != effectTeiHit3, false))
	}
}

func (m *WorldMode) applyCloseConfineStatus(ctx client.Context, change network.StatusEffectChange) {
	if change.StatusID != db.StatusRgCconfineM && change.StatusID != db.StatusRgCconfineS {
		return
	}
	id := change.ActorID
	if id == 0 || isLocalActor(ctx, id) {
		id = localSkillTarget(ctx)
	}
	m.setActorStateEffect(ctx, id, effectNPCStop2, change.Active)
	if change.Active && change.HasDuration {
		now := time.Now()
		for i := range m.worldEffects {
			effect := &m.worldEffects[i]
			if effect.effectID == effectNPCStop2 && effect.actorID == id && !effect.expired(now) {
				effect.expires = now.Add(change.Duration)
			}
		}
	}
}
