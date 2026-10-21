package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func sharpShootingLine(effect worldEffect, index, frame int) (from, to modelPoint3, alpha int) {
	salt := index * 6
	age := frame + 1 - index/4 - 45 - hitEffectRandom(effect, salt, 20)
	alpha = max(0, 2*clampInt(age, 0, 5)-max(0, age-5))
	angle := math.Atan2(effect.projectileDirection.z, effect.projectileDirection.x)
	s, c := math.Sincos(angle)
	back := .15 * float64(hitEffectRandom(effect, salt+1, 100))
	from = add3(effect.actorOrigin, modelPoint3{
		x: (.08*float64(hitEffectRandom(effect, salt+2, 100)) - 4 - back*c) / 5,
		z: (.08*float64(hitEffectRandom(effect, salt+3, 100)) - 4 - back*s) / 5,
		y: (8 + .04*float64(hitEffectRandom(effect, salt+4, 100))) / 5,
	})
	length := math.Hypot(effect.projectileDirection.x, effect.projectileDirection.z) + .12*float64(hitEffectRandom(effect, salt+5, 100))
	to = add3(from, modelPoint3{x: length * c, z: length * s})
	return
}

func (m *WorldMode) drawSniperEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	n := frame + 1
	var vertices []render.Vertex3D
	var indices []uint16
	switch effect.effectID {
	case effectTrueSight:
		alpha := 6*min(n, 10) - max(0, n-11)
		if alpha <= 0 {
			return
		}
		origin := effectWorldAnchor(ctx, effect, now)
		origin.y += 2.4
		distance := float64(2+3*n) / 5
		for layer, width := range []float64{2, 5} {
			vertices, indices = nil, nil
			for i := 0; i < 48; i++ {
				yaw := degreesToRadians(float64(90*(i%4) + 7*(i/4)))
				roll := deterministicAngle(effect, i)
				point := func(angle float64) modelPoint3 {
					r := distance * math.Cos(angle)
					return add3(origin, modelPoint3{x: r * math.Cos(roll), y: -r * math.Sin(roll), z: distance * math.Sin(angle)})
				}
				appendEffectQuad(&vertices, &indices, texture,
					[4]modelPoint3{origin, origin, point(yaw + degreesToRadians(width)), point(yaw - degreesToRadians(width))},
					color.RGBA{250, 250, 250, uint8(alpha)})
			}
			screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(layer == 0, false))
		}
		return
	case effectPortal4, effectPortal5:
		alpha := 10*min(n, 11) - 2*max(0, n-20)
		if alpha <= 0 {
			return
		}
		// PP_WIND stays at its creation point, unlike the caster's body flash.
		fullAngle := min(120, 30+3*n)
		step := fullAngle / 21
		segments := (fullAngle - 1) / step
		height, radiusBase, radiusRandom := 5, 4.5, .03
		tint := color.RGBA{255, 255, 255, uint8(alpha)}
		if effect.effectID == effectPortal5 {
			height, radiusBase, radiusRandom = 3, 2.5, .01
			tint.B = 170
		}
		for i := 0; i < 4; i++ {
			origin := effect.actorOrigin
			origin.y += float64(height+2*i) / 5
			rise := degreesToRadians(float64(80 + hitEffectRandom(effect, 2*i, 21)))
			radius := (radiusBase + radiusRandom*float64(hitEffectRandom(effect, 2*i+1, 101)) + .1*float64(max(0, n-20))) / 5
			start := len(vertices)
			appendEffectBand(&vertices, &indices, texture, origin, radius,
				degreesToRadians(float64(i*90+n*5)), degreesToRadians(float64(segments*step)), 1/math.Tan(rise), segments,
				tint, func(int) float64 { return math.Sin(rise) / 5 })
			for j := 0; j < segments; j++ {
				for k, u := range []int{min(j, 19), min(j+1, 20), min(j+1, 20), min(j, 19)} {
					vertices[start+4*j+k].SrcX = float32(u) / 21 * float32(texture.Bounds().Dx())
				}
			}
		}
	case effectTripleAttack2:
		for layer := 0; layer < 2; layer++ {
			vertices, indices = nil, nil
			for i := 0; i < 16; i++ {
				from, to, alpha := sharpShootingLine(effect, i, frame)
				if alpha > 0 {
					appendMonkTrail(&vertices, &indices, texture, from, to, .01+.05*float64(layer), 5, color.RGBA{255, 255, 255, uint8(alpha)})
				}
			}
			if len(indices) > 0 {
				screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(layer == 1, false))
			}
		}
		return
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(effect.effectID != effectPortal4, false))
	}
}

func sniperBodyFlashActive(effectID int, frame int) bool {
	switch effectID {
	case effectFalconAssault:
		return frame >= 30 && frame <= 50 && frame%10 < 5
	case effectPortal4:
		return frame >= 5 && frame <= 25
	}
	return false
}
