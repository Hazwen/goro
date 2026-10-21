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

func isAlchemistBottle(effectID int) bool {
	return effectID == effectThrowItem || effectID == effectThrowItem2 || effectID == effectThrowItem4 || effectID == effectThrowItem5
}

func (m *WorldMode) applyAlchemistSkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	effectID, level := effectThrowItem2, int(notify.Amount)
	switch notify.SkillID {
	case db.SkillAMPotionpitcher:
	case db.SkillAMBerserkpitcher:
		effectID, level = effectThrowItem5, 4
	default:
		return
	}
	if m.addWorldEffectBetweenAt(ctx, effectID, notify.TargetID, notify.SourceID, now) {
		m.worldEffects[len(m.worldEffects)-1].itemLevel = level
	}
}

func (m *WorldMode) drawChemicalProtection(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	alpha := max(0, 18*min(n, 10)-15*max(0, n-20))
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if alpha == 0 || texture == nil {
		return
	}
	center := effect.actorOrigin
	center.y += 1.8
	right, up, _, ok := projection.BillboardBasis(center.x, center.z, center.y)
	if !ok {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for i := 0; i < 8; i++ {
		angle := (i%4)*90 + 30 + (i/4)*30
		for layer := 0; layer < 2; layer++ {
			spread := 3 + layer*2
			c := color.RGBA{255, 255, 255, uint8(alpha / 2)}
			if layer == 1 {
				c = color.RGBA{130, 130, 255, uint8(alpha)}
			}
			edge := func(degrees int) modelPoint3 {
				sin, cos := math.Sincos(degreesToRadians(float64(degrees)))
				return add3(center, add3(mul3(right, 20*cos), mul3(up, 20*sin)))
			}
			appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{center, edge(angle + spread), edge(angle - spread), center}, c)
		}
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
}

type twilightParticle struct {
	position modelPoint3
	steering [4]int
	angle    int
	phase    int
	radius   float64
	texture  int
}

type twilightParticles struct {
	tick      int
	particles [320]twilightParticle
}

func newTwilightParticles(ctx client.Context, effect worldEffect) *twilightParticles {
	state := &twilightParticles{}
	// Both originals scatter the ingredients around the local viewer, even
	// when another player's cast launches the effect.
	center := effectWorldAnchor(ctx, worldEffect{actorID: localSkillTarget(ctx)}, effect.starts)
	for i := range state.particles {
		p := &state.particles[i]
		salt := i * 12
		p.position.x = center.x + float64(hitEffectRandom(effect, salt, 160)-80)/5
		p.position.z = center.z + float64(hitEffectRandom(effect, salt+1, 160)-80)/5
		p.position.y = terrainHeightAt(ctx.World, p.position.x-.5, p.position.z-.5) + float64(5+hitEffectRandom(effect, salt+2, 66))/5
		for j := range p.steering {
			p.steering[j] = hitEffectRandom(effect, salt+3+j, 360)
		}
		p.angle, p.phase = hitEffectRandom(effect, salt+7, 360), hitEffectRandom(effect, salt+8, 360)
		p.radius = (3 + .1*float64(hitEffectRandom(effect, salt+9, 11))) / 5
		if effect.effectID == effectTwilight3 {
			p.texture = hitEffectRandom(effect, (i/4)*48+10, 3)
		}
	}
	return state
}

func (state *twilightParticles) advance(effect worldEffect, frame int) {
	for state.tick <= frame {
		state.tick++
		for i := range state.particles {
			p := &state.particles[i]
			for j := 0; j < 4; j += 2 {
				angle, target := &p.steering[j], &p.steering[j+1]
				salt := 4000 + state.tick*1280 + i*4 + j
				if *target > *angle {
					*angle = min(*angle+1, 359)
					if *angle > *target {
						*target = hitEffectRandom(effect, salt, max(1, *angle))
					}
				} else {
					*angle = max(*angle-1, 0)
					if *angle <= *target {
						*target = 359 - hitEffectRandom(effect, salt, max(1, *angle))
					}
				}
			}
			p.position.x += .02 * math.Sin(degreesToRadians(float64(p.steering[0])))
			p.position.z += .02 * math.Sin(degreesToRadians(float64(p.steering[2])))
			p.angle, p.phase = (p.angle+1)%360, (p.phase+1)%360
		}
	}
}

func (m *WorldMode) drawTwilight(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	if effect.twilight == nil {
		return
	}
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	duration := int(effect.expires.Sub(effect.starts) / db.EffectFrameDuration)
	n := frame + 1
	alpha := max(0, 5*min(n, 49)-5*max(0, n-(duration-50)))
	if alpha == 0 {
		return
	}
	effect.twilight.advance(effect, frame)
	for index, file := range component.textureFiles {
		texture := m.effectFileTexture(ctx.Resources, file)
		if texture == nil {
			continue
		}
		var vertices []render.Vertex3D
		var indices []uint16
		for _, p := range effect.twilight.particles {
			if p.texture != index {
				continue
			}
			right, up, _, ok := projection.BillboardBasis(p.position.x, p.position.z, p.position.y)
			if !ok {
				continue
			}
			radius := p.radius * (1 + .05*math.Sin(degreesToRadians(float64(p.phase))))
			var corners [4]modelPoint3
			for j := range corners {
				sin, cos := math.Sincos(degreesToRadians(float64(p.angle + 180 + j*90)))
				corners[j] = add3(p.position, add3(mul3(right, radius*cos), mul3(up, radius*sin)))
			}
			appendEffectQuad(&vertices, &indices, texture, corners, color.RGBA{255, 255, 255, uint8(alpha)})
		}
		if len(indices) > 0 {
			screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
		}
	}
}
