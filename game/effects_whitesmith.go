package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) drawCartTermination(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	frame := int(now.Sub(effect.starts)/db.EffectFrameDuration) - 30
	if frame < 0 {
		return
	}
	n := frame + 1
	alpha := max(0, 230-6*max(0, n-25))
	if alpha == 0 {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	center := effectWorldAnchor(ctx, effect, now)
	right, up, _, ok := projection.BillboardBasis(center.x, center.z, center.y)
	if texture == nil || !ok {
		return
	}
	// Four ParticleSpread primitives, each with four groups of three lights.
	distance := (float64(min(n, 26)) + 9*(1-math.Pow(.9, float64(max(0, n-26))))) / 5
	var vertices []render.Vertex3D
	var indices []uint16
	for group := 0; group < 16; group++ {
		salt := group * 8
		half := (2.5 + .1*float64(hitEffectRandom(effect, salt, 6))) / (5 * math.Sqrt2)
		pink := uint8(200 - 40*hitEffectRandom(effect, salt+7, 3))
		r, u := mul3(right, half), mul3(up, half)
		for i := 0; i < 3; i++ {
			yaw := degreesToRadians(float64(hitEffectRandom(effect, salt+1+i*2, 360)))
			rise := degreesToRadians(float64(hitEffectRandom(effect, salt+2+i*2, 180)))
			offset := modelPoint3{x: distance * math.Cos(rise) * math.Sin(yaw), y: distance * math.Sin(rise), z: distance * math.Cos(rise) * math.Cos(yaw)}
			if offset.y < half*up.y {
				continue
			}
			position := add3(center, offset)
			appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{
				add3(sub3(position, r), u), add3(add3(position, r), u), sub3(add3(position, r), u), sub3(sub3(position, r), u),
			}, color.RGBA{255, pink, pink, uint8(alpha)})
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
	}
}
