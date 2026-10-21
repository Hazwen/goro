package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

const effectShieldBoomerang3 = 520

type shieldChainFlight struct {
	origins [5]modelPoint3
	count   int
}

func (chain *shieldChainFlight) advance(ctx client.Context, effect worldEffect, now time.Time) {
	for chain.count < len(chain.origins) {
		at := effect.starts.Add(time.Duration(3*chain.count) * db.EffectFrameDuration)
		if now.Before(at) {
			break
		}
		chain.origins[chain.count] = effectWorldAnchor(ctx, effect, at)
		chain.count++
	}
}

func shieldChainImage(effect worldEffect, origin modelPoint3, index, process int) shieldImage {
	alpha := min(250, 25+45*process) - 10*max(0, process-14)
	direction := deterministicAngle(effect, index)
	position := origin
	position.x += .5 * float64(process-15) * math.Cos(direction)
	position.z += .5 * float64(process-15) * math.Sin(direction)
	position.y += 1.6
	return shieldImage{position: position, alpha: max(0, alpha), angle: -float64(process)*math.Pi/12 - 3*math.Pi/4}
}

func (m *WorldMode) drawShieldChainEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil || effect.shieldChain == nil {
		return
	}
	effect.shieldChain.advance(ctx, effect, now)
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	for i := 0; i < 5; i++ {
		process := frame - 3*i + 1
		if process <= 0 {
			continue
		}
		origin := effect.shieldChain.origins[i]
		drawShieldImage(screen, projection, texture, shieldChainImage(effect, origin, i, process), math.Sqrt2, false)
		for trail := 0; trail < 3; trail++ {
			previous := process/3*3 - 1 - 3*trail
			if previous < 2 {
				continue
			}
			particle := shieldChainImage(effect, origin, i, previous)
			particle.alpha -= 150 + 25*trail
			drawShieldImage(screen, projection, texture, particle, math.Sqrt2, false)
		}
	}
}

func pressureImage(effect worldEffect, process int) shieldImage {
	position := effect.actorOrigin
	position.y += 23 - .6*float64(min(process, 32))
	return shieldImage{
		position: position,
		alpha:    max(0, min(process, 10)*20-max(0, process-32)*5),
		angle:    deterministicAngle(effect, 0) - float64(min(process, 32))*math.Pi/12,
	}
}

func (m *WorldMode) drawPressureEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	process := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	for i := 0; i < 4; i++ {
		if process-i <= 0 {
			continue
		}
		particle := pressureImage(effect, process-i)
		if i > 0 {
			particle.alpha -= 100 + 25*(i-1)
		}
		drawShieldImage(screen, projection, texture, particle, 12*math.Sqrt2/5, true)
	}
	age := process - 28
	alpha := min(age, 10)*2 - max(0, age-10)
	if age > 0 && alpha > 0 {
		if ground := m.effectFileTexture(ctx.Resources, component.textureFiles[0]); ground != nil {
			origin := effectWorldAnchor(ctx, effect, now)
			origin.y++
			radius := 2.7 * math.Pow(1.07, float64(age))
			vertices, indices := []render.Vertex3D{}, []uint16{}
			appendEffectQuad(&vertices, &indices, ground, [4]modelPoint3{
				add3(origin, modelPoint3{x: radius}), add3(origin, modelPoint3{z: radius}),
				add3(origin, modelPoint3{x: -radius}), add3(origin, modelPoint3{z: -radius}),
			}, color.RGBA{255, 255, 255, uint8(alpha)})
			screen.DrawTriangles3DOwned(vertices, indices, ground, texturedEffectBillboardDrawOptions(true, false))
		}
	}
}

func (m *WorldMode) drawGospelCrossEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	n := float64(now.Sub(effect.starts)/db.EffectFrameDuration + 1)
	origin := effectWorldAnchor(ctx, effect, now)
	origin.y += 1.2 - .6*math.Sin(deterministicAngle(effect, effect.x*31+effect.y)+degreesToRadians(n))
	size := 1.5 * math.Sqrt2 / 5 * (1 + .05*math.Sin(degreesToRadians(n*5)))
	drawShieldImage(screen, projection, texture, shieldImage{position: origin, alpha: 200}, size, true)
}
