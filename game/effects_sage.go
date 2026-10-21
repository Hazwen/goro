package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

// VOLCANO / PrimGI2: four expanding rings at the selected ground cell.
func (m *WorldMode) drawSageGroundCastEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	heightScale, initialRise, fade := 5.0, 80, 2
	if effect.effectID == effectGanbantein {
		heightScale, initialRise, fade = 3, 70, 8
	}
	fadeIn := 20
	if effect.effectID == effectGumgang3 {
		fadeIn = 10
	}
	alpha := fadeIn*min(n, 10) - fade*max(0, n-10)
	if n <= 0 || alpha <= 0 {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	rise := degreesToRadians(float64(max(40, initialRise-n)))
	origin := modelPoint3{x: x, y: z, z: y}
	vertices, indices := []render.Vertex3D{}, []uint16{}
	for i := 0; i < 4; i++ {
		height := heightScale * math.Sin(rise) * (1 + .15*math.Sin(degreesToRadians(float64(n+i*90))))
		appendEffectBand(&vertices, &indices, texture, origin, (float64(i+1)+.3*float64(n))/5,
			degreesToRadians(float64(i*90+n*5)), 2*math.Pi, 1/math.Tan(rise), 20,
			color.RGBA{255, 255, 255, uint8(alpha)},
			func(j int) float64 { return height * math.Sin(degreesToRadians(float64(j*9))) })
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(false, false))
}

// PrimGI4 resets when its rising angle falls below ten degrees. After the
// random initial fade, its 65-tick cycle can be sampled without replaying it.
func sageGroundRing(tick, initialAlpha int) (radius, rise float64, alpha int) {
	reset := max(1, (initialAlpha+1)/2)
	if tick < reset {
		return 2.6 + .04*float64(tick), float64(80 - tick), max(0, initialAlpha-2*tick)
	}
	phase := (tick - reset) % 65
	alpha = 20*min(phase, 9) - 2*max(0, phase-9)
	return 2.14 + .04*float64(phase), float64(74 - phase), alpha
}

func (m *WorldMode) drawPropertyGroundEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	if n <= 0 {
		return
	}
	radius, rise, alpha := sageGroundRing(n, hitEffectRandom(effect, 1, 100))
	if alpha <= 0 {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	rise = degreesToRadians(rise)
	origin := modelPoint3{x: x, y: z, z: y}
	vertices, indices := []render.Vertex3D{}, []uint16{}
	tint := color.RGBA{255, 255, 255, uint8(alpha)}
	if effect.effectID == effectBottomSuiton {
		tint.R, tint.G = 120, 120
	}
	appendEffectBand(&vertices, &indices, texture, origin, radius/5,
		degreesToRadians(float64(hitEffectRandom(effect, 0, 360))), 2*math.Pi, 1/math.Tan(rise), 10,
		tint, func(j int) float64 {
			return 3 * math.Sin(rise) * (1 + .3*math.Sin(degreesToRadians(float64(j*9)))*math.Sin(degreesToRadians(float64(n%360))))
		})
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(false, false))
}

func (m *WorldMode) drawLandProtectorGroundEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	if n <= 0 {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	speed, size := 3, 7.0
	if effect.effectID == effectBottomSpider {
		speed, size = 1, 12
	}
	phase := (hitEffectRandom(effect, 0, 360) + speed*n) % 360
	radius := size * (.9 + .1*math.Sin(degreesToRadians(float64(phase)))) / 5
	origin := modelPoint3{x: x, y: z + 2.0/5, z: y}
	vertices, indices := []render.Vertex3D{}, []uint16{}
	appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{
		add3(origin, modelPoint3{z: -radius}), add3(origin, modelPoint3{x: -radius}),
		add3(origin, modelPoint3{z: radius}), add3(origin, modelPoint3{x: radius}),
	}, color.RGBA{255, 255, 255, 100})
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
}
