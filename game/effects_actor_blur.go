package game

import (
	"image/color"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

type actorBlur struct {
	billboard *spriteBillboard
	born      time.Time
	x, y, z   float64
	scale     float64
	tint      color.RGBA
}

func (m *WorldMode) drawActorBlurs(screen *render.Frame, projection sceneProjection, entry sceneActorDrawEntry, billboard *spriteBillboard, scale, visibility, shadow float64, tint color.RGBA, now time.Time) {
	id := entry.actor.ID
	if actorHasStealth(entry.actor) {
		delete(m.actorBlurs, id)
		return
	}
	blurs := m.actorBlurs[id]
	live := blurs[:0]
	for _, blur := range blurs {
		if now.Sub(blur.born) < 23*db.EffectFrameDuration {
			live = append(live, blur)
		}
	}
	clear(blurs[len(live):])
	blurs = live
	// MakeBlur emits a frozen CBlurPC every five ticks, except on dyed bodies.
	active := (entry.isPlayer || actorRepresentsPlayer(entry.actor)) && entry.actor.BodyPal == 0 && entry.actor.Opt3State&db.Opt3Overthrust != 0
	if active && (len(blurs) == 0 || now.Sub(blurs[len(blurs)-1].born) >= 5*db.EffectFrameDuration) {
		blurs = append(blurs, actorBlur{billboard: billboard, born: now, x: entry.worldX, y: entry.worldY, z: entry.worldZ,
			scale: scale / actorBillboardScreenScale(projection, entry.worldX, entry.worldY, entry.worldZ), tint: tint})
	}
	if len(blurs) == 0 {
		delete(m.actorBlurs, id)
		return
	}
	if m.actorBlurs == nil {
		m.actorBlurs = make(map[uint32][]actorBlur)
	}
	m.actorBlurs[id] = blurs
	for _, blur := range blurs {
		alpha := float64(max(0, 180-8*int(now.Sub(blur.born)/db.EffectFrameDuration))) / 255
		m.drawActorBlur(screen, projection, blur, blur.billboard, visibility*alpha, shadow)
	}
}

func (m *WorldMode) drawActorBlur(screen *render.Frame, projection sceneProjection, blur actorBlur, billboard *spriteBillboard, alpha, shadow float64) {
	if len(billboard.parts) > 0 {
		for _, part := range billboard.parts {
			m.drawActorBlur(screen, projection, blur, part, alpha, shadow)
		}
		return
	}
	options := spriteBillboardTriangleDrawOptions()
	options.Filter = render.FilterSprite
	scale := blur.scale * actorBillboardScreenScale(projection, blur.x, blur.y, blur.z)
	drawSpriteBillboardTintAlpha3DWithOptions(screen, projection, billboard, m.spriteTexture(billboard.image), blur.x, blur.y, actorSpriteWorldZ(blur.z), scale, alpha, shadow, blur.tint, options)
}
