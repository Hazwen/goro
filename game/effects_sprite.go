package game

import (
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func (m *WorldMode) draw3DSpriteEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, effect worldEffect, component worldEffectComponent, worldX, worldY, worldZ float64, size float64, alpha float64, progress float64, starts time.Time, now time.Time) {
	view := m.effectSpriteView(ctx.Resources, component.spriteFile)
	if view == nil || len(view.act.Actions) == 0 {
		return
	}
	actionIndex := 0
	action := view.act.Actions[actionIndex]
	if len(action.Animations) == 0 {
		return
	}
	delayMS := float64(action.DelayMS)
	if component.spriteDelay > 0 {
		delayMS = float64(component.spriteDelay) / float64(time.Millisecond)
	}
	motion := 0
	if component.spriteRepeat {
		motion = spriteMotionIndexWithDelay(action, starts, now, true, delayMS)
	} else {
		motion = spriteMotionIndexWithDelay(action, starts, now, false, delayMS)
	}
	if motion < 0 || motion >= len(action.Animations) {
		return
	}
	ignoreLayerAngles := component.rotateToTarget
	key := singleSpriteBillboardKey{actionIndex: actionIndex, motion: motion, ignoreLayerAngles: ignoreLayerAngles}
	billboard, ok := view.billboards[key]
	if !ok {
		var baseOK bool
		billboard, baseOK = composeSingleSpriteBillboardWithOptions(view, action.Animations[motion], ignoreLayerAngles)
		if !baseOK {
			return
		}
		view.billboards[key] = billboard
	}
	tint := effectComponentTint(component, 1)
	scale := effect3DSpriteScale(size)
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = effectPixelRatio
	}
	options := effect3DSpriteDrawOptions(component)
	angle := 0.0
	if rotation, ok := effectSpriteRobrowserRotation(ctx, projection, component, effect, progress); ok {
		angle = rotation
	}
	drawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard, worldX, worldY, worldZ, scale, angle, alpha, 1, tint, options)
}

func effect3DSpriteDrawOptions(component worldEffectComponent) *render.DrawTrianglesOptions {
	options := spriteBillboardTriangleDrawOptions()
	if component.worldSizedSprite {
		options.DepthBias = strEffectDepthBias
	}
	if component.blendAdditive {
		options.Blend = render.BlendLighter
	}
	if component.overlay {
		options.DepthTest = false
	}
	return options
}

func effect3DSpriteScale(size float64) float64 {
	if size <= 0 || math.IsNaN(size) || math.IsInf(size, 0) {
		return effectPixelRatio
	}
	// robr's ThreeDEffect multiplies frame pixels by (layer.scale / 100) * size,
	// then SpriteRenderer converts those pixels to world units through /35.
	// effectTableSize has already applied /35, and ACT layer composition keeps
	// the layer scale in the composed bitmap, so the remaining per-pixel scale is
	// the table size divided by 100.
	return size / 100
}

func effectSpriteRobrowserRotation(ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, progress float64) (float64, bool) {
	angle := component.angleStart
	if component.rotate {
		angle += (component.angleEnd - component.angleStart) * progress
	}
	rotated := component.rotate || component.angleStart != 0 || component.angleEnd != 0
	if component.rotateToTarget {
		startX, startY, _, endX, endY, _, ok := effectTrajectoryEndpoints(ctx, component, effect)
		if ok {
			angle += 90 - math.Atan2(endY-startY, endX-startX)*180/math.Pi
			rotated = true
		}
	}
	if component.rotateWithCamera {
		angle += projection.cameraYaw
		rotated = true
	}
	if !rotated && math.Abs(angle) <= 0.0001 {
		return 0, false
	}
	return -angle * math.Pi / 180, true
}

