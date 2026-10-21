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

func (m *WorldMode) drawPriestGroundEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
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
		appendPriestSquareBand(&vertices, &indices, texture, center, half/5, height/5, 0, 8*half/5, tint)
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

func appendBasilicaWalls(vertices *[]render.Vertex3D, indices *[]uint16, texture *render.Image, center modelPoint3, tick int) {
	// Two concentric walls and two moving strips per layer (PP_RECT_UP2).
	for layer := 0; layer < 2; layer++ {
		for i := 0; i < 4; i++ {
			maxHeight, half, fade := 30.0+float64(i), 12.5, 2415
			tint := color.RGBA{200, 250, 200, 65}
			if layer == 1 {
				maxHeight, half, fade = 20+float64(i), 14, 2385
				tint.A = 32
			}
			half += .4 * float64(min(i, 2))
			start, length := 0.0, 8*half
			if i >= 2 {
				start = math.Mod(float64(i-2)*53.2+float64(layer)*26.6+.5*float64(tick), 8*half)
				length = 26.6
				tint = color.RGBA{150, 250, 150, 15}
				fade = 2415
			}
			angle := min(tick, 90)
			if tick > fade {
				angle = max(0, min(90, 2505-layer*30-tick))
			}
			height := maxHeight * math.Sin(degreesToRadians(float64(angle)))
			appendPriestSquareBand(vertices, indices, texture, center, half/5, height/5, start/5, length/5, tint)
		}
	}
}

// A band follows the square perimeter, splitting at corners. The native
// RenderTeiRect restarts the texture on each face, including partial faces.
func appendPriestSquareBand(vertices *[]render.Vertex3D, indices *[]uint16, texture *render.Image, center modelPoint3, half, height, start, length float64, tint color.RGBA) {
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
