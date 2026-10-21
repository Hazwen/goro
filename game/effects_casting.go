package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

type castingRing struct {
	process, alpha, angle int
	radius                float64
}

type blueCasting struct {
	tick      int
	rings     [8]castingRing
	auraBlade bool
}

// BlueCasting launches two four-layer SAINTCASTING rings. PrimGI1 makes
// each ring flare and reset until the final half-second of the cast.
func (p *blueCasting) advance(frame, duration int) {
	if p.tick == 0 {
		for i := range p.rings {
			r := &p.rings[i]
			r.radius, r.angle = 4.1, 80
			r.alpha = 45 - (i/4)*20 + (3-i%4)*45
			if duration <= 60 {
				r.alpha, r.process = 0, -(i%4)*5
			}
		}
	}
	for p.tick < min(frame+1, duration) {
		p.tick++
		for i := range p.rings {
			r := &p.rings[i]
			r.process++
			if r.process <= 0 {
				continue
			}
			r.radius += .07
			r.angle--
			if r.angle < 10 {
				r.angle, r.alpha = 10, 0
			}
			if r.radius >= 4 {
				r.alpha = max(r.alpha-3, 0)
				if r.alpha == 0 && r.process < duration-30 {
					r.radius, r.angle = 3.37, 74
					if p.auraBlade {
						r.angle = 64
					}
				}
			} else {
				if p.auraBlade {
					r.alpha += 5
				} else {
					r.alpha += 10
				}
			}
		}
	}
}

func (m *WorldMode) drawBlueCastingEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	if now.Before(effect.starts) || effect.blueCasting == nil {
		return
	}
	duration := 60
	if effect.duration > 0 {
		duration = max(1, int(effect.duration/db.EffectFrameDuration))
	} else if !effect.expires.IsZero() {
		duration = max(1, int(effect.expires.Sub(effect.starts)/db.EffectFrameDuration))
	}
	effect.blueCasting.advance(int(now.Sub(effect.starts)/db.EffectFrameDuration), duration)
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for i, r := range effect.blueCasting.rings {
		if i == 4 && effect.blueCasting.auraBlade {
			screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
			vertices, indices = nil, nil
			texture = m.effectFileTexture(ctx.Resources, component.textureFiles[0])
			if texture == nil {
				return
			}
		}
		if r.alpha <= 0 {
			continue
		}
		layer := i % 4
		phase := r.process + layer*90
		if layer >= 2 {
			phase += r.process
		}
		riseSin, riseCos := math.Sincos(degreesToRadians(float64(r.angle)))
		base := uint16(len(vertices))
		for segment := 0; segment <= 20; segment++ {
			sin, cos := math.Sincos(degreesToRadians(float64(segment*18 + (layer+2)*90)))
			maxHeight := 20 - layer
			if effect.effectID == effectBeginAsura || effect.effectID == effectBeginAsura11 {
				maxHeight = 25 - layer
			}
			if effect.blueCasting.auraBlade {
				maxHeight = 15 - layer
			}
			height := float64(maxHeight) * (1 + .3*math.Sin(degreesToRadians(float64(segment*9)))*math.Sin(degreesToRadians(float64(phase))))
			bottom := modelPoint3{x: x + cos*r.radius/5, y: z, z: y + sin*r.radius/5}
			top := add3(bottom, modelPoint3{x: cos * riseCos * height / 5, y: riseSin * height / 5, z: sin * riseCos * height / 5})
			u := float32(segment) / 21
			c := color.RGBA{100, 100, 255, uint8(clampInt(r.alpha, 0, 255))}
			if effect.blueCasting.auraBlade || effect.effectID == effectBeginAsura || effect.effectID == effectBeginAsura11 {
				c.R, c.G = 255, 255
			}
			if effect.effectID == effectCoupleCasting {
				c.R, c.G, c.B = 255, 89, 182
			}
			w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
			vertices = append(vertices, texturedSurfaceVertex3D(bottom, texturePoint{u: u, v: 1}, c, w, h), texturedSurfaceVertex3D(top, texturePoint{u: u, v: 0}, c, w, h))
			if segment < 20 {
				b := base + uint16(segment*2)
				indices = append(indices, b, b+1, b+2, b+1, b+3, b+2)
			}
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
	}
}