func effectTrajectoryEndpoints(ctx client.Context, component worldEffectComponent, effect worldEffect) (float64, float64, float64, float64, float64, float64, bool) {
	if ctx.World == nil {
		return 0, 0, 0, 0, 0, 0, false
	}
	x, y, ok := effectAnchor(ctx, effect.actorID)
	if !ok {
		return 0, 0, 0, 0, 0, 0, false
	}
	worldX := cellCenter(float64(x))
	worldY := cellCenter(float64(y))
	worldZ := terrainHeightAt(ctx.World, float64(x), float64(y)) + 0.07
	startX := component.posX + component.posXStartMiddle
	startY := component.posY + component.posYStartMiddle
	startZ := component.posZ + component.posZStartMiddle
	endX := component.posXEnd + component.posXEndMiddle
	endY := component.posYEnd + component.posYEndMiddle
	endZ := component.posZEnd + component.posZEndMiddle
	if component.posXEnd == 0 && component.posXEndMiddle == 0 && component.posXStartMiddle == 0 {
		endX = startX
	}
	if component.posYEnd == 0 && component.posYEndMiddle == 0 && component.posYStartMiddle == 0 {
		endY = startY
	}
	if component.posZEnd == 0 && component.posZEndMiddle == 0 {
		endZ = startZ
	}
	if component.fromSrc || component.toSrc {
		otherX, otherY, otherZ, otherOK := effectOtherEndpoint(ctx, effect, worldX, worldY, worldZ)
		if otherOK {
			dx := otherX - worldX
			dy := otherY - worldY
			dz := otherZ - worldZ
			if component.fromSrc {
				endX += dx
				endY += dy
				endZ += dz
			}
			if component.toSrc {
				startX += dx
				startY += dy
				startZ += dz
			}
		}
	}
	return worldX + startX, worldY + startY, worldZ + startZ, worldX + endX, worldY + endY, worldZ + endZ, true
}

func (m *WorldMode) drawSPREffect(screen *render.Frame, ctx client.Context, projection sceneProjection, effect worldEffect, component worldEffectComponent, worldX, worldY, worldZ float64, now time.Time) {
	view := m.effectSpriteView(ctx.Resources, component.spriteFile)
	if view == nil || len(view.act.Actions) == 0 {
		return
	}
	actionIndex := component.spriteFrame
	if effect.hasSpriteFrame {
		actionIndex = effect.spriteFrameOverride
	}
	if component.spriteDirection {
		if actor, ok := ctx.World.Actors[effect.actorID]; ok {
			actionIndex = actorRenderDirection(actor, now) % len(view.act.Actions)
		} else if isLocalActor(ctx, effect.actorID) {
			actionIndex = actorRenderDirection(ctx.World.Player, now) % len(view.act.Actions)
		}
	}
	if actionIndex < 0 || actionIndex >= len(view.act.Actions) {
		actionIndex = 0
	}
	action := view.act.Actions[actionIndex]
	if len(action.Animations) == 0 {
		return
	}
	delayMS := float64(action.DelayMS)
	if component.spriteDelay > 0 {
		delayMS = float64(component.spriteDelay) / float64(time.Millisecond)
	}
	motion := 0
	if component.spriteRepeat {
		motion = spriteMotionIndexWithDelay(action, effect.starts, now, true, delayMS)
	} else {
		if delayMS <= 0 {
			delayMS = 150
		}
		motion = spriteMotionIndexWithDelay(action, effect.starts, now, false, delayMS)
		// The last frame remains visible until its own delay has elapsed.
		if !component.spriteStopAtEnd && component.duration <= 0 && float64(now.Sub(effect.starts))/float64(time.Millisecond) >= float64(len(action.Animations))*delayMS {
			return
		}
	}
	if motion < 0 || motion >= len(action.Animations) {
		return
	}
	key := singleSpriteBillboardKey{
		actionIndex: actionIndex,
		motion:      motion,
		anchorX:     component.spriteXOffset,
		anchorY:     component.spriteYOffset,
	}
	billboard, ok := view.billboards[key]
	if !ok {
		base, baseOK := composeSingleSpriteBillboard(view, action.Animations[motion])
		if !baseOK {
			return
		}
		copy := *base
		copy.anchorX -= component.spriteXOffset
		copy.anchorY -= component.spriteYOffset
		billboard = &copy
		view.billboards[key] = billboard
	}
	z := worldZ + component.posZ
	if component.spriteHead {
		z += 2.0
	}
	if component.worldSizedSprite {
		options := effect3DSpriteDrawOptions(component)
		drawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard, worldX, worldY, z, effectPixelRatio, 0, 1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255}, options)
		return
	}
	scale := 1.0
	if component.attachedEntity {
		scale = actorBillboardScreenScale(projection, worldX, worldY, worldZ)
	}
	drawSpriteBillboardTintAlpha3D(screen, projection, billboard, worldX, worldY, z, scale, 1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})
}

const effectSpriteRoot = "data\\sprite\\이팩트\\"

