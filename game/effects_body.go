package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func blueBodyAlpha(bodyTime int) float64 {
	if bodyTime < 0 || bodyTime > 50 {
		return 0
	}
	if bodyTime <= 10 {
		return float64(bodyTime*15) / 255
	}
	if bodyTime <= 20 {
		return 160.0 / 255
	}
	return float64(155-(bodyTime-20)*5) / 255
}

// Body effects reuse the composed sprite, including its appearance and scale.
// They never mutate the cached image or the actor's authoritative position.
func (m *WorldMode) drawActorSpriteWithEffects3D(ctx client.Context, screen *render.Frame, projection sceneProjection, entry sceneActorDrawEntry, billboard *spriteBillboard, scale, visibility, shadow float64, tint color.RGBA, blend render.Blend, now time.Time) {
	m.drawActorBlurs(screen, projection, entry, billboard, scale, visibility, shadow, tint, now)
	m.drawActorSpriteBodyEffects3D(ctx, screen, projection, entry, billboard, scale, visibility, shadow, tint, blend, now)
}

func (m *WorldMode) drawActorSpriteBodyEffects3D(ctx client.Context, screen *render.Frame, projection sceneProjection, entry sceneActorDrawEntry, billboard *spriteBillboard, scale, visibility, shadow float64, tint color.RGBA, blend render.Blend, now time.Time) {
	if len(billboard.parts) > 0 {
		for _, part := range billboard.parts {
			m.drawActorSpriteBodyEffects3D(ctx, screen, projection, entry, part, scale, visibility, shadow, tint, blend, now)
		}
		return
	}
	if billboard.layerGroup == 1 && entry.actor.Opt3State&db.Opt3Aurablade != 0 {
		blend = render.BlendLighter
	}
	doubleBody := false
	weaponSpark := false
	blue, red, eskeAge, fourWayAge := 0.0, 0.0, -1, -1
	soulAlpha := 0.0
	// Monsters use the Energy Coat option bit for Eske. The initial burst
	// settles into a faint outline until the server clears this bit.
	if isMonsterLikeHoverActor(entry.actor) && entry.actor.Opt3State&db.Opt3Energycoat != 0 {
		eskeAge = 149
	}
	var shake [4]float64
	squash, lift, turn := 1.0, 0.0, 0.0
	jump := m.starFusionHeight(entry.actor, now)
	zoomScale := 160 / normalizeSceneCameraZoom(projection.cameraZoom)
	local := isLocalActor(ctx, entry.actor.ID) || ctx.World != nil && entry.actor.ID != 0 && entry.actor.ID == ctx.World.Player.ID
	for _, effect := range m.worldEffects {
		owner := effect.actorID == entry.actor.ID || local && isLocalActor(ctx, effect.actorID)
		if !owner || now.Before(effect.starts) || effect.expired(now) {
			continue
		}
		frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
		switch effect.effectID {
		case effectHitLine2:
			if frame >= 10 && frame <= 20 {
				doubleBody = doubleBody || billboard.layerGroup == 0
				tint.R, tint.G, tint.B = 255, 120, 50
			}
		case effectChemical2Dash:
			if frame >= 20 && frame <= 100 {
				doubleBody = doubleBody || billboard.layerGroup == 0
				tint.R, tint.G, tint.B = 255, 255, 150
			}
		case effectBash3D2:
			if frame >= 5 && frame <= 35 && billboard.layerGroup == 0 {
				blend = render.BlendLighter
			}
		case effectPortal5:
			if frame >= 5 && frame <= 65 {
				tint.R, tint.G, tint.B = 250, 250, 200
				if billboard.layerGroup == 0 {
					blend = render.BlendLighter
				}
			}
		case effectRejectSword:
			if frame < 30 {
				doubleBody = doubleBody || billboard.layerGroup == 0
				shade := uint8(100)
				if frame%6 >= 3 {
					shade = 250
				}
				tint.R, tint.G, tint.B = shade, shade, shade
			}
		case effectPreserve:
			if frame >= 10 && frame < 31 {
				tint.R, tint.G, tint.B = 255, 255, 255
				if frame%2 == 0 {
					tint.R, tint.G, tint.B = 250, 250, 250
					if billboard.layerGroup == 0 {
						blend = render.BlendLighter
					}
				}
			}
		case effectSoulBreaker, effectSoulBreaker2, effectEDP:
			end := 10
			if effect.effectID == effectEDP {
				end = 80
			}
			if frame <= end {
				tint.R, tint.G, tint.B = 255, 255, 255
				if effect.effectID != effectEDP || frame%2 == 0 {
					tint.G = 0
					if billboard.layerGroup == 0 {
						blend = render.BlendLighter
					}
				}
			}
		case effectMagicCrasher:
			if billboard.layerGroup == 0 && entry.stealth == stealthVisible && magicCrasherFlashActive(now.Sub(effect.starts)) {
				blend = render.BlendLighter
			}
		case effectLightBlade:
			weaponSpark = billboard.layerGroup == 1 && frame%2 == 0
		case effectEnergyDrain2:
			if frame >= 55 && frame <= 65 {
				tint.R, tint.G, tint.B = 100, 100, 255
				if billboard.layerGroup == 0 {
					blend = render.BlendLighter
				}
			}
		case effectEnergyDrain3:
			if frame >= 50 && frame <= 80 {
				tint.R, tint.G, tint.B = uint8(250-2*frame), uint8(250-2*frame), 250
				doubleBody = billboard.layerGroup == 0
			}
		case effectLinkLight:
			if frame >= 40 && frame <= 70 {
				tint.R, tint.G, tint.B = 200, 150, 50
				if billboard.layerGroup == 0 {
					blend = render.BlendLighter
				}
			}
		case effectMemorize, effectDoubleCastBody:
			end := 36
			if effect.effectID == effectDoubleCastBody {
				end = 51
			}
			if frame >= 10 && frame < end {
				doubleBody = doubleBody || effect.effectID == effectDoubleCastBody && billboard.layerGroup == 0
				tint.R, tint.G, tint.B = 255, 255, 255
				if frame%2 == 0 {
					tint.R, tint.G, tint.B = 250, 250, 100
					if effect.effectID == effectDoubleCastBody {
						tint.R, tint.G, tint.B = 255, 0, 0
					}
					if billboard.layerGroup == 0 {
						blend = render.BlendLighter
					}
				}
			}
		case effectItemFast3:
			if frame <= 150 {
				tint.R, tint.G, tint.B = 250, 120, 120
			}
		case effectAssumptio:
			doubleBody = billboard.layerGroup == 0
		case effectPinkBody:
			tint.R, tint.G, tint.B = 255, 89, 182
			if billboard.layerGroup == 0 {
				doubleBody, blend = true, render.BlendLighter
			}
		case effectChemicalBody:
			if frame < 31 {
				doubleBody = billboard.layerGroup == 0
				tint.R, tint.G, tint.B = 255, 255, 255
				if frame%2 == 0 {
					tint.R, tint.G = 0, 0
					if billboard.layerGroup == 0 {
						blend = render.BlendLighter
					}
				}
			}
		case effectRedBody:
			tint.R, tint.G, tint.B = 255, 100, 100
			if billboard.layerGroup == 0 {
				blend = render.BlendLighter
			}
		case effectLKConcentration:
			tint.R, tint.G, tint.B = 255, 255, 160
		case effectPierceBody:
			if frame < 141 {
				doubleBody = billboard.layerGroup == 0
				tint.R, tint.G, tint.B = 255, 255, 255
				if frame%2 == 0 {
					tint.R, tint.G, tint.B = 250, 250, 100
					if billboard.layerGroup == 0 {
						blend = render.BlendLighter
					}
				}
			}
		case effectBash3D3, effectBash3D4:
			if frame >= 20 && frame <= 50 {
				doubleBody = billboard.layerGroup == 0
				tint.R, tint.G, tint.B = 255, 255, 255
				if effect.effectID == effectBash3D3 {
					tint.B = 200
				}
			}
		case effectAsuraBody:
			soulAlpha = max(soulAlpha, float64(min(frame, 200))/255)
		case effectFlowerCast:
			soulAlpha = max(soulAlpha, float64(max(0, min(frame, 400-frame)))/255)
		case effectRedLightBody:
			red = max(red, blueBodyAlpha(min(frame/8, 25)))
		case effectTaeReady:
			blue = max(blue, blueBodyAlpha(frame))
		case effectMadnessBlue:
			blue = max(blue, blueBodyAlpha(20+frame%32))
		case effectMadnessRed:
			red = max(red, blueBodyAlpha(20+frame%32))
		case effectRedHit:
			if frame <= 16 {
				red = max(red, blueBodyAlpha(frame*3))
			}
		case effectBlueHit:
			if frame <= 16 {
				blue = max(blue, blueBodyAlpha(frame*3))
			}
		case effectBlueBody, effectReflectBody:
			if frame <= 150 {
				blue = max(blue, blueBodyAlpha(frame/3))
			}
		case effectHated, effectKaizel:
			bodyTime := min(frame, 25)
			if frame > 150 {
				bodyTime = max(175-frame, 0)
			}
			blue = max(blue, blueBodyAlpha(bodyTime))
		case effectAsuraBodyMonster:
			eskeAge = min(frame, 149)
		case effectFourWayBody:
			if frame >= 30 && frame < 60 {
				fourWayAge = frame - 30
			}
		case effectPressedBody:
			if frame >= 17 && frame < 50 {
				age := frame - 17
				squash = .4 - float64(hitEffectRandom(effect, frame, 6))*.01
				if age <= 3 {
					squash = 1 - .2*float64(age)
				}
			}
		case effectKickedBody:
			age := frame - 30
			if age >= 0 && age <= 23 {
				lift = .3 * math.Sin(degreesToRadians(float64(age*5)))
				if age > 18 {
					lift = .3
				}
				if age > 20 {
					lift = float64(23-age) * .1
				}
			}
		case effectSpinedBody, effectSpinedBody2:
			age := frame * 2
			if effect.effectID == effectSpinedBody2 {
				age = frame - 14
			}
			if age >= 0 && age < 36 {
				turn = degreesToRadians(180 - 180*math.Cos(degreesToRadians(float64(age*5))))
			}
		case effectJumpBody:
			if frame > 35 {
				jump = float64((frame - 35) * 20)
				visibility *= float64(max(0, 255-(frame-35)*20)) / 255
			}
		case effectLandBody:
			jump = float64(max(0, 25-frame) * 20)
			visibility *= float64(clampInt((frame-10)*17, 0, 255)) / 255
			if frame < 25 {
				turn = degreesToRadians(180 - 180*math.Cos(degreesToRadians(float64((frame+11)*5))))
			}
		case effectQuakeBody, effectQuakeBody2, effectQuakeBody3, effectQuakeBody4:
			start, end := 30, 50
			randomRange, randomOffset := 21, 10
			if effect.effectID == effectQuakeBody || effect.effectID == effectQuakeBody2 {
				start, end = -1, 35
				if res.HasPlayerJobToken(int(entry.actor.Job)) {
					randomRange, randomOffset = 11, 5
				}
			}
			if effect.effectID == effectQuakeBody4 {
				start, end = 20, 59
				if frame > start && frame <= end && frame%2 == 0 {
					tint = multiplyTint(tint, 250.0/255, 50.0/255, 50.0/255, 1)
				}
			}
			if frame > start && frame <= end {
				for i := range shake {
					shake[i] = float64(hitEffectRandom(effect, frame*4+i, randomRange)-randomOffset) * zoomScale
				}
			}
		}
	}
	reflectBody := entry.actor.Opt3State&db.Opt3Kaite != 0
	if blue == 0 && red == 0 && soulAlpha == 0 && eskeAge < 0 && fourWayAge < 0 && shake == [4]float64{} && !reflectBody && !doubleBody && !weaponSpark && squash == 1 && lift == 0 && turn == 0 && jump == 0 {
		m.drawActorSpriteBillboardTintAlpha3D(screen, projection, billboard, entry.worldX, entry.worldY, entry.worldZ, scale, visibility, shadow, tint, blend)
		return
	}
	x, y, z := entry.worldX, entry.worldY, actorSpriteWorldZ(entry.worldZ)
	right, up, units, ok := projection.BillboardBasis(x, y, z)
	if !ok {
		return
	}
	w, h := float64(billboard.image.Bounds().Dx()), float64(billboard.image.Bounds().Dy())
	left, top := -billboard.anchorX*scale+shake[2], -billboard.anchorY*scale+shake[0]
	width, height := w*scale+shake[3]-shake[2], h*scale+shake[1]-shake[0]
	top += height*(1-squash-lift) - jump*zoomScale
	height *= squash
	if w == 0 || h == 0 || width == 0 || height == 0 {
		return
	}
	options := spriteBillboardTriangleDrawOptions()
	options.Filter, options.Blend = render.FilterSprite, blend
	alpha := visibility * float64(tint.A) / 255
	center := modelPoint3{x: x, y: z, z: y}
	if turn != 0 {
		s, c := math.Sincos(turn)
		// Rotate around the native body's pivot, 40 pixels above its feet.
		center = add3(center, mul3(add3(mul3(right, -s), mul3(up, 1-c)), 40*zoomScale*units))
		right, up = sub3(mul3(right, c), mul3(up, s)), add3(mul3(right, s), mul3(up, c))
	}
	cmd := worldSpriteBillboardCommand(m.spriteTexture(billboard.image), *options,
		center, mul3(right, units*width/w), mul3(up, -units*height/h),
		float32(w), float32(h), float32(-left*w/width), float32(-top*h/height), multiplyTint(tint, shadow, shadow, shadow, visibility))
	cmd.DepthUpAxis = [3]float32{0, float32(-units * height / h), 0}
	if weaponSpark {
		spark := cmd
		spark.Options.Blend = render.BlendLighter
		spark.Options.DepthWrite = false
		screen.DrawWorldBillboard(spark)
	}
	if doubleBody {
		phase := float64(now.UnixNano() / int64(db.EffectFrameDuration) % 181)
		drawBodyExpansion(screen, cmd, billboard, right, up, units, scale,
			(5+1.5*math.Sin(degreesToRadians(phase)))*zoomScale, alpha, render.BlendLighter, tint)
	}
	// Native Kaite draws four expanding silhouettes behind a translucent body.
	if reflectBody && blend != render.BlendLighter {
		phase := float64(now.UnixNano()/int64(db.EffectFrameDuration)%200) * .1
		for i := 1; i <= 4; i++ {
			spread := math.Mod(float64(i*5)+phase, 20)
			drawBodyExpansion(screen, cmd, billboard, right, up, units, scale, spread*zoomScale, alpha*(100-spread*5)/255, blend, color.RGBA{255, 255, 255, 255})
		}
		cmd.ColorA *= 150.0 / 255
	}
	cmd.Options.DepthWrite = true
	screen.DrawWorldBillboard(cmd)
	cmd.Options.DepthWrite = false
	if blue > 0 {
		flash := cmd
		flash.Options.Blend = render.BlendLighter
		flash.ColorR, flash.ColorG, flash.ColorB, flash.ColorA = 5.0/255, 5.0/255, 1, float32(blue*alpha)
		screen.DrawWorldBillboard(flash)
		screen.DrawWorldBillboard(flash)
	}
	if red > 0 {
		flash := cmd
		flash.Options.Blend = render.BlendLighter
		flash.ColorR, flash.ColorG, flash.ColorB, flash.ColorA = 1, 5.0/255, 5.0/255, float32(red*alpha)
		screen.DrawWorldBillboard(flash)
		screen.DrawWorldBillboard(flash)
	}
	if soulAlpha > 0 {
		// Pc::RenderBodyLayer scales two blue copies around the feet by 1% and 2%.
		for i := 1; i <= 2; i++ {
			copy := cmd
			factor := float32(1 + float64(i)*.01)
			for axis := range copy.RightAxis {
				copy.RightAxis[axis] *= factor
				copy.UpAxis[axis] *= factor
				copy.DepthUpAxis[axis] *= factor
			}
			copy.Options.Blend = render.BlendLighter
			copy.ColorR, copy.ColorG, copy.ColorB, copy.ColorA = 55.0/255, 55.0/255, 1, float32(soulAlpha*alpha)
			screen.DrawWorldBillboard(copy)
		}
	}
	if eskeAge >= 0 {
		for i := 1; i <= 10; i++ {
			a := eskeAge - i*5
			if eskeAge > 80 {
				a = 95 - eskeAge + (10-i)*10
			}
			drawBodyExpansion(screen, cmd, billboard, right, up, units, scale, float64(i*3)*zoomScale, alpha*float64(clampInt(a, 0, 200))/255, render.BlendLighter, color.RGBA{255, 255, 255, 255})
		}
	}
	if fourWayAge >= 0 {
		for _, offset := range [...]modelPoint3{right, mul3(right, -1), up, mul3(up, -1)} {
			copy := cmd
			shift := mul3(offset, units*height*.05*float64(fourWayAge)*zoomScale)
			copy.Center[0] += float32(shift.x)
			copy.Center[1] += float32(shift.y)
			copy.Center[2] += float32(shift.z)
			copy.ColorR, copy.ColorG, copy.ColorB, copy.ColorA = 1, 1, 1, float32(alpha*float64(150-fourWayAge*5)/255)
			screen.DrawWorldBillboard(copy)
		}
	}
}

