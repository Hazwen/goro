package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) drawBerserkPotion(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	origin := effectWorldAnchor(ctx, effect, now)
	if texture := m.effectFileTexture(ctx.Resources, component.textureFile); texture != nil && frame < 50 {
		n := float64(frame + 1)
		height := (10 + n + .01*n*(n+1)/2) / 5
		tint := color.RGBA{255, 255, 255, effectParticleAlpha(frame, 50, 20, 40, 90)}
		var vertices []render.Vertex3D
		var indices []uint16
		for i := 0; i < 10; i++ {
			a, b := degreesToRadians(float64(i*36)), degreesToRadians(float64((i+1)*36))
			left := add3(origin, modelPoint3{x: .9 * math.Cos(a), z: .9 * math.Sin(a)})
			right := add3(origin, modelPoint3{x: .9 * math.Cos(b), z: .9 * math.Sin(b)})
			up := modelPoint3{y: height}
			appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{add3(left, up), add3(right, up), right, left}, tint)
		}
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFiles[0])
	if texture == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for spawn := 16; spawn < 110 && spawn <= frame; spawn += 4 {
		age := frame - spawn
		if age >= 40 {
			continue
		}
		salt := spawn * 4
		angle := degreesToRadians(float64(hitEffectRandom(effect, salt, 360)))
		radius := float64(2+hitEffectRandom(effect, salt+1, 7)) / 5
		speed := float64(20+hitEffectRandom(effect, salt+2, 50)) / (60 * 5)
		center := add3(origin, modelPoint3{x: radius * math.Sin(angle), y: speed * float64(age+1), z: radius * math.Cos(angle)})
		up := modelPoint3{y: float64(30+hitEffectRandom(effect, salt+3, 60)) / 50}
		tint := color.RGBA{240, 0, 0, effectParticleAlpha(age, 40, 20, 20, 180)}
		for _, side := range []modelPoint3{{x: .04}, {z: .04}} {
			appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{
				add3(add3(center, side), up), sub3(add3(center, side), up), sub3(sub3(center, side), up), add3(sub3(center, side), up),
			}, tint)
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
	}
}
