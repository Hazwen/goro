package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) drawWizardWaterBall(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if now.Before(effect.starts) || frame >= 40 {
		return
	}
	for i := 11; i >= 0; i-- {
		p := soulStrikePosition(effect, max(0, frame+1-i), 0)
		m.drawEffectParticleSprite(screen, ctx, projection, component.spriteFile, p, frame/4+1,
			.75*(1-float64(i)/24), 0, color.RGBA{255, 255, 255, uint8(254 * (1 - float64(i)/12))})
	}
}

func (m *WorldMode) drawJupitel(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, origin modelPoint3, now time.Time) {
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if now.Before(effect.starts) {
		return
	}
	projectile := effect.effectID == effectJupitelThunder
	if projectile {
		if frame >= 20 {
			return
		}
		origin = add3(effect.actorOrigin, mul3(effect.projectileDirection, float64(frame+1)/20))
	}
	origin.y++
	draw := func(file string, halfSize, alpha float64) {
		texture := m.effectFileTexture(ctx.Resources, file)
		drawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, origin.x, origin.z, origin.y,
			halfSize*2/5, halfSize*2/5, 0, color.RGBA{255, 255, 255, uint8(alpha)},
			texturedEffectBillboardDrawOptions(true, !projectile))
	}
	if projectile {
		draw(component.textureFile, 3.5, 170)
		draw(component.textureFiles[(frame+1)%len(component.textureFiles)], 4.5, 254)
	} else if frame < 250 {
		age := frame % 20
		if age < 10 {
			draw(component.textureFile, 2.5*float64(age+1), 254*clampFloat(float64(9-age)/9, 0, 1))
		}
		if frame >= 10 {
			draw(component.textureFiles[(frame-10+1)%len(component.textureFiles)], 7.5, 254)
		}
	}
}

func (m *WorldMode) drawSightTrasher(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, origin modelPoint3, now time.Time) {
	if now.Before(effect.starts) {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	if frame >= 85 {
		return
	}
	for birth := 0; birth <= min(frame, 15); birth += 5 {
		n := float64(frame - birth + 1)
		change := float64(35 - birth)
		late := max(0, n-change)
		distance := 5 + float64(birth/5) + min(n, change) + .5*late
		for i := 0; i < 8; i++ {
			s, c := math.Sincos(degreesToRadians(float64(-45 * i)))
			position := add3(origin, modelPoint3{
				x: (distance*s - .004*late*(late+1)/2) / 5,
				y: float64(20-birth/3) / 5,
				z: (-distance*c + .004*late*(late+1)/2) / 5,
			})
			size, speed := 2.5-float64(birth)/10, 3+hitEffectRandom(effect, birth*8+i, 2)
			if component.shadowTexture {
				size, speed = 1-float64(birth)/30, 3
				position.y = terrainHeightAtRenderPoint(ctx.World, position.x, position.z) + actorShadowTerrainLift
			}
			alpha := float64(150-2*birth) * clampFloat(float64(84-frame)/20, 0, 1)
			m.drawEffectParticleSprite(screen, ctx, projection, component.spriteFile, position, int(n-1)/speed+1, size, 0, color.RGBA{255, 255, 255, uint8(alpha)})
		}
	}
	age := frame - 10
	if !component.shadowTexture && age >= 0 && age < 30 {
		texture := m.effectFileTexture(ctx.Resources, component.textureFile)
		if texture == nil {
			return
		}
		n := float64(age + 1)
		radius := (2.5*n - (2.5/45)*n*(n+1)/2) / 5
		vertices, indices := groundEffectRingMesh(texture, origin, radius+2, radius, uint8(150*clampFloat(float64(29-age)/10, 0, 1)))
		screen.DrawTriangles3DOwned(vertices, indices, texture, triangleDrawOptions(render.FilterLinear, render.AddressRepeat))
	}
}

func (m *WorldMode) drawFirePillar(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, origin modelPoint3, now time.Time) {
	if now.Before(effect.starts) {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	frame := float64(now.Sub(effect.starts) / db.EffectFrameDuration)
	rotate := func(p modelPoint3) modelPoint3 {
		// Prim3DCylinder advances longitude twice per tick, drawing between them.
		return rotateModelPointY(p, degreesToRadians(-3*(2*frame+1)))
	}
	for i := 1; i <= 3; i++ {
		height := float64(2*(4-i)*(5-i)) + 3.5
		drawWorldCylinderBandWithOptions(screen, m.whitePixel, texture, origin.x, origin.z, origin.y,
			(float64(i)+1.5)/5, float64(i+3)/5, height/5, color.RGBA{255, 255, 255, 240}, 10,
			rotate(modelPoint3{x: 1}), rotate(modelPoint3{z: 1}), rotate(modelPoint3{y: 1}), 2.5, render.BlendSourceOver)
	}
}
