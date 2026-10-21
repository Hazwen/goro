package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func monkRay(effect worldEffect, index, frame int) ([4]modelPoint3, color.RGBA) {
	salt := index * 3
	age := frame + 1 - 10 - hitEffectRandom(effect, salt, 25)
	alpha := 15*clampInt(age-10, 0, 10) - 5*max(0, age-20)
	tint := color.RGBA{225, 225, 80, 0}
	if effect.effectID == effectTeiHit1X {
		tint = color.RGBA{255, 255, 255, 0}
	}
	var corners [4]modelPoint3
	if alpha <= 0 {
		return corners, tint
	}
	tint.A = uint8(alpha)
	pitch := degreesToRadians(float64(hitEffectRandom(effect, salt+1, 360)))
	yaw := degreesToRadians(float64(hitEffectRandom(effect, salt+2, 360)))
	distance := float64(-10 + age*3)
	// TEIHIT1 uses a .7-unit half-width and a length multiplier of 24.
	for i, p := range [4][2]float64{{16.8, -.7}, {16.8, .7}, {-16.8, .7}, {-16.8, -.7}} {
		corners[i] = add3(effect.actorOrigin, effectParticlePoint(distance+p[0], p[1], pitch, yaw))
		corners[i].y += 1.8
	}
	return corners, tint
}

func (m *WorldMode) drawMonkRays(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	var vertices []render.Vertex3D
	var indices []uint16
	for i := 0; i < component.duplicate; i++ {
		corners, tint := monkRay(effect, i, frame)
		if tint.A > 0 {
			appendEffectQuad(&vertices, &indices, texture,
				[4]modelPoint3{corners[3], corners[2], corners[1], corners[0]}, tint)
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
	}
}

func initTripleAttack(ctx client.Context, effect *worldEffect) {
	destination := effectWorldAnchor(ctx, *effect, effect.starts)
	if _, ok, _ := actorForCombatID(ctx, effect.targetID); effect.targetID != 0 && ok {
		effect.actorID, effect.targetID = effect.targetID, effect.actorID
		effect.x, effect.y, _ = effectAnchor(ctx, effect.actorID)
	}
	effect.actorOrigin = effectWorldAnchor(ctx, *effect, effect.starts)
	effect.projectileDirection = sub3(destination, effect.actorOrigin)
	effect.hasActorTransform = true
}

func tripleAttackLine(effect worldEffect, index, frame int) (from, to modelPoint3, alpha int) {
	salt := index * 6
	age := frame + 1 - index/4 - hitEffectRandom(effect, salt, 20)
	alpha = max(0, 8*clampInt(age, 0, 5)-4*max(0, age-5))
	angle := math.Atan2(effect.projectileDirection.z, effect.projectileDirection.x)
	s, c := math.Sincos(angle)
	back := .15 * float64(hitEffectRandom(effect, salt+1, 100))
	from = add3(effect.actorOrigin, modelPoint3{
		x: (.06*float64(hitEffectRandom(effect, salt+2, 100)) - 3 - back*c) / 5,
		z: (.06*float64(hitEffectRandom(effect, salt+3, 100)) - 3 - back*s) / 5,
		y: (8 + .04*float64(hitEffectRandom(effect, salt+4, 100))) / 5,
	})
	length := (20 + .2*float64(hitEffectRandom(effect, salt+5, 100))) / 5
	to = add3(from, modelPoint3{x: length * c, z: length * s})
	return
}

func (m *WorldMode) drawTripleAttack(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	for layer := 0; layer < 2; layer++ {
		var vertices []render.Vertex3D
		var indices []uint16
		for i := 0; i < 8; i++ {
			from, to, alpha := tripleAttackLine(effect, i, frame)
			if alpha == 0 {
				continue
			}
			tint := color.RGBA{255, 255, uint8(55 + 145*layer), uint8(alpha)}
			appendMonkTrail(&vertices, &indices, texture, from, to, float64(layer+1)*.05, 5, tint)
		}
		if len(indices) > 0 {
			screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(layer == 1, false))
		}
	}
}

// HITLINE keeps twenty samples of a bouncing ray, reducing each bounce by 30%.
func hitLinePoint(age int, speed, direction float64) modelPoint3 {
	if age <= 0 {
		return modelPoint3{}
	}
	phase := age * 8
	bounces := (phase - 1) / 180
	phase -= bounces * 180
	radius := math.Pow(.7, float64(bounces))
	distance := speed * (2*(1-radius)/.3 + float64(phase)/90*radius) / 5
	return modelPoint3{
		x: distance * math.Cos(direction), z: distance * math.Sin(direction),
		y: speed * math.Sin(degreesToRadians(float64(phase))) * 1.5 * radius / 5,
	}
}