func drawBodyExpansion(screen *render.Frame, cmd render.WorldBillboardCommand, billboard *spriteBillboard, right, up modelPoint3, units, scale, spread, alpha float64, blend render.Blend, tint color.RGBA) {
	if alpha <= 0 {
		return
	}
	w, h := float64(billboard.image.Bounds().Dx()), float64(billboard.image.Bounds().Dy())
	width, height := w*scale+2*spread, h*scale+2*spread
	r, u := mul3(right, units*width/w), mul3(up, -units*height/h)
	cmd.RightAxis = [3]float32{float32(r.x), float32(r.y), float32(r.z)}
	cmd.UpAxis = [3]float32{float32(u.x), float32(u.y), float32(u.z)}
	cmd.DepthUpAxis = [3]float32{0, float32(-units * height / h), 0}
	cmd.AnchorX = float32((billboard.anchorX*scale + spread) * w / width)
	cmd.AnchorY = float32((billboard.anchorY*scale + spread) * h / height)
	cmd.Options.Blend = blend
	cmd.ColorR, cmd.ColorG, cmd.ColorB, cmd.ColorA = float32(tint.R)/255, float32(tint.G)/255, float32(tint.B)/255, float32(alpha)
	screen.DrawWorldBillboard(cmd)
}

func (m *WorldMode) skillEffectDirection(ctx client.Context, actorID uint32, direction int, now time.Time) int {
	for _, effect := range m.worldEffects {
		if (effect.effectID != effectFalconAssault && effect.effectID != effectCastSpin && effect.effectID != effectStormKick) || now.Before(effect.starts) || effect.expired(now) {
			continue
		}
		if effect.actorID != actorID && !(isLocalActor(ctx, actorID) && isLocalActor(ctx, effect.actorID)) {
			continue
		}
		frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
		turn := 0.0
		switch effect.effectID {
		case effectStormKick:
			if frame > 47 {
				continue
			}
			turn = float64(frame+1) * 30
		case effectFalconAssault:
			if frame < 30 || frame > 54 {
				continue
			}
			turn = float64(frame-29) * 30
		case effectCastSpin:
			anim, ok := m.actorAnims[effect.actorID]
			if frame >= 100 || !ok || anim.actionFamily != spriteActionIdle || !anim.started.Equal(effect.starts) || now.Sub(anim.started) >= anim.duration {
				continue
			}
			ticks := frame + 1
			turn = float64(12*min(ticks, 16) + 18*clampInt(ticks-16, 0, 15) + 24*clampInt(ticks-31, 0, 10) + 48*max(0, ticks-41))
		default:
			continue
		}
		direction = (direction + int(math.Floor((turn+22.5)/45))) % 8
	}
	return direction
}
