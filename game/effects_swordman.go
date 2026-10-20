package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func bashFlash(effect worldEffect, index, frame int) (angle, halfAngle, radius float64) {
	n := float64(frame + 1)
	speed := float64(10+hitEffectRandom(effect, index*5+1, 60)) / 10
	angle = float64(hitEffectRandom(effect, index*5, 360)) + speed*n - speed/60*n*(n+1)/2
	halfAngle = float64(5+hitEffectRandom(effect, index*5+2, 25)) / 10
	radius = float64(20+hitEffectRandom(effect, index*5+3, 40)) + float64(20+hitEffectRandom(effect, index*5+4, 30))/10*n
	return
}

func (m *WorldMode) drawBashEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	center, ok := projection.projectPoint(x, y, z)
	if !ok {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	center.y -= 40
	options := &render.DrawTrianglesOptions{Filter: render.FilterLinear, Address: render.AddressRepeat}
	vertex := func(angle, radius float64, u, v, alpha float32) render.Vertex {
		return render.Vertex{
			DstX: center.x + float32(math.Cos(angle)*radius),
			DstY: center.y - float32(math.Sin(angle)*radius),
			SrcX: u, SrcY: v, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: alpha,
		}
	}
	// Native PP_2D primitives overlay actors and geometry at a fixed pixel size.
	if texture := m.effectFileTexture(ctx.Resources, component.textureFile); texture != nil {
		vertices := make([]render.Vertex, 0, 30)
		indices := make([]uint16, 0, 30)
		w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
		alpha := float32(effectParticleAlpha(frame, 40, 6, 30, 170)) / 255
		for i := 0; i < 10; i++ {
			u := float32(i%4) * w / 4
			base := uint16(len(vertices))
			vertices = append(vertices,
				vertex(0, 0, u, h, alpha),
				vertex(float64(i+1)*math.Pi/5, 100, u+w/4, 0, alpha),
				vertex(float64(i)*math.Pi/5, 100, u, 0, alpha))
			indices = append(indices, base, base+1, base+2)
		}
		screen.DrawTrianglesOwned(vertices, indices, texture, options)
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFiles[0])
	if texture == nil {
		return
	}
	vertices := make([]render.Vertex, 0, 60)
	indices := make([]uint16, 0, 60)
	w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
	alpha := float32(effectParticleAlpha(frame, 40, 10, 27, 200)) / 255
	for i := 0; i < 20; i++ {
		angle, halfAngle, radius := bashFlash(effect, i, frame)
		base := uint16(len(vertices))
		// The native renderer indexes integer-degree sine/cosine tables.
		vertices = append(vertices,
			vertex(0, 0, w/2, 0, alpha),
			vertex(degreesToRadians(float64(int(angle-halfAngle))), radius, w, h, alpha),
			vertex(degreesToRadians(float64(int(angle+halfAngle))), radius, 0, h, alpha))
		indices = append(indices, base, base+1, base+2)
	}
	screen.DrawTrianglesOwned(vertices, indices, texture, options)
}

func bashHitParticle(effect worldEffect, index, frame int) effectScreenParticle {
	life := 10 + hitEffectRandom(effect, index*6, 20)
	if frame < 0 || frame >= life {
		return effectScreenParticle{}
	}
	n := float64(frame + 1)
	axis := degreesToRadians(float64(index * 45))
	angle := degreesToRadians(float64(index*45 - 15 + hitEffectRandom(effect, index*6+1, 30)))
	speed := float64(5+hitEffectRandom(effect, index*6+2, 45)) / 10
	distance := speed*n - speed/float64(2*life)*n*(n+1)/2
	halfWidth := float64(5+hitEffectRandom(effect, index*6+3, 15)) * (1 - n/float64(life))
	halfHeight := float64(20+hitEffectRandom(effect, index*6+4, 20)) + 1.5*n + .25*n*(n+1)/2
	radius := float64(hitEffectRandom(effect, index*6+5, 5))
	// SIZEUP anchors the bottom edge, so shift the centered quad outward.
	return effectScreenParticle{
		x:         math.Sin(axis)*radius + math.Sin(angle)*(distance+halfHeight),
		y:         -math.Cos(axis)*radius - math.Cos(angle)*(distance+halfHeight) - 20,
		halfWidth: halfWidth, halfHeight: halfHeight, angle: angle,
		alpha: uint8(254 * math.Min(1, n/8)),
	}
}

func (m *WorldMode) drawBashHitEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	origin, ok := projection.projectPoint(x, y, z)
	if !ok {
		return
	}
	textures := [2]*render.Image{m.effectFileTexture(ctx.Resources, component.textureFile), m.effectFileTexture(ctx.Resources, component.textureFiles[0])}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	for i := 0; i < 8; i++ {
		drawEffectScreenParticle(screen, textures[i%2], origin, bashHitParticle(effect, i, frame))
	}
}