func (m *WorldMode) drawHitLine(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	var vertices []render.Vertex3D
	var indices []uint16
	for i := 0; i < 16; i++ {
		salt := i * 4
		age := frame + 1 - i/4 - hitEffectRandom(effect, salt, 26)
		alpha := 125 + hitEffectRandom(effect, salt+1, 51) - 5*max(0, age-35)
		if age <= 0 || alpha <= 0 {
			continue
		}
		direction := degreesToRadians(float64(hitEffectRandom(effect, salt+2, 360)))
		speed := float64(6 + hitEffectRandom(effect, salt+3, 8))
		for j := 0; j < 20 && alpha-j*5 > 0; j++ {
			// The native shift copies the new head into slot 1 as well.
			from := add3(effect.actorOrigin, hitLinePoint(age-max(0, j-1), speed, direction))
			to := add3(effect.actorOrigin, hitLinePoint(age-j, speed, direction))
			tint := color.RGBA{255, 200, uint8(10 + j*7), uint8(alpha - j*5)}
			if effect.effectID == effectHitLine2 {
				tint.G, tint.B = uint8(60+j*7), uint8(60+j*7)
			}
			appendMonkTrail(&vertices, &indices, texture, from, to, .02, 3, tint)
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
	}
}

func appendMonkTrail(vertices *[]render.Vertex3D, indices *[]uint16, texture *render.Image, from, to modelPoint3, radius float64, sides int, tint color.RGBA) {
	delta := sub3(to, from)
	if delta.x == 0 && delta.z == 0 {
		return
	}
	direction := math.Atan2(delta.z, delta.x) + math.Pi/2
	var previous modelPoint3
	for i := 0; i <= sides; i++ {
		s, c := math.Sincos(float64(i) * 2 * math.Pi / float64(sides))
		offset := modelPoint3{x: radius * c * math.Cos(direction), z: radius * c * math.Sin(direction), y: -radius * s}
		if i > 0 {
			appendEffectQuad(vertices, indices, texture, [4]modelPoint3{
				add3(to, previous), add3(to, offset), add3(from, offset), add3(from, previous),
			}, tint)
		}
		previous = offset
	}
}

type asuraGlyphs struct {
	origins [2]modelPoint3
	count   int
}

func (m *WorldMode) drawAsuraGlyphs(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	mildWind := effect.effectID >= effectBeginAsura1 && effect.effectID <= effectBeginAsura7
	if !mildWind && effect.asuraGlyphs == nil {
		return
	}
	for !mildWind && effect.asuraGlyphs.count < 2 {
		i := effect.asuraGlyphs.count
		at := effect.starts.Add(time.Duration(1+20*i) * db.EffectFrameDuration)
		if now.Before(at) {
			break
		}
		effect.asuraGlyphs.origins[i] = effectWorldAnchor(ctx, effect, at)
		effect.asuraGlyphs.count++
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	champion := effect.effectID == effectBeginAsura11
	spacing, radiusBase, shade := 12.0, 18.0, uint8(0)
	if champion {
		spacing, radiusBase, shade = 16, 24, 10
	}
	origin := effect.actorOrigin
	if mildWind {
		origin = effectWorldAnchor(ctx, effect, now)
		shade = 255
	}
	right, _, _, ok := projection.BillboardBasis(origin.x, origin.z, origin.y)
	if !ok {
		return
	}
	for i, file := range component.textureFiles {
		// The left and right primitives start at ticks 1 and 21; their
		// internal delays make the six characters appear 20 ticks apart.
		age := frame - 20*i
		if mildWind {
			age = frame + 1
		}
		if age <= 0 {
			continue
		}
		texture := m.effectFileTexture(ctx.Resources, file)
		position := origin
		if !mildWind {
			origin = effect.asuraGlyphs.origins[i/3]
			position = add3(origin, mul3(right, (float64(i)-2.5)*spacing/5))
		}
		position.y += 6
		radius := (radiusBase - .1*float64(min(age, 49))) * math.Pow(1.01, float64(clampInt(age-140, 0, 49)))
		for layer := 10; layer >= 1; layer-- {
			alpha := min(13*min(age, 19), 50+layer*20)
			if layer < 10 {
				alpha -= 5 * max(0, age-19)
			} else {
				alpha -= 5 * max(0, age-140)
			}
			if alpha <= 0 {
				continue
			}
			size := math.Sqrt2 * (radius + float64(10-layer)*3) / 5
			tint := color.RGBA{shade, shade, shade, uint8(alpha)}
			if champion && layer == 10 {
				tint.A = uint8(alpha / 2)
				drawTexturedEffectBillboardRotatedXY(screen, projection, texture, position.x, position.z, position.y, size, size, 0, tint, false)
				tint.A = uint8(alpha)
			}
			drawTexturedEffectBillboardRotatedXY(screen, projection, texture, position.x, position.z, position.y, size, size, 0, tint, false)
		}
	}
}
