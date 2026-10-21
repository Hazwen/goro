package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) drawTurnUndeadEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	if now.Before(effect.starts) {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if frame < 20 {
		m.drawTurnUndeadFlash(screen, ctx, projection, component, effect, x, y, z, frame)
	}
	age := frame - 50
	if age < 0 || age >= 30 {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFiles[1])
	if texture == nil {
		return
	}
	// Prim3DTexture accelerates for 15 ticks, then eases to a stop.
	n := float64(min(age+1, 15))
	size := 1.5*n + .15*n*(n+1)/2
	n = float64(max(age-14, 0))
	size += .5*n - (.5/15)*n*(n+1)/2
	size /= 5
	center := modelPoint3{x: x, y: z + 7.0/5, z: y}
	vertices := make([]render.Vertex3D, 0, 4)
	indices := make([]uint16, 0, 6)
	appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{
		add3(center, modelPoint3{x: -size, z: -size}), add3(center, modelPoint3{x: size, z: -size}),
		add3(center, modelPoint3{x: size, z: size}), add3(center, modelPoint3{x: -size, z: size}),
	}, color.RGBA{255, 255, 255, effectParticleAlpha(age, 30, 1, 20, 150)})
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(false, false))
}

func (m *WorldMode) drawTurnUndeadFlash(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, x, y, z float64, frame int) {
	center, ok := projection.projectPoint(x, y, z)
	if !ok {
		return
	}
	center.y -= 40
	vertex := func(angle, radius float64, u, v, alpha float32) render.Vertex {
		return render.Vertex{
			DstX: center.x + float32(math.Cos(angle)*radius),
			DstY: center.y - float32(math.Sin(angle)*radius),
			SrcX: u, SrcY: v, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: alpha,
		}
	}
	options := &render.DrawTrianglesOptions{Filter: render.FilterLinear, Address: render.AddressRepeat}
	n := float64(frame + 1)
	// The native circle and five collapsing rays are fixed-size screen overlays.
	if texture := m.effectFileTexture(ctx.Resources, component.textureFile); texture != nil {
		w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
		alpha := float32(effectParticleAlpha(frame, 20, 6, 10, 130)) / 255
		radius := 370 - 18.5*n
		vertices := make([]render.Vertex, 0, 30)
		indices := make([]uint16, 0, 30)
		for i := 0; i < 10; i++ {
			u := float32(i%4) * w / 4
			base := uint16(len(vertices))
			vertices = append(vertices, vertex(0, 0, u, h, alpha),
				vertex(float64(i+1)*math.Pi/5, radius, u+w/4, 0, alpha),
				vertex(float64(i)*math.Pi/5, radius, u, 0, alpha))
			indices = append(indices, base, base+1, base+2)
		}
		screen.DrawTrianglesOwned(vertices, indices, texture, options)
	}
	if texture := m.effectFileTexture(ctx.Resources, component.textureFiles[0]); texture != nil {
		w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
		alpha := float32(effectParticleAlpha(frame, 20, 10, 14, 150)) / 255
		vertices := make([]render.Vertex, 0, 15)
		indices := make([]uint16, 0, 15)
		for i := 0; i < 5; i++ {
			speed := float64(10+hitEffectRandom(effect, i*5+1, 60)) / 10
			angle := float64(hitEffectRandom(effect, i*5, 360)) + speed*n - speed/30*n*(n+1)/2
			halfAngle := float64(5+hitEffectRandom(effect, i*5+2, 25)) / 10
			radius := float64(350+hitEffectRandom(effect, i*5+3, 30)) - float64(150+hitEffectRandom(effect, i*5+4, 30))/10*n
			base := uint16(len(vertices))
			vertices = append(vertices, vertex(0, 0, w/2, 0, alpha),
				vertex(degreesToRadians(float64(int(angle-halfAngle))), radius, w, h, alpha),
				vertex(degreesToRadians(float64(int(angle+halfAngle))), radius, 0, h, alpha))
			indices = append(indices, base, base+1, base+2)
		}
		screen.DrawTrianglesOwned(vertices, indices, texture, options)
	}
}

func (m *WorldMode) drawReviveEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	if now.Before(effect.starts) {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if frame >= 250 {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	for birth := 0; birth <= frame; birth += 25 {
		age := frame - birth
		life, fade, top, delta := 60, 30, 10.5, 0.0
		n := float64(age + 1)
		if birth == 0 {
			life, fade, top = 250, 225, 11
		} else {
			delta = -.12*n + .003*n*(n+1)/2
		}
		if age >= life {
			continue
		}
		alpha := float64(effectParticleAlpha(age, life, 1, fade, 240))
		if birth+life > 250 {
			// Late rings start fading immediately as their parent ends.
			alpha = 240 * max(0, 1-n/float64(249-birth))
			if age >= fade {
				alpha = 240 * (1 - float64(fade)/float64(249-birth)) * float64(life-age-1) / float64(life-fade)
			}
		}
		// Prim3DCylinder increments longitude twice, building its matrix between
		// those increments. Its ten faces repeat the texture every four faces.
		angle := degreesToRadians(float64(int(3*n - 1.5 + .03*n*n)))
		drawWorldCylinderBandWithOptions(screen, nil, texture, x, y, z,
			(7.5+delta)/5, (top+delta)/5, 7.5/5, color.RGBA{255, 255, 255, uint8(max(0, alpha))}, 10,
			rotateModelPointY(modelPoint3{x: 1}, angle), rotateModelPointY(modelPoint3{z: 1}, angle), modelPoint3{y: 1}, 2.5, render.BlendSourceOver)
	}
}