func (m *WorldMode) effectSpriteView(manager *res.Manager, file string) *spriteView {
	file = strings.TrimSpace(file)
	if manager == nil || file == "" {
		return nil
	}
	if m.effectViews == nil {
		m.effectViews = make(map[string]*spriteView)
	}
	if m.effectViewMiss == nil {
		m.effectViewMiss = make(map[string]struct{})
	}
	key := strings.ReplaceAll(file, "/", "\\")
	if view, ok := m.effectViews[key]; ok {
		return view
	}
	if _, ok := m.effectViewMiss[key]; ok {
		return nil
	}
	actCandidates := effectSpriteResourceCandidates(file, "act")
	sprCandidates := effectSpriteResourceCandidates(file, "spr")
	view, status := loadSpriteView(manager, actCandidates, sprCandidates, nil, "effect sprite "+file)
	if view == nil {
		m.effectViewMiss[key] = struct{}{}
		glog.Warnf("effect sprite unavailable file=%q: %s", file, status)
		return nil
	}
	m.effectViews[key] = view
	glog.Debugf("effect sprite resources file=%q %s", file, status)
	return view
}

func effectSpriteResourceCandidates(file, ext string) []string {
	normalized := strings.TrimSpace(strings.ReplaceAll(file, "/", "\\"))
	normalized = strings.TrimSuffix(normalized, ".spr")
	normalized = strings.TrimSuffix(normalized, ".act")
	if normalized == "" {
		return nil
	}
	base := normalized
	if !strings.HasPrefix(strings.ToLower(base), "data\\sprite\\") {
		base = effectSpriteRoot + base
	}
	path := base + "." + ext
	return []string{path, strings.ReplaceAll(path, "\\", "/")}
}

func (m *WorldMode) initDirectedSpriteEffect(ctx client.Context, effect *worldEffect) {
	effect.actorOrigin = effectWorldAnchor(ctx, *effect, effect.starts)
	caster := *effect
	caster.actorID = effect.targetID
	effect.projectileDirection = sub3(effectWorldAnchor(ctx, caster, effect.starts), effect.actorOrigin)
	if _, ok, _ := actorForCombatID(ctx, effect.targetID); effect.targetID != 0 && ok {
		effect.actorID, effect.targetID = effect.targetID, effect.actorID
	}
	effect.hasActorTransform = true
	yaw := cameraYawForMap(ctx)
	if !cameraRotationLockedForMap(ctx) {
		yaw += m.camera.yawOffset
	}
	effect.spriteFrameOverride = directedSpriteAction(effect.projectileDirection, yaw)
	effect.hasSpriteFrame = true
}

func directedSpriteAction(direction modelPoint3, yaw float64) int {
	angle := math.Atan2(direction.z, direction.x)*180/math.Pi - yaw
	quadrant := int(math.Floor((math.Mod(angle, 360)+360)/90)) % 4
	return (6 - quadrant) % 4
}

func (m *WorldMode) drawDirectedSpriteEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	component.duration = 0 // The non-repeating ACT owns its visible lifetime.
	p := effect.actorOrigin
	m.drawSPREffect(screen, ctx, projection, effect, component, p.x, p.z, p.y, now)
}

func (m *WorldMode) drawNativeSpriteEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, c worldEffectComponent, effect worldEffect, x, y, z float64, now time.Time) {
	if now.Before(effect.starts.Add(c.delay)) {
		return
	}
	if effect.effectID == effectKaen || effect.effectID >= effectLightningS && effect.effectID <= effectFlareS {
		period := 16 * db.EffectFrameDuration
		cycle := now.Sub(effect.starts) / period
		for previous := time.Duration(1); previous >= 0; previous-- {
			birth := (cycle - previous) * period
			if birth < 0 || (!effect.persistent && birth > 0) || now.Sub(effect.starts)-birth >= 23*db.EffectFrameDuration {
				continue
			}
			copy := effect
			copy.starts = effect.starts.Add(birth)
			if effect.effectID == effectKaen {
				c.spriteDelay = time.Duration(3+hitEffectRandom(effect, int(cycle-previous), 5)) * db.EffectFrameDuration
			}
			copy.starts = copy.starts.Add(-c.spriteDelay)
			m.drawSPREffect(screen, ctx, projection, copy, c, x, y, z, now)
		}
		return
	}
	// Prim3DParticle advances its first motion on tick zero.
	effect.starts = effect.starts.Add(c.delay - c.spriteDelay)
	c.delay = 0
	if effect.effectID == effectThunderStorm2 {
		m.draw3DSpriteEffect(screen, ctx, projection, effect, c, x, y, z+c.posZ, 250*effectPixelRatio, 1, 0, effect.starts, now)
		return
	}
	m.drawSPREffect(screen, ctx, projection, effect, c, x, y, z, now)
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
