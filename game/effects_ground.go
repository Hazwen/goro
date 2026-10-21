package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

// Sanctuary, Magnus, Basilica, and Fog Wall share the native rectangular walls.
func (m *WorldMode) drawGroundWallsEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	if now.Before(effect.starts) {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	tick := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	vertices := make([]render.Vertex3D, 0, 96)
	indices := make([]uint16, 0, 144)
	center := modelPoint3{x: x, y: z + .2, z: y}
	if effect.effectID == effectBottomBasilica {
		appendBasilicaWalls(&vertices, &indices, texture, center, tick)
	} else {
		half, height, alpha, fade := 5.0, 50.0, 30, 864
		tint := color.RGBA{255, 255, 255, 255}
		if effect.effectID == effectBottomSanc {
			half, height, alpha, fade = 2.5, 16, 120, 1880
			tint.R, tint.B = 235, 235
		}
		if effect.effectID == effectFogWallGround {
			half, height, alpha, fade = 2.5, 32, 30, 29999
			tint.R, tint.G, tint.B = 80, 80, 80
		}
		phase := degreesToRadians(float64((hitEffectRandom(effect, 0, 360) + tick) % 360))
		height *= (.65 + .35*math.Sin(phase)) * math.Sin(degreesToRadians(float64(min(tick, 90))))
		tint.A = uint8(max(0, alpha-2*max(0, tick-fade)))
		appendSquareEffectBand(&vertices, &indices, texture, center, half/5, height/5, 0, 8*half/5, tint)
		if effect.effectID == effectFogWallGround {
			tint.A *= 2
			for _, sign := range []float64{-1, 1} {
				a := add3(center, modelPoint3{x: half / 5, z: sign * half / 5})
				b := add3(center, modelPoint3{x: -half / 5, z: -sign * half / 5})
				up := modelPoint3{y: height / 5}
				appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{add3(a, up), add3(b, up), b, a}, tint)
			}
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(effect.effectID != effectBottomSanc && effect.effectID != effectFogWallGround, false))
	}
}

// A band follows the square perimeter, splitting at corners. The native
// RenderTeiRect restarts the texture on each face, including partial faces.
func appendSquareEffectBand(vertices *[]render.Vertex3D, indices *[]uint16, texture *render.Image, center modelPoint3, half, height, start, length float64, tint color.RGBA) {
	if height <= 0 || tint.A == 0 {
		return
	}
	corners := [4]modelPoint3{{x: half, z: half}, {x: half, z: -half}, {x: -half, z: -half}, {x: -half, z: half}}
	position := math.Mod(start/(2*half), 4)
	side := int(position)
	offset := (position - float64(side)) * 2 * half
	for segment := 0; segment < 5 && length > 1e-9; segment++ {
		step := math.Min(length, 2*half-offset)
		a, b := corners[side], corners[(side+1)%4]
		direction := mul3(sub3(b, a), 1/(2*half))
		a = add3(center, add3(a, mul3(direction, offset)))
		b = add3(a, mul3(direction, step))
		up := modelPoint3{y: height}
		appendEffectQuad(vertices, indices, texture, [4]modelPoint3{add3(a, up), add3(b, up), b, a}, tint)
		length -= step
		side = (side + 1) % 4
		offset = 0
	}
}

func (m *WorldMode) drawPropertyGroundEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	if n <= 0 {
		return
	}
	radius, rise, alpha := propertyGroundRing(n, hitEffectRandom(effect, 1, 100))
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

// PrimGI4 resets when its rising angle falls below ten degrees. After the
// random initial fade, its 65-tick cycle can be sampled without replaying it.
func propertyGroundRing(tick, initialAlpha int) (radius, rise float64, alpha int) {
	reset := max(1, (initialAlpha+1)/2)
	if tick < reset {
		return 2.6 + .04*float64(tick), float64(80 - tick), max(0, initialAlpha-2*tick)
	}
	phase := (tick - reset) % 65
	alpha = 20*min(phase, 9) - 2*max(0, phase-9)
	return 2.14 + .04*float64(phase), float64(74 - phase), alpha
}

func (m *WorldMode) drawGroundSigilEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
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