func (m *WorldMode) drawCastRingEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, componentIndex int, x, y, z, progress float64) {
	alpha := effectComponentAlpha(progress, component)
	if alpha <= 0 {
		return
	}
	texture := m.effectTexture(ctx.Resources, component.textureName)
	tint := effectComponentTint(component, alpha)
	z += component.posZ
	angleOffset := 0.0
	if component.rotate {
		angleOffset = progress * 2 * math.Pi
		angleOffset += deterministicAngle(effect, componentIndex+17) * 0.08
	}
	drawWorldSoftRing(screen, m.whitePixel, x, y, z+0.015, component.bottomSize*1.15, 0.16, withAlpha(component.color, alpha*0.75), maxInt(component.circleSides, 20))
	if texture != nil {
		drawWorldCylinderBandRotated(screen, m.whitePixel, texture, x, y, z+0.035, component.bottomSize, component.topSize, component.height, tint, maxInt(component.circleSides, component.totalCircleSides), angleOffset)
		return
	}
	drawWorldCylinderBandRotated(screen, m.whitePixel, nil, x, y, z+0.035, component.bottomSize, component.topSize, component.height, tint, maxInt(component.circleSides, component.totalCircleSides), angleOffset)
}

const groundSampleFallbackRotationRadiansPerSecond = 40 * math.Pi / 180

func (m *WorldMode) drawGroundPlaneEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, worldX, worldY, progress float64, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil || ctx.World == nil {
		return
	}
	alpha := effectBillboardAlpha(progress, component)
	if alpha <= 0 {
		return
	}
	size := component.sizeStart
	if effect.size > 0 {
		size = effect.size
	}
	if size <= 0 {
		size = 1
	}
	half := size * 0.5
	angle := groundSampleRotationAngle(effect, now)
	uv := func(u, v float64) texturePoint {
		sinA, cosA := math.Sin(angle), math.Cos(angle)
		x, y := u-0.5, v-0.5
		return texturePoint{
			u: float32(x*cosA - y*sinA + 0.5),
			v: float32(x*sinA + y*cosA + 0.5),
		}
	}
	point := func(dx, dy float64) modelPoint3 {
		x := worldX + dx
		y := worldY + dy
		return modelPoint3{x: x, y: terrainHeightAt(ctx.World, x-0.5, y-0.5) + component.posZ, z: y}
	}
	verts := [4]modelPoint3{
		point(-half, -half),
		point(half, -half),
		point(half, half),
		point(-half, half),
	}
	uvs := [4]texturePoint{
		uv(0, 0),
		uv(1, 0),
		uv(1, 1),
		uv(0, 1),
	}
	tint := effectComponentTint(component, alpha)
	drawTexturedSurface3DWithOptions(screen, texture, verts, uvs, quadIndices012023, [4]color.RGBA{tint, tint, tint, tint}, groundSampleDrawOptions())
}

func groundSampleDrawOptions() *render.DrawTrianglesOptions {
	return triangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
}

func groundSampleRotationAngle(effect worldEffect, now time.Time) float64 {
	elapsed := math.Max(0, now.Sub(effect.starts).Seconds())
	speed := effect.groundSampleRotationRadiansPerSecond
	if speed <= 0 {
		speed = groundSampleFallbackRotationRadiansPerSecond
	}
	return elapsed * speed
}

func (m *WorldMode) drawLockOnTargetEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, worldX, worldY, worldZ float64, progress float64, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	alpha := effectComponentAlpha(progress, component)
	if alpha <= 0 {
		return
	}
	size := lockOnTargetSize(effect.starts, now)
	half := size * 0.5
	angle := float64(now.UnixMilli())*math.Pi/720 + math.Pi/4
	sinA, cosA := math.Sin(angle), math.Cos(angle)
	point := func(dx, dy float64) modelPoint3 {
		x := dx*cosA - dy*sinA
		y := dx*sinA + dy*cosA
		return modelPoint3{x: worldX + x, y: worldZ + component.posZ, z: worldY + y}
	}
	verts := [4]modelPoint3{
		point(-half, -half),
		point(half, -half),
		point(half, half),
		point(-half, half),
	}
	uvs := [4]texturePoint{
		{u: 0, v: 0},
		{u: 1, v: 0},
		{u: 1, v: 1},
		{u: 0, v: 1},
	}
	tint := lockOnTargetTint(effect.starts, now)
	tint.A = uint8(clampFloat(alpha, 0, 1) * 255)
	options := triangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
	drawTexturedSurface3DWithOptions(screen, texture, verts, uvs, quadIndices012023, [4]color.RGBA{tint, tint, tint, tint}, options)
}

func lockOnTargetSize(starts, now time.Time) float64 {
	elapsed := now.Sub(starts).Seconds() * 1000 / 50
	elapsed = clampFloat(elapsed, 1, 5)
	return (6 - elapsed) * 3
}

func lockOnTargetTint(starts, now time.Time) color.RGBA {
	elapsed := int(now.Sub(starts) / (20 * time.Millisecond))
	if elapsed < 0 {
		elapsed = 0
	}
	factor := float64(20-(elapsed%20)) / 20
	gb := uint8(clampFloat(factor, 0, 1) * 255)
	return color.RGBA{R: 255, G: gb, B: gb, A: 255}
}

// VOLCANO / PrimGI2: four expanding rings at the selected ground cell.
func (m *WorldMode) drawGroundCastEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
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
