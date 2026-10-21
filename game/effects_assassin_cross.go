package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) drawSoulBreaker(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if now.Before(effect.starts) || frame > 61 {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	count := 4
	if effect.effectID == effectSoulBreaker2 {
		count = 32
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for i := 0; i < count; i++ {
		n := frame + 1 - 28 + i%4
		yaw := math.Atan2(effect.projectileDirection.z, effect.projectileDirection.x) + degreesToRadians(float64(hitEffectRandom(effect, i*2, 31)-15))
		if effect.effectID == effectSoulBreaker2 {
			n -= hitEffectRandom(effect, 100+i/4, 6)
			yaw = degreesToRadians(float64(i / 4 * 45))
		}
		alpha := 10*clampInt(n-1, 0, 9) - 5*max(0, n-10)
		if alpha <= 0 {
			continue
		}
		// PP_SOULBREAKER: acceleration changes after the first ten ticks.
		distance, speed := 0.0, .8/5
		for tick := 1; tick <= n; tick++ {
			if tick > 10 {
				speed *= 1.04
			} else if tick > 1 {
				speed *= 1.02
			}
			distance += speed
		}
		center := add3(effect.actorOrigin, modelPoint3{x: distance * math.Cos(yaw), z: distance * math.Sin(yaw), y: float64(10-hitEffectRandom(effect, i*2+1, 5)) / 5})
		var corners [4]modelPoint3
		for j := range corners {
			angle := yaw + math.Pi/4 + float64(3-j)*math.Pi/2
			corners[j] = add3(center, modelPoint3{x: 4 * math.Cos(angle), z: 4 * math.Sin(angle)})
		}
		// The 2008 renderer uses normal alpha and pale blue, unlike later dhxj.
		appendEffectQuad(&vertices, &indices, texture, corners, color.RGBA{200, 200, 255, uint8(alpha)})
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(false, false))
	}
}