func endureStreak(effect worldEffect, birth, frame int) effectScreenParticle {
	age := frame - birth
	// The parent increments its tick before ProcessAlpha runs. A new primitive
	// with more remaining life is immediately faded from zero and stays invisible.
	if birth < 0 || birth >= 50 || 40 > 80-birth || age < 0 || age >= 40 || frame >= 80 {
		return effectScreenParticle{}
	}
	n := float64(age + 1)
	angle := degreesToRadians(float64(hitEffectRandom(effect, birth*4, 360)))
	radius := float64(100+hitEffectRandom(effect, birth*4+3, 40)) - 4*n + .1*n*(n+1)/2
	return effectScreenParticle{
		x: math.Sin(angle) * radius, y: -math.Cos(angle)*radius - 20,
		halfWidth:  float64(60+hitEffectRandom(effect, birth*4+1, 100)) / 100,
		halfHeight: float64(30 + hitEffectRandom(effect, birth*4+2, 20)),
		angle:      angle,
		alpha:      effectParticleAlpha(age, 40, 20, 27, 254),
	}
}

func (m *WorldMode) drawEndureEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	origin, ok := projection.projectPoint(x, y, z)
	if !ok {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if texture := m.effectFileTexture(ctx.Resources, component.textureFile); texture != nil {
		n := float64(min(frame+1, 45))
		halfSize := 150 - 5*n + (5.0/45)*n*(n+1)/2
		drawEffectScreenParticle(screen, texture, origin, effectScreenParticle{
			y: -20, halfWidth: halfSize, halfHeight: halfSize,
			alpha: effectParticleAlpha(frame, 80, 20, 54, 254),
		})
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFiles[0])
	if texture == nil {
		return
	}
	for birth := max(0, frame-39); birth <= min(frame, 49); birth++ {
		drawEffectScreenParticle(screen, texture, origin, endureStreak(effect, birth, frame))
	}
}

func magnumRadius(speed float64, frame int) float64 {
	n := float64(frame + 1)
	return (speed*n - speed/60*n*(n+1)/2) / 5
}

func magnumSphereMesh(texture *render.Image, center modelPoint3, frame int, alpha uint8) ([]render.Vertex3D, []uint16) {
	vertices := make([]render.Vertex3D, 0, 200)
	indices := make([]uint16, 0, 300)
	w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
	radius := magnumRadius(1.15, frame)
	tint := color.RGBA{255, 255, 255, alpha}
	point := func(latitude, longitude float64) modelPoint3 {
		pitch := degreesToRadians(latitude)
		yaw := degreesToRadians(longitude + 3*float64(frame+1))
		return add3(center, modelPoint3{x: radius * math.Cos(pitch) * math.Sin(yaw), y: radius * math.Sin(pitch), z: radius * math.Cos(pitch) * math.Cos(yaw)})
	}
	for row := 0; row < 5; row++ {
		for col := 0; col < 10; col++ {
			latitude, longitude := float64(-90+row*36), float64(col*36)
			// GetRadian takes integer degrees; the source's extra .25 is truncated.
			corners := [4]modelPoint3{
				point(latitude, longitude), point(latitude+36, longitude),
				point(latitude+36, longitude+36), point(latitude, longitude+36),
			}
			u, v := float32(col)/10, float32(row)/5
			base := uint16(len(vertices))
			for i, uv := range [4]texturePoint{{u: u, v: v}, {u: u, v: v + .2}, {u: u + .1, v: v + .2}, {u: u + .1, v: v}} {
				vertices = append(vertices, texturedSurfaceVertex3D(corners[i], uv, tint, w, h))
			}
			indices = append(indices, base, base+1, base+3, base+2, base+3, base+1)
		}
	}
	return vertices, indices
}

func (m *WorldMode) drawMagnumBreakEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	alpha := effectParticleAlpha(frame, 30, 15, 15, 254)
	if alpha == 0 {
		return
	}
	if texture := m.effectFileTexture(ctx.Resources, component.textureFiles[0]); texture != nil {
		vertices, indices := magnumSphereMesh(texture, effect.actorOrigin, frame, alpha)
		screen.DrawTriangles3DOwned(vertices, indices, texture, triangleDrawOptions(render.FilterLinear, render.AddressRepeat))
	}
	if texture := m.effectFileTexture(ctx.Resources, component.textureFile); texture != nil {
		radius := magnumRadius(1.75, frame)
		vertices, indices := groundEffectRingMesh(texture, effect.actorOrigin, math.Max(0, radius-2.4), radius, alpha)
		options := triangleDrawOptions(render.FilterLinear, render.AddressRepeat)
		options.DepthTest = false
		// The 2008 client deliberately launches two identical ground rings.
		screen.DrawTriangles3DOwned(vertices, indices, texture, options)
		screen.DrawTriangles3DOwned(vertices, indices, texture, options)
	}
}
