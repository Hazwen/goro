package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

func healEffectID(amount uint32) int {
	// HighPriest 2008 uses EXIT for small heals, not HEAL4 as in dhxj.
	switch {
	case amount < 200:
		return effectExit
	case amount < 2000:
		return effectHeal
	case amount < 4000:
		return effectHeal2
	default:
		return effectHeal4
	}
}

func agilityLabel(decrease bool, frame, duration int) effectScreenParticle {
	n := float64(frame + 1)
	label := effectScreenParticle{
		halfWidth: 40, halfHeight: 20,
		y:     -(1.5*n - 1.8/float64(duration)*n*(n+1)/2),
		alpha: effectParticleAlpha(frame, duration, 15, duration-15, 200),
	}
	if decrease {
		label.halfHeight = 10
		label.y = -80 + n - .01*n*(n+1)/2
	}
	return label
}

func (m *WorldMode) drawAgilityEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	if now.Before(effect.starts) {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	duration := 100
	if effect.effectID == effectIncAgiDex && effect.duration > 0 {
		duration = max(1, int(effect.duration/db.EffectFrameDuration))
	}
	if frame >= duration {
		return
	}
	decrease := effect.effectID == effectDecAgility
	if origin, ok := projection.projectPoint(x, y, z); ok {
		drawEffectScreenParticle(screen, m.effectFileTexture(ctx.Resources, component.textureFile), origin, agilityLabel(decrease, frame, duration))
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFiles[0])
	if texture == nil {
		return
	}
	vertices := make([]render.Vertex3D, 0, 26*8)
	indices := make([]uint16, 0, 26*12)
	// Native Prim3DTexture follows the actor. One crossed streak is born every
	// two ticks; births whose 50-tick lifetime exceeds the parent fade from zero.
	for birth := max(0, frame/2*2-48); birth <= minInt(frame, duration-50); birth += 2 {
		age := frame - birth
		alpha := effectParticleAlpha(age, 50, 20, 30, 200)
		if alpha == 0 {
			continue
		}
		n := float64(age + 1)
		angle := degreesToRadians(float64(hitEffectRandom(effect, birth*4, 360)))
		radius := float64(2+hitEffectRandom(effect, birth*4+1, 7)) / 5
		height := float64(20+hitEffectRandom(effect, birth*4+2, 50)) / 100 * n / 5
		if decrease {
			height = 4 - .015*n*(n+1)/2/5
		}
		center := modelPoint3{x: x + math.Sin(angle)*radius, y: z + height, z: y + math.Cos(angle)*radius}
		length := float64(30+hitEffectRandom(effect, birth*4+3, 60)) / 10 / 5
		tint := color.RGBA{R: 255, G: 255, B: 255, A: alpha}
		for _, side := range []modelPoint3{{x: .18 / 5}, {z: .18 / 5}} {
			// The texture's U axis runs along the streak, as in Render3DCrossTexture.
			up := modelPoint3{y: length}
			appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{
				sub3(sub3(center, up), side), sub3(add3(center, up), side),
				add3(add3(center, up), side), add3(sub3(center, up), side),
			}, tint)
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(false, false))
	}
}

func blessingCircleAlpha(component worldEffectComponent, elapsed time.Duration) float64 {
	if elapsed < 0 || elapsed >= component.duration || component.frameDelay <= 0 {
		return 0
	}
	n := float64(elapsed/component.frameDelay + 1)
	frames := float64(component.duration / component.frameDelay)
	// Blessing's Prim3DCircle ramps for 30 ticks at both ends.
	return component.alphaMax * clampFloat(math.Min(n/30, (frames-n)/30), 0, 1)
}

func (m *WorldMode) drawBlessingCircleEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	alpha := blessingCircleAlpha(component, now.Sub(effect.starts)-component.delay)
	if alpha <= 0 || component.circleSides < 3 {
		return
	}
	texture := m.effectTexture(ctx.Resources, component.textureName)
	if texture == nil {
		return
	}
	bounds := texture.Bounds()
	w, h := float32(bounds.Dx()), float32(bounds.Dy())
	tint := effectComponentTint(component, alpha)
	vertices := make([]render.Vertex3D, 0, component.circleSides*3)
	indices := make([]uint16, 0, component.circleSides*3)
	center := modelPoint3{x: x, y: z + component.posZ, z: y}
	point := func(i int) modelPoint3 {
		angle := float64(i) * 2 * math.Pi / float64(component.circleSides)
		return add3(center, modelPoint3{x: component.sizeStart * math.Cos(angle), z: component.sizeStart * math.Sin(angle)})
	}
	// Render3DCircle with PT_FILLCIRCLE: a fan on the world X/Z plane,
	// alpha_down from centre (V=1) to rim (V=0), U advancing 1/4 per wedge.
	for i := 0; i < component.circleSides; i++ {
		u := float32(i) / 4
		base := uint16(len(vertices))
		vertices = append(vertices,
			texturedSurfaceVertex3D(center, texturePoint{u: u + 0.25, v: 1}, tint, w, h),
			texturedSurfaceVertex3D(point(i), texturePoint{u: u, v: 0}, tint, w, h),
			texturedSurfaceVertex3D(point(i+1), texturePoint{u: u + 0.25, v: 0}, tint, w, h))
		indices = append(indices, base, base+1, base+2)
	}
	options := triangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	options.DepthTest = !component.overlay // The original explicitly uses RF_NODEPTHCHECK.
	screen.DrawTriangles3DOwned(vertices, indices, texture, options)
}
