package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) drawWizardSpikes(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, origin modelPoint3, now time.Time) {
	if now.Before(effect.starts) {
		return
	}
	if effect.effectID == effectStormGust && lessEffectsEnabled(ctx) {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	if effect.effectID == effectHyousensou {
		origin = effect.actorOrigin
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	count := 7
	switch effect.effectID {
	case effectIceWall:
		count = 3
	case effectHeavenDrive:
		count = 25
	case effectStormGust:
		count = 8
	}
	vertices := make([]render.Vertex3D, 0, count*12)
	indices := make([]uint16, 0, count*12)
	for i := 0; i < count; i++ {
		salt := i * 11
		rand := func(n, limit int) int { return hitEffectRandom(effect, salt+n, limit) }
		age, life := frame, 240
		position := origin
		pitch, yaw := float64(80+rand(0, 20)), float64(rand(1, 360))
		size, height := float64(300+rand(2, 50))/100, 18.0
		alpha := 250.0
		travel := wizardEarthSpikeTravel(age, 12, 15)
		position.y -= 2
		switch effect.effectID {
		case effectIceWall:
			life, alpha = 9999, 200
			pitch = float64(87 + rand(0, 6))
			size, height = 2.5, float64(300+rand(2, 100))/10
			position.y = origin.y - 6
			position.z -= float64(-2+2*i) * math.Cos(degreesToRadians(float64(rand(3, 360)))) / 5
			n := float64(min(age+1, 21))
			travel = 3*n - .15*n*(n+1)/2
		case effectHeavenDrive:
			life = 300 + rand(4, 30) - 10
			height = float64(1+rand(3, 60))/10 + 9
			position.x += float64(i%5 - 2)
			position.z += float64(i/5 - 2)
			travel = wizardEarthSpikeTravel(age, 11, 14)
			n := float64(max(0, age-(life-30)))
			travel -= n + .01*n*(n+1)/2
			alpha = 254
		case effectStormGust:
			age -= (i/2 + 2) * 29
			life = 215 - (i/2+2)*29
			radius, angle := float64(3+rand(3, 25))/5, degreesToRadians(float64(rand(4, 360)))
			position.x += radius * math.Sin(angle)
			position.z -= radius * math.Cos(angle)
			position.y = origin.y - 4
			size, height, pitch = float64(400+rand(2, 100))/100, 20, 100
			travel = 1.5 * float64(min(age+1, 4))
			alpha *= clampFloat(float64(life-1-age)/10, 0, 1)
		default:
			if i > 0 {
				angle := degreesToRadians(float64(rand(3, 360)))
				position.x += .7 * math.Sin(angle)
				position.z -= .7 * math.Cos(angle)
				position.y = origin.y - 4
				yaw = float64(min(360, rand(4, i*60)+(i-1)*60))
				size, height, pitch = float64(400+rand(2, 100))/100, 20, 100
				travel = 1.5 * float64(min(age+1, 3))
				alpha *= clampFloat(float64(life-1-age)/10, 0, 1)
			}
		}
		if age < 0 || (age >= life && effect.effectID != effectIceWall) {
			continue
		}
		appendEffectSpike(&vertices, &indices, texture, position, size, height, degreesToRadians(pitch), degreesToRadians(yaw), travel, color.RGBA{255, 255, 255, uint8(alpha)})
	}
	if len(indices) > 0 {
		options := texturedEffectBillboardDrawOptions(false, false)
		options.AfterBillboards = effect.effectID == effectIceWall
		screen.DrawTriangles3DOwned(vertices, indices, texture, options)
	}
}

func wizardEarthSpikeTravel(age, reverse, stop int) float64 {
	n := float64(min(age+1, reverse))
	return n + .01*n*(n+1)/2 - 1.2*float64(max(0, min(age, stop)-reverse+1))
}

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
