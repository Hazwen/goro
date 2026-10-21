package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) drawSonicBlowEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	right, up, units, ok := projection.BillboardBasis(x, y, z)
	if texture == nil || !ok {
		return
	}
	// Sakexe's PP_2DCIRCLE is a ten-pixel band, twenty pixels above the caster.
	n := float64(now.Sub(effect.starts)/db.EffectFrameDuration + 1)
	radius := 7*n - .175*n*(n+1)/2
	alpha := clampFloat((40-n)/40, 0, 1) * 254 / 255
	center := add3(modelPoint3{x: x, y: z, z: y}, mul3(up, 20*units))
	tint := color.RGBA{255, 255, 255, uint8(alpha * 255)}
	bounds := texture.Bounds()
	vertices := make([]render.Vertex3D, 0, 74)
	indices := make([]uint16, 0, 216)
	for i := 0; i <= 36; i++ {
		angle := float64(i) * math.Pi / 18
		axis := add3(mul3(right, math.Cos(angle)), mul3(up, math.Sin(angle)))
		for j, r := range [2]float64{math.Max(0, radius-10), radius} {
			point := add3(center, mul3(axis, r*units))
			uv := texturePoint{u: float32(i) / 4, v: float32(1 - j)}
			vertices = append(vertices, texturedSurfaceVertex3D(point, uv, tint, float32(bounds.Dx()), float32(bounds.Dy())))
		}
		if i < 36 {
			base := uint16(i * 2)
			indices = append(indices, base, base+1, base+2, base+1, base+3, base+2)
		}
	}
	options := triangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	options.DepthTest = false
	screen.DrawTriangles3DOwned(vertices, indices, texture, options)
}

type grimtoothHorn struct {
	origin, right, up, axis modelPoint3
	size, height, alpha     float64
}

func grimtoothSpike(effect worldEffect, index, frame int) grimtoothHorn {
	pitch, yaw := 75.0, float64((6-index)*120)
	size, height, speed, accel := .18, 5.0, .7, .0002
	life, stop, age, fade := 1000, 11, frame, 990
	offset := [3]modelPoint3{{y: -8, z: -2.4}, {x: 2.4, y: -8, z: 1.2}, {x: -2.4, y: -8, z: 1.2}}[index%3]
	if effect.effectID == effectGrimtooth {
		birth := index * 3
		age = frame - birth
		life, stop, fade = min(40, 250-birth), 7, 30
		if life < 40 {
			fade = 0
		}
		speed, accel, height = .6, -.03, 2
		size = float64(60+hitEffectRandom(effect, index*3+1, 40)) / 500
		pitch = float64(75 + hitEffectRandom(effect, index*3+2, 30))
		yaw = float64(hitEffectRandom(effect, index*3+3, 360))
		offset = add3(modelPoint3{y: -4}, mul3(effect.projectileDirection, float64(birth+1)))
	}
	if age < 0 || age >= life {
		return grimtoothHorn{}
	}
	// Native horns have a square base in XY and their tip on Z, rotated X then Y.
	// Convert the native downward Y axis to our upward one after the rotation.
	p, q := degreesToRadians(pitch), degreesToRadians(yaw)
	right := modelPoint3{x: math.Cos(q), z: -math.Sin(q)}
	up := modelPoint3{x: math.Sin(p) * math.Sin(q), y: -math.Cos(p), z: math.Sin(p) * math.Cos(q)}
	axis := modelPoint3{x: math.Cos(p) * math.Sin(q), y: math.Sin(p), z: math.Cos(p) * math.Cos(q)}
	n := float64(min(age+1, stop))
	distance := n*speed + accel*n*(n+1)/2
	return grimtoothHorn{
		origin: add3(effect.actorOrigin, add3(offset, mul3(axis, distance))),
		right:  right, up: up, axis: axis, size: size, height: height,
		alpha: clampFloat(float64(life-age-1)/float64(life-fade), 0, 1) * 254 / 255,
	}
}

func (m *WorldMode) drawGrimtoothEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	first, last := 0, 2
	if effect.effectID == effectGrimtooth {
		first, last = max(0, (frame-39+2)/3), min(frame/3, 83)
		if effect.projectileDirection == (modelPoint3{}) {
			last = 0
		}
	}
	bounds := texture.Bounds()
	vertices := make([]render.Vertex3D, 0, 168)
	indices := make([]uint16, 0, 168)
	for i := first; i <= last; i++ {
		horn := grimtoothSpike(effect, i, frame)
		if horn.alpha <= 0 {
			continue
		}
		tint := color.RGBA{255, 255, 255, uint8(horn.alpha * 255)}
		corners := [4]modelPoint3{
			mul3(add3(horn.right, horn.up), -horn.size),
			mul3(sub3(horn.right, horn.up), horn.size),
			mul3(add3(horn.right, horn.up), horn.size),
			mul3(sub3(horn.up, horn.right), horn.size),
		}
		for side, edge := range [4][2]int{{0, 1}, {0, 3}, {3, 2}, {2, 1}} {
			for j, local := range [3]modelPoint3{mul3(horn.axis, horn.height), corners[edge[0]], corners[edge[1]]} {
				indices = append(indices, uint16(len(vertices)))
				vertices = append(vertices, texturedSurfaceVertex3D(add3(horn.origin, local), quadHornUVs[side*3+j], tint, float32(bounds.Dx()), float32(bounds.Dy())))
			}
		}
	}
	if len(vertices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, triangleDrawOptions(render.FilterLinear, render.AddressRepeat))
	}
}

func sonicBlowSkillActionSpec() skillActionSpec {
	next := newSkillActionSpec(skillActorActionReadyFight, true, nil)
	for hit := 8; hit >= 1; hit-- {
		spec := newSkillActionSpec(skillActorActionAttack, false, &next)
		if hit > 1 {
			spec.speed = 30 * time.Millisecond
		}
		next = spec
	}
	return next
}
