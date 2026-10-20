package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) drawBlacksmithEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if frame < 0 {
		return
	}
	origin := effectWorldAnchor(ctx, effect, now)
	if effect.effectID == effectHasteUp && frame < 100 {
		n := float64(frame + 1)
		for i := 0; i < 4; i++ {
			angle := degreesToRadians(float64(i*90) + .3*n + .1*n*(n+1))
			position := add3(origin, modelPoint3{
				x: 1.4 * math.Sin(angle), z: 1.4 * math.Cos(angle),
				y: (.1*n + .001*n*(n+1)) / 5,
			})
			m.drawEffectParticleSprite(screen, ctx, projection, component.spriteFile, position, frame/4+1, 1.2, 0,
				color.RGBA{255, 255, 255, effectParticleAlpha(frame, 100, 1, 90, 254)})
		}
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	center, ok := projection.projectPoint(origin.x, origin.z, origin.y)
	if texture == nil || !ok {
		return
	}
	w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
	vertex := func(angle, radius float64, u, v, alpha float32) render.Vertex {
		return render.Vertex{
			DstX: center.x + float32(math.Cos(angle)*radius), DstY: center.y - float32(math.Sin(angle)*radius),
			SrcX: u, SrcY: v, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: alpha,
		}
	}
	var vertices []render.Vertex
	var indices []uint16
	switch effect.effectID {
	case effectHasteUp:
		if frame >= 80 {
			return
		}
		center.y -= 40
		n := float64(min(frame+1, 40))
		for i := 0; i < 20; i++ {
			salt := i * 5
			speed := float64(10+hitEffectRandom(effect, salt+1, 60)) / 10
			angle := float64(hitEffectRandom(effect, salt, 360)) + speed*n - speed/120*n*(n+1)/2
			angle += float64(max(0, frame-39)) * float64(15+hitEffectRandom(effect, salt+4, 10)) / 10
			halfAngle := float64(5+hitEffectRandom(effect, salt+2, 25)) / 10
			radius := float64(50+hitEffectRandom(effect, salt+3, 20)) / 13 * n
			base := uint16(len(vertices))
			vertices = append(vertices, vertex(0, 0, w/2, 0, 254.0/255),
				vertex(degreesToRadians(float64(int(angle-halfAngle))), radius, w, h, 254.0/255),
				vertex(degreesToRadians(float64(int(angle+halfAngle))), radius, 0, h, 254.0/255))
			indices = append(indices, base, base+1, base+2)
		}
	case effectOverthrust:
		if frame >= 40 {
			return
		}
		center.y -= 20
		n := float64(frame + 1)
		radius := 7*n - .175*n*(n+1)/2
		inner := math.Max(0, radius-10)
		alpha := float32(254 * (40 - n) / 40 / 255)
		for i := 0; i < 36; i++ {
			angle, next := degreesToRadians(float64(i*10)), degreesToRadians(float64((i+1)*10))
			u := float32(i%4) * w / 4
			base := uint16(len(vertices))
			vertices = append(vertices, vertex(angle, inner, u, h, alpha), vertex(next, inner, u+w/4, h, alpha),
				vertex(next, radius, u+w/4, 0, alpha), vertex(angle, radius, u, 0, alpha))
			indices = append(indices, base, base+2, base+3, base, base+1, base+2)
		}
	}
	if len(indices) > 0 {
		screen.DrawTrianglesOwned(vertices, indices, texture, &render.DrawTrianglesOptions{Filter: render.FilterLinear, Address: render.AddressRepeat})
	}
}
