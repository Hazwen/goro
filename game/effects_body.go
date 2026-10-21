package game

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
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
	blue, eskeAge, fourWayAge := 0.0, -1, -1
	soulAlpha := 0.0
	// Monsters use the Energy Coat option bit for Eske. The initial burst
	// settles into a faint outline until the server clears this bit.
	if isMonsterLikeHoverActor(entry.actor) && entry.actor.Opt3State&db.Opt3Energycoat != 0 {
		eskeAge = 149
	}
	var shake [4]float64
	zoomScale := 160 / normalizeSceneCameraZoom(projection.cameraZoom)
	local := isLocalActor(ctx, entry.actor.ID) || ctx.World != nil && entry.actor.ID != 0 && entry.actor.ID == ctx.World.Player.ID
	for _, effect := range m.worldEffects {
		owner := effect.actorID == entry.actor.ID || local && isLocalActor(ctx, effect.actorID)
		if !owner || now.Before(effect.starts) || effect.expired(now) {
			continue
		}
		frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
		switch effect.effectID {
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
			soulAlpha = float64(min(frame, 200)) / 255
		case effectTaeReady:
			blue = max(blue, blueBodyAlpha(frame))
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
		case effectQuakeBody3, effectQuakeBody4:
			start, end := 30, 50
			if effect.effectID == effectQuakeBody4 {
				start, end = 20, 59
				if frame > start && frame <= end && frame%2 == 0 {
					tint = multiplyTint(tint, 250.0/255, 50.0/255, 50.0/255, 1)
				}
			}
			if frame > start && frame <= end {
				for i := range shake {
					shake[i] = float64(hitEffectRandom(effect, frame*4+i, 21)-10) * zoomScale
				}
			}
		}
	}
	reflectBody := entry.actor.Opt3State&db.Opt3Kaite != 0
	if blue == 0 && soulAlpha == 0 && eskeAge < 0 && fourWayAge < 0 && shake == [4]float64{} && !reflectBody && !doubleBody && !weaponSpark {
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
	if w == 0 || h == 0 || width == 0 || height == 0 {
		return
	}
	options := spriteBillboardTriangleDrawOptions()
	options.Filter, options.Blend = render.FilterSprite, blend
	alpha := visibility * float64(tint.A) / 255
	cmd := worldSpriteBillboardCommand(m.spriteTexture(billboard.image), *options,
		modelPoint3{x: x, y: z, z: y}, mul3(right, units*width/w), mul3(up, -units*height/h),
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
