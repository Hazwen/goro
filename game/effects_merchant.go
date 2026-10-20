package game

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) initMammonite(ctx client.Context, effect *worldEffect) {
	if lessEffectsEnabled(ctx) {
		return
	}
	origin := effectWorldAnchor(ctx, *effect, effect.starts)
	effect.coinTrails = new([5][]modelPoint3)
	// The native Coin effect spawns five PP_3DPARTICLEGRAVITY sprites at tick 20.
	// Compute their short paths once, so bounce audio and visuals share timing.
	for i := range effect.coinTrails {
		salt := i * 80
		life := 40 + hitEffectRandom(*effect, salt, 30)
		yaw := degreesToRadians(float64(30 * hitEffectRandom(*effect, salt+1, 12)))
		axis := modelPoint3{x: math.Sin(yaw), z: math.Cos(yaw)}
		position := add3(origin, mul3(axis, float64(2+hitEffectRandom(*effect, salt+2, 2))/5))
		position.y += 2
		speed := float64(50+hitEffectRandom(*effect, salt+3, 50)) / 500
		accel := -speed / float64(life) / 3.5
		vertical := float64(50+hitEffectRandom(*effect, salt+4, 100)) / 125
		gravity := -vertical / float64(life) * 4
		trail := make([]modelPoint3, life)
		for age := range trail {
			speed = math.Max(0, speed+accel)
			vertical += gravity
			position = add3(position, mul3(axis, speed))
			position.y += vertical
			if vertical < 0 && position.y < terrainHeightAtRenderPoint(ctx.World, position.x, position.z) {
				vertical /= -1.5
				path := fmt.Sprintf("effect\\ef_coin%d.wav", 1+hitEffectRandom(*effect, salt+5+age, 3))
				m.scheduleSoundVolumeAtPosition(effect.starts.Add(time.Duration(20+age)*db.EffectFrameDuration), 1, position.x-.5, position.z-.5, path)
			}
			trail[age] = position
		}
		effect.coinTrails[i] = trail
	}
}

func (m *WorldMode) drawMammoniteCoins(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	if effect.coinTrails == nil || lessEffectsEnabled(ctx) {
		return
	}
	age := int(now.Sub(effect.starts)/db.EffectFrameDuration) - 20
	for _, trail := range effect.coinTrails {
		if age < 0 || age >= len(trail) {
			continue
		}
		alpha := effectParticleAlpha(age, len(trail), 1, len(trail)-len(trail)/3, 254)
		// ShiftSegment keeps one smaller, half-opacity afterimage.
		for segment := 1; segment >= 0; segment-- {
			if age < segment {
				continue
			}
			m.drawEffectParticleSprite(screen, ctx, projection, component.spriteFile, trail[age-segment],
				age+1, 2-.5*float64(segment), 0, color.RGBA{255, 255, 255, alpha / uint8(segment+1)})
		}
	}
}

func (m *WorldMode) drawCartRevolutionEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	for _, birth := range [...]int{7, 20} {
		age := frame - birth
		if age < 0 || age >= 20 {
			continue
		}
		n := float64(age + 1)
		radius := (n - n*(n+1)/80) / 5
		if texture := m.effectFileTexture(ctx.Resources, component.textureFile); texture != nil {
			alpha := effectParticleAlpha(age, 20, 15, 5, 180)
			vertices, indices := groundEffectRingMesh(texture, effect.actorOrigin, math.Max(0, 1.75*radius-1), 1.75*radius, alpha)
			options := triangleDrawOptions(render.FilterLinear, render.AddressRepeat)
			options.DepthTest = false
			screen.DrawTriangles3DOwned(vertices, indices, texture, options)
		}
		if texture := m.effectFileTexture(ctx.Resources, component.textureFiles[0]); texture != nil {
			alpha := effectParticleAlpha(age, 20, 7, 5, 240)
			vertices, indices := effectSphereMesh(texture, effect.actorOrigin, 1.35*radius, 3*n, alpha)
			screen.DrawTriangles3DOwned(vertices, indices, texture, triangleDrawOptions(render.FilterLinear, render.AddressRepeat))
		}
	}
}
