package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func stealParticleAt(effect worldEffect, index, frame int) effectParticle {
	if frame < 0 || frame >= 50 {
		return effectParticle{}
	}
	salt := index * 6
	yaw := degreesToRadians(float64(hitEffectRandom(effect, salt, 360)))
	pitch := degreesToRadians(float64(hitEffectRandom(effect, salt+1, 100) - 50))
	radius := float64(4+hitEffectRandom(effect, salt+2, 4)) / 5
	position := rotateModelPointX(rotateModelPointY(modelPoint3{z: radius}, yaw), pitch)
	position.y = 2 - position.y
	n := float64(frame + 1)
	speed := float64(50+hitEffectRandom(effect, salt+3, 50)) / 500
	gravity := float64(60+hitEffectRandom(effect, salt+4, 90)) / 500
	distance := speed * (n - n*(n+1)/200)
	position.x += math.Sin(yaw) * distance
	position.z += math.Cos(yaw) * distance
	position.y += gravity * (n - n*(n+1)/50)
	size := float64(100+hitEffectRandom(effect, salt+5, 100)) / 100 * (1 - n/50)
	return effectParticle{position: position, size: size, alpha: math.Min(1, (50-n)/16) * 254 / 255}
}

func (m *WorldMode) drawStealEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	for i := 0; i < 10; i++ {
		particle := stealParticleAt(effect, i, frame)
		m.drawEffectParticleSprite(screen, ctx, projection, component.spriteFile, add3(effect.actorOrigin, particle.position),
			frame/4+1, particle.size, 0, color.RGBA{255, 255, 255, uint8(particle.alpha * 255)})
	}
}

func sprinkleSandParticleAt(effect worldEffect, index, frame int) effectParticle {
	salt := index * 7
	n := frame + 1 - (10 - hitEffectRandom(effect, salt, 10))
	alpha := 15*clampInt(n-1, 0, 9) - 3*max(0, n-10)
	if n <= 0 || alpha <= 0 {
		return effectParticle{}
	}
	yaw := degreesToRadians(float64(hitEffectRandom(effect, salt+1, 360)))
	speed := float64(hitEffectRandom(effect, salt+2, 10)) / 500
	rise := float64(20+hitEffectRandom(effect, salt+3, 40)) / 500
	return effectParticle{
		position: modelPoint3{
			x: float64(hitEffectRandom(effect, salt+4, 9)-4)/5 + float64(n)*speed*math.Cos(yaw),
			y: float64(n) * rise,
			z: float64(hitEffectRandom(effect, salt+5, 9)-4)/5 + float64(n)*speed*math.Sin(yaw),
		},
		size:  (.1 + .03*float64(hitEffectRandom(effect, salt+6, 16))) / 5 * math.Sqrt2,
		angle: degreesToRadians(float64(-5*n)) - 3*math.Pi/4,
		alpha: float64(alpha) / 255,
	}
}

func (m *WorldMode) drawSprinkleSandEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	// PARTICLE_UP launches twenty groups of four gold particles on the victim.
	for i := 0; i < 80; i++ {
		particle := sprinkleSandParticleAt(effect, i, frame)
		if particle.alpha <= 0 {
			continue
		}
		p := add3(effect.actorOrigin, particle.position)
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture, p.x, p.z, p.y,
			particle.size, particle.size, particle.angle, color.RGBA{250, 250, 150, uint8(particle.alpha * 255)}, true)
	}
}
