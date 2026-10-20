package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) applyHunterTrapSkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) bool {
	if notify.SkillID != db.SkillHTRemovetrap && notify.SkillID != db.SkillHTSpringtrap {
		return false
	}
	x, y, ok := m.skillUnitCell(notify.TargetID)
	if ok {
		for _, effectID := range skillEffectIDs(notify.SkillID) {
			// The following disappearance packet must not cut off the burst.
			m.addWorldEffectAtCellLifetime(ctx, effectID, 0, x, y, now, 0, false)
		}
		if notify.SkillID == db.SkillHTSpringtrap {
			m.startFalconAttackAt(ctx, notify.SourceID, x, y, now)
		}
	}
	return true
}

func (m *WorldMode) applyHunterGroundSkill(ctx client.Context, notify network.GroundSkillNotify, now time.Time) bool {
	switch notify.SkillID {
	case db.SkillHTSkidtrap, db.SkillHTLandmine, db.SkillHTAnklesnare, db.SkillHTShockwave,
		db.SkillHTSandman, db.SkillHTFlasher, db.SkillHTFreezingtrap, db.SkillHTBlastmine,
		db.SkillHTClaymoretrap, db.SkillHTTalkiebox, db.SkillHTDetecting:
	default:
		return false
	}
	m.faceSkillSource(ctx, notify.SourceID, 0, int(notify.X), int(notify.Y))
	if actor, ok, _ := actorForCombatID(ctx, notify.SourceID); ok {
		m.startSkillActionAnimation(ctx, notify.SourceID, actor, skillAction(notify.SkillID), now, defaultAttackAnimationDuration)
	}
	m.applyFalconGroundSkillNotify(ctx, notify, now)
	for _, effectID := range skillGroundEffectIDs(notify.SkillID) {
		m.addWorldEffectAtCellLifetime(ctx, effectID, 0, int(notify.X), int(notify.Y), now, 0, false)
	}
	return true
}

