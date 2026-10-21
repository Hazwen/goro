package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) drawFullStrip(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if now.Before(effect.starts) || frame > 108 {
		return
	}
	for kind, path := range component.textureFiles {
		texture := m.effectFileTexture(ctx.Resources, path)
		if texture == nil {
			continue
		}
		count, size := 16, float64(5+kind*4)*math.Sqrt2/5
		if effect.effectID == effectRGCoin3 {
			count, size = 4, math.Sqrt2
		}
		for i := 0; i < count; i++ {
			seed := (kind*16 + i) * 4
			n := frame + 1 - 18 - hitEffectRandom(effect, seed, 30)
			alpha := min(n, 10)*25 - max(0, n-25)*7
			if n <= 0 || alpha <= 0 {
				continue
			}
			pitch := degreesToRadians(float64(hitEffectRandom(effect, seed+1, 360)))
			yaw := degreesToRadians(float64(hitEffectRandom(effect, seed+2, 360)))
			center := add3(effect.actorOrigin, effectParticlePoint(float64(12+n), 0, pitch, yaw))
			roll := degreesToRadians(float64(hitEffectRandom(effect, seed+3, 360) + 3*(frame+1)))
			drawTexturedEffectBillboardRotatedXY(screen, projection, texture, center.x, center.z, center.y, size, size, roll, color.RGBA{250, 100, 100, uint8(alpha)}, true)
		}
	}
}
