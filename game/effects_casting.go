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
			if effect.blueCasting.auraBlade {
				maxHeight = 15 - layer
			}
			height := float64(maxHeight) * (1 + .3*math.Sin(degreesToRadians(float64(segment*9)))*math.Sin(degreesToRadians(float64(phase))))
			bottom := modelPoint3{x: x + cos*r.radius/5, y: z, z: y + sin*r.radius/5}
			top := add3(bottom, modelPoint3{x: cos * riseCos * height / 5, y: riseSin * height / 5, z: sin * riseCos * height / 5})
			u := float32(segment) / 21
			c := color.RGBA{100, 100, 255, uint8(clampInt(r.alpha, 0, 255))}
			if effect.blueCasting.auraBlade {
				c.R, c.G = 255, 255
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