func (m *WorldMode) drawHunterEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, origin modelPoint3, now time.Time) {
	if now.Before(effect.starts) {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	switch effect.effectID {
	case effectDetecting:
		if frame >= 57 {
			return
		}
		// Prim3DTexture replaces the initial growth speed on tick 17.
		n := float64(min(frame+1, 17))
		size := 1.5*n + (1.5/14)*n*(n+1)/2
		n = float64(max(0, frame-16))
		size += .1*n + (.1/30)*n*(n+1)/2
		texture := m.effectFileTexture(ctx.Resources, component.textureFile)
		drawGroundTextureQuad(screen, texture, origin.x, origin.z, origin.y+1, size*2/5, size*2/5, 0,
			color.RGBA{255, 255, 255, effectParticleAlpha(frame, 57, 1, 40, 254)}, false)
	case effectRemoveTrap:
		m.drawRemoveTrap(screen, ctx, projection, component, effect, origin, frame)
	case effectFlasher:
		m.drawFlasher(screen, ctx, projection, component, effect, origin, frame)
	case effectBlitzBeat:
		m.drawBlitzBeat(screen, ctx, component, effect, origin, frame)
	}
}

func removeTrapParticle(effect worldEffect, index, age int) effectParticle {
	if age < 0 || age >= 50 {
		return effectParticle{}
	}
	n := float64(age + 1)
	yaw := degreesToRadians(float64(hitEffectRandom(effect, index*2, 360)))
	accel := .08 + float64(hitEffectRandom(effect, index*2+1, 10))/1000
	return effectParticle{
		position: modelPoint3{x: math.Sin(yaw) * .2 * n / 5, z: math.Cos(yaw) * .2 * n / 5, y: (2*n - accel*n*(n+1)/2) / 5},
		size:     .8,
		alpha:    float64(effectParticleAlpha(age, 50, 1, 35, 254)) / 255,
	}
}

func (m *WorldMode) drawRemoveTrap(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, origin modelPoint3, frame int) {
	for i := 0; i < 24; i++ {
		age := frame - (i/12)*7
		if age < 0 || age >= 50 {
			continue
		}
		p := removeTrapParticle(effect, i, age)
		m.drawEffectParticleSprite(screen, ctx, projection, component.spriteFile, add3(origin, p.position),
			age/4+1, p.size, 0, color.RGBA{255, 255, 255, uint8(p.alpha * 255)})
	}
}

func (m *WorldMode) drawFlasher(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, origin modelPoint3, frame int) {
	if frame < 0 || frame >= 70 {
		return
	}
	alpha := effectParticleAlpha(frame, 70, 6, 60, 200)
	if frame >= 40 && frame < 43 {
		alpha = uint8(math.Min(200, 120+float64(frame-39)*(200.0/6)))
	}
	drawTexturedEffectBillboardRotatedXY(screen, projection, m.effectFileTexture(ctx.Resources, component.textureFile),
		origin.x, origin.z, origin.y+2, 8, 8, 0, color.RGBA{255, 255, 255, alpha}, true)
	center, ok := projection.projectPoint(origin.x, origin.z, origin.y)
	texture := m.effectFileTexture(ctx.Resources, component.textureFiles[0])
	if !ok || texture == nil {
		return
	}
	center.y -= 40
	w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
	n := float64(frame + 1)
	a := float32(effectParticleAlpha(frame, 70, 10, 47, 200)) / 255
	vertex := func(angle, radius float64, u, v float32) render.Vertex {
		angle = degreesToRadians(angle)
		return render.Vertex{DstX: center.x + float32(math.Cos(angle)*radius), DstY: center.y - float32(math.Sin(angle)*radius),
			SrcX: u, SrcY: v, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: a}
	}
	vertices := make([]render.Vertex, 0, 60)
	indices := make([]uint16, 0, 60)
	for i := 0; i < 20; i++ {
		speed := float64(10+hitEffectRandom(effect, i*5+1, 60)) / 10
		angle := float64(hitEffectRandom(effect, i*5, 360)) + speed*n - speed/105*n*(n+1)/2
		halfAngle := float64(5+hitEffectRandom(effect, i*5+2, 25)) / 10
		radius := float64(20+hitEffectRandom(effect, i*5+3, 40)) + float64(50+hitEffectRandom(effect, i*5+4, 30))/10*n
		base := uint16(len(vertices))
		vertices = append(vertices, vertex(angle, 0, w/2, h), vertex(angle-halfAngle, radius, 0, 0), vertex(angle+halfAngle, radius, w, 0))
		indices = append(indices, base, base+1, base+2)
	}
	screen.DrawTrianglesOwned(vertices, indices, texture, &render.DrawTrianglesOptions{Filter: render.FilterLinear, Address: render.AddressRepeat})
}

func blitzStreakDistance(frame int) float64 {
	n := float64(min(frame+1, 10))
	distance := 15 - 1.2*n - .1*n*(n+1)/2
	n = float64(max(0, frame-9))
	return (distance - .3*n + .001*n*(n+1)/2) / 5
}

func (m *WorldMode) drawBlitzBeat(screen *render.Frame, ctx client.Context, component worldEffectComponent, effect worldEffect, origin modelPoint3, frame int) {
	// The attacked-effect table launches one EF_BLITZBEAT, with count=1,
	// on the first hit. Further damage ticks do not repeat it.
	yaw := 180 + 45*float64(normalizeDirectionIndex(effect.actorDirection))
	if frame < 20 {
		texture := m.effectFileTexture(ctx.Resources, component.textureFile)
		if texture == nil {
			return
		}
		forward := rotateEffectCylinderVector(modelPoint3{z: 1}, 0, yaw, 0)
		right := rotateEffectCylinderVector(modelPoint3{x: .2 / 5}, 0, yaw, 0)
		n := float64(frame + 1)
		vertices := make([]render.Vertex3D, 0, 80)
		indices := make([]uint16, 0, 120)
		for i := 0; i < 10; i++ {
			angle := degreesToRadians(float64(hitEffectRandom(effect, i*2, 360)))
			center := add3(origin, modelPoint3{x: .7 * math.Sin(angle), y: 1.4, z: -.7 * math.Cos(angle)})
			center = add3(center, mul3(forward, blitzStreakDistance(frame)))
			length := (float64(20+hitEffectRandom(effect, i*2+1, 40))/10 + .1*n - .01*n*(n+1)/2) / 5
			along := mul3(forward, length)
			for _, side := range []modelPoint3{right, {y: .2 / 5}} {
				appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{
					sub3(add3(center, along), side), sub3(sub3(center, along), side),
					add3(sub3(center, along), side), add3(add3(center, along), side),
				}, color.RGBA{255, 255, 255, effectParticleAlpha(frame, 20, 2, 18, 250)})
			}
		}
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(false, true))
	}
	age := frame - 36
	if age < 0 || age >= 12 {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFiles[0])
	if texture == nil {
		return
	}
	// Convert native downward Y to world upward Y after the X/Y rotations.
	rotate := func(p modelPoint3) modelPoint3 {
		p = rotateEffectCylinderVector(p, -90, yaw, 0)
		p.y = -p.y
		return p
	}
	n := float64(age + 1)
	distance := .4*n - (.4/24)*n*(n+1)/2
	center := add3(origin, mul3(rotate(modelPoint3{y: -2 - distance, z: 1}), .2))
	center.x += float64((1+hitEffectRandom(effect, 50, 10))/5) / 5
	center.y += float64(12-(1+hitEffectRandom(effect, 51, 20))/2) / 5
	drawWorldCylinderBandWithOptions(screen, m.whitePixel, texture, center.x, center.z, center.y,
		.5/5, 1.5/5, 3.5/5, color.RGBA{255, 255, 255, effectParticleAlpha(age, 12, 1, 6, 254)}, 10,
		rotate(modelPoint3{x: 1}), rotate(modelPoint3{z: 1}), rotate(modelPoint3{y: -1}), 2.5, render.BlendSourceOver)
}
