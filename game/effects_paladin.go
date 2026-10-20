package game

import (
	"github.com/kivutar/goro/network"
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
)

const (
	effectShieldBoomerang3 = 520
	effectShrink           = 599
)

type shieldImage struct {
	position modelPoint3
	angle    float64
	alpha    int
}

type shieldFlight struct {
	images         [4]shieldImage
	tick, phase    int
	heading, slope float64
	distance       float64
}

type shieldChainFlight struct {
	origins [5]modelPoint3
	count   int
}

func (chain *shieldChainFlight) advance(ctx client.Context, effect worldEffect, now time.Time) {
	for chain.count < len(chain.origins) {
		at := effect.starts.Add(time.Duration(3*chain.count) * db.EffectFrameDuration)
		if now.Before(at) {
			break
		}
		chain.origins[chain.count] = effectWorldAnchor(ctx, effect, at)
		chain.count++
	}
}

func initShieldBoomerang(ctx client.Context, effect *worldEffect) {
	destination := effectWorldAnchor(ctx, *effect, effect.starts)
	if _, ok, _ := actorForCombatID(ctx, effect.targetID); effect.targetID != 0 && ok {
		effect.actorID, effect.targetID = effect.targetID, effect.actorID
		effect.x, effect.y, _ = effectAnchor(ctx, effect.actorID)
	}
	origin := effectWorldAnchor(ctx, *effect, effect.starts)
	delta := sub3(destination, origin)
	distance := math.Hypot(delta.x, delta.z)
	origin.y += 1.6
	effect.shield = &shieldFlight{
		images:  [4]shieldImage{{position: origin, alpha: 255, angle: -3 * math.Pi / 4}},
		heading: math.Atan2(delta.z, delta.x), distance: distance,
	}
	if distance > 0 {
		effect.shield.slope = .4 * delta.y / distance
	}
}

func (flight *shieldFlight) advance(tick int, caster modelPoint3) {
	for flight.tick < min(tick, 499) {
		flight.tick++
		current := &flight.images[0]
		if flight.tick%3 == 0 {
			flight.images[3] = flight.images[2]
			flight.images[3].alpha = max(0, flight.images[3].alpha-25)
			flight.images[2] = flight.images[1]
			flight.images[2].alpha = max(0, flight.images[2].alpha-25)
			flight.images[1] = *current
			flight.images[1].alpha = max(0, current.alpha-150)
		}
		current.angle -= math.Pi / 12
		switch flight.phase {
		case 0:
			if .4*float64(flight.tick) > flight.distance {
				flight.phase = 1
			}
			current.position.y += flight.slope
		case 1:
			want := math.Atan2(caster.z-current.position.z, caster.x-current.position.x)
			want = math.Mod(want+2*math.Pi, 2*math.Pi)
			flight.heading = math.Mod(flight.heading+2*math.Pi, 2*math.Pi)
			if math.Abs(want-flight.heading) > degreesToRadians(4) {
				flight.heading = math.Mod(flight.heading+degreesToRadians(5), 2*math.Pi)
			} else {
				flight.phase = 2
			}
		case 2:
			current.alpha = max(0, current.alpha-12)
			current.position.y -= flight.slope
		}
		current.position.x += .4 * math.Cos(flight.heading)
		current.position.z += .4 * math.Sin(flight.heading)
	}
}

func drawShieldImage(screen *render.Frame, projection sceneProjection, texture *render.Image, particle shieldImage, size float64, additive bool) {
	if particle.alpha <= 0 {
		return
	}
	p := particle.position
	drawTexturedEffectBillboardRotatedXY(screen, projection, texture, p.x, p.z, p.y,
		size, size, particle.angle, color.RGBA{255, 255, 255, uint8(min(255, particle.alpha))}, additive)
}

func (m *WorldMode) drawShieldBoomerangEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil || effect.shield == nil {
		return
	}
	effect.shield.advance(int(now.Sub(effect.starts)/db.EffectFrameDuration)+1, effectWorldAnchor(ctx, effect, now))
	for _, particle := range effect.shield.images {
		drawShieldImage(screen, projection, texture, particle, 7*math.Sqrt2/5, false)
	}
}

func shieldChainImage(effect worldEffect, origin modelPoint3, index, process int) shieldImage {
	alpha := min(250, 25+45*process) - 10*max(0, process-14)
	direction := deterministicAngle(effect, index)
	position := origin
	position.x += .5 * float64(process-15) * math.Cos(direction)
	position.z += .5 * float64(process-15) * math.Sin(direction)
	position.y += 1.6
	return shieldImage{position: position, alpha: max(0, alpha), angle: -float64(process)*math.Pi/12 - 3*math.Pi/4}
}

func (m *WorldMode) drawShieldChainEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil || effect.shieldChain == nil {
		return
	}
	effect.shieldChain.advance(ctx, effect, now)
	frame := int(now.Sub(effect.starts) / db.EffectFrameDuration)
	for i := 0; i < 5; i++ {
		process := frame - 3*i + 1
		if process <= 0 {
			continue
		}
		origin := effect.shieldChain.origins[i]
		drawShieldImage(screen, projection, texture, shieldChainImage(effect, origin, i, process), math.Sqrt2, false)
		for trail := 0; trail < 3; trail++ {
			previous := process/3*3 - 1 - 3*trail
			if previous < 2 {
				continue
			}
			particle := shieldChainImage(effect, origin, i, previous)
			particle.alpha -= 150 + 25*trail
			drawShieldImage(screen, projection, texture, particle, math.Sqrt2, false)
		}
	}
}

func pressureImage(effect worldEffect, process int) shieldImage {
	position := effect.actorOrigin
	position.y += 23 - .6*float64(min(process, 32))
	return shieldImage{
		position: position,
		alpha:    max(0, min(process, 10)*20-max(0, process-32)*5),
		angle:    deterministicAngle(effect, 0) - float64(min(process, 32))*math.Pi/12,
	}
}

func (m *WorldMode) drawPressureEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	process := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	for i := 0; i < 4; i++ {
		if process-i <= 0 {
			continue
		}
		particle := pressureImage(effect, process-i)
		if i > 0 {
			particle.alpha -= 100 + 25*(i-1)
		}
		drawShieldImage(screen, projection, texture, particle, 12*math.Sqrt2/5, true)
	}
	age := process - 28
	alpha := min(age, 10)*2 - max(0, age-10)
	if age > 0 && alpha > 0 {
		if ground := m.effectFileTexture(ctx.Resources, component.textureFiles[0]); ground != nil {
			origin := effectWorldAnchor(ctx, effect, now)
			origin.y++
			radius := 2.7 * math.Pow(1.07, float64(age))
			vertices, indices := []render.Vertex3D{}, []uint16{}
			appendEffectQuad(&vertices, &indices, ground, [4]modelPoint3{
				add3(origin, modelPoint3{x: radius}), add3(origin, modelPoint3{z: radius}),
				add3(origin, modelPoint3{x: -radius}), add3(origin, modelPoint3{z: -radius}),
			}, color.RGBA{255, 255, 255, uint8(alpha)})
			screen.DrawTriangles3DOwned(vertices, indices, ground, texturedEffectBillboardDrawOptions(true, false))
		}
	}
}

func (m *WorldMode) drawGospelCrossEffect(screen *render.Frame, ctx client.Context, projection sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	n := float64(now.Sub(effect.starts)/db.EffectFrameDuration + 1)
	origin := effectWorldAnchor(ctx, effect, now)
	origin.y += 1.2 - .6*math.Sin(deterministicAngle(effect, effect.x*31+effect.y)+degreesToRadians(n))
	size := 1.5 * math.Sqrt2 / 5 * (1 + .05*math.Sin(degreesToRadians(n*5)))
	drawShieldImage(screen, projection, texture, shieldImage{position: origin, alpha: 200}, size, true)
}

func (m *WorldMode) drawGuardEffect(screen *render.Frame, ctx client.Context, _ sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	alpha := min(100, n*20) - 6*max(0, n-26)
	if alpha <= 0 {
		return
	}
	origin := effectWorldAnchor(ctx, effect, now)
	facing := degreesToRadians(float64(90 + 45*effectActorDirection(ctx, effect.actorID, now)))
	for layer := 0; layer < 3; layer++ {
		path := component.textureFile
		if layer == 2 {
			path = component.textureFiles[0]
		}
		texture := m.effectFileTexture(ctx.Resources, path)
		if texture == nil {
			continue
		}
		vertices, indices := []render.Vertex3D{}, []uint16{}
		radius, centerHeight := 7.8+.2*float64(layer), 7.8
		if layer == 0 {
			centerHeight += 2
		}
		if layer == 2 {
			radius += .1 * float64(max(0, n-26))
		}
		for row := 0; row < 3; row++ {
			pitch := degreesToRadians(float64(row*45 - 45))
			halfSize := float64(3 - absInt(row-1))
			for col := 0; col < 5; col++ {
				yaw := facing + degreesToRadians(float64(col*45-90))
				center := modelPoint3{x: radius * math.Cos(pitch) * math.Cos(yaw), y: radius*math.Sin(pitch) - centerHeight, z: radius * math.Cos(pitch) * math.Sin(yaw)}
				var corners [4]modelPoint3
				for i, offset := range [4][2]float64{{halfSize, halfSize}, {-halfSize, halfSize}, {-halfSize, -halfSize}, {halfSize, -halfSize}} {
					x, y := offset[0], offset[1]
					z := y * math.Cos(math.Pi/2+pitch)
					point := add3(center, modelPoint3{
						x: x*math.Cos(yaw-math.Pi/2) - z*math.Sin(yaw-math.Pi/2),
						y: y * math.Sin(math.Pi/2+pitch),
						z: x*math.Sin(yaw-math.Pi/2) + z*math.Cos(yaw-math.Pi/2),
					})
					corners[3-i] = add3(origin, modelPoint3{x: point.x / 5, y: -point.y / 5, z: point.z / 5})
				}
				green := []uint8{205, 155, 225}[layer]
				appendEffectQuad(&vertices, &indices, texture, corners, color.RGBA{green, 255, green, uint8(alpha)})
			}
		}
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(layer < 2, false))
	}
}

func (m *WorldMode) drawSacrificeEffect(screen *render.Frame, ctx client.Context, _ sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	process := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1 - 24
	alpha := min(process, 10)*20 - max(0, process-12)*15
	if process <= 0 || alpha <= 0 {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	origin := effectWorldAnchor(ctx, effect, now)
	origin.y += 2.4
	distance := .4 * math.Pow(1.15, float64(process))
	vertices, indices := []render.Vertex3D{}, []uint16{}
	count := 20
	if effect.effectID == effectBash3D3 || effect.effectID == effectBash3D4 {
		count = 24
	}
	for i := 0; i < count; i++ {
		yaw := degreesToRadians(float64(90*(i%4) + 22*(i/4)))
		roll := deterministicAngle(effect, i)
		point := func(angle float64) modelPoint3 {
			r := distance * math.Cos(angle)
			return add3(origin, modelPoint3{x: r * math.Cos(roll), y: -r * math.Sin(roll), z: distance * math.Sin(angle)})
		}
		for layer, width := range []float64{2, 5} {
			tint := color.RGBA{0, 250, 250, uint8(alpha)}
			if layer == 1 {
				tint = color.RGBA{250, 0, 0, uint8(alpha)}
			}
			if effect.effectID == effectBash3D3 {
				tint = color.RGBA{0, 0, 250, uint8(alpha)}
				if layer == 1 {
					tint = color.RGBA{250, 250, 0, uint8(alpha)}
				}
			} else if effect.effectID == effectBash3D4 {
				tint = color.RGBA{50, 50, 50, uint8(alpha)}
				if layer == 1 {
					tint = color.RGBA{250, 250, 250, uint8(alpha)}
				}
			}
			a, b := point(yaw-degreesToRadians(width)), point(yaw+degreesToRadians(width))
			appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{origin, origin, b, a}, tint)
			if count == 24 {
				// Head Crush and Joint Beat use alpha blending for the wider ray.
				screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(layer == 0, false))
				vertices, indices = nil, nil
			}
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
	}
}

// The native ring primitives have a height at each of their 21 vertices.
func appendPaladinBand(vertices *[]render.Vertex3D, indices *[]uint16, texture *render.Image, origin modelPoint3, radius, angle, sweep, flare float64, segments int, tint color.RGBA, height func(int) float64) {
	point := func(i int) (modelPoint3, modelPoint3) {
		a, h := angle+sweep*float64(i)/float64(segments), height(i)
		c, s := math.Cos(a), math.Sin(a)
		bottom := add3(origin, modelPoint3{x: radius * c, z: radius * s})
		top := add3(origin, modelPoint3{x: (radius + h*flare) * c, y: h, z: (radius + h*flare) * s})
		return bottom, top
	}
	for i := 0; i < segments; i++ {
		b1, t1 := point(i)
		b2, t2 := point(i + 1)
		start := len(*vertices)
		appendEffectQuad(vertices, indices, texture, [4]modelPoint3{t1, t2, b2, b1}, tint)
		for j, u := range []float32{float32(i) / 21, float32(i+1) / 21, float32(i+1) / 21, float32(i) / 21} {
			(*vertices)[start+j].SrcX = u * float32(texture.Bounds().Dx())
		}
	}
}

func (m *WorldMode) drawCrusaderDefenseEffect(screen *render.Frame, ctx client.Context, _ sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	alpha := min(n, 80) - 5*max(0, n-179)
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if alpha <= 0 || texture == nil {
		return
	}
	origin := effectWorldAnchor(ctx, effect, now)
	vertices, indices := []render.Vertex3D{}, []uint16{}
	for i := 0; i < 4; i++ {
		rotationSpeed, waveSpeed := 1+i%2, 1+i/2
		height := float64(40-i) / 5 * (1 + .3*math.Sin(degreesToRadians(float64(n*waveSpeed+i*90))))
		appendPaladinBand(&vertices, &indices, texture, origin, (8-.1*float64(i))/5,
			degreesToRadians(float64(180+90*i+n*rotationSpeed)), 2*math.Pi, 0, 20, color.RGBA{255, 255, 255, uint8(alpha)},
			func(j int) float64 { return height * math.Sin(degreesToRadians(float64(j*9))) })
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(false, false))
}

func (m *WorldMode) drawProvidenceEffect(screen *render.Frame, ctx client.Context, _ sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	alpha := 40 - max(0, n-50)
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if alpha <= 0 || texture == nil {
		return
	}
	origin := effectWorldAnchor(ctx, effect, now)
	vertices, indices := []render.Vertex3D{}, []uint16{}
	for i := 0; i < 4; i++ {
		rise := degreesToRadians(float64(83 - i))
		height := 24 * math.Sin(degreesToRadians(float64(min(n, 90)))) * math.Sin(rise)
		appendPaladinBand(&vertices, &indices, texture, origin, (6+.5*float64(i))/5,
			degreesToRadians(float64(90*i+n)), 2*math.Pi, 1/math.Tan(rise), 20, color.RGBA{255, 255, 255, uint8(alpha)},
			func(int) float64 { return height })
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
}

func (m *WorldMode) drawGrandCrossEffect(screen *render.Frame, ctx client.Context, _ sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	origin := effect.actorOrigin
	if texture := m.effectFileTexture(ctx.Resources, component.textureFile); texture != nil {
		alpha := min(n, 9)*10 - max(0, n-9)
		if alpha > 0 {
			vertices, indices := []render.Vertex3D{}, []uint16{}
			for i, offset := range []modelPoint3{{x: 4.6, z: 4.6}, {x: 4.6, z: -4.6}, {x: -4.6, z: -4.6}, {x: -4.6, z: 4.6}} {
				height := 24 * math.Sin(degreesToRadians(float64(n)))
				appendPaladinBand(&vertices, &indices, texture, add3(origin, offset), (19.9-.01*float64(n))/5,
					degreesToRadians(float64(180-90*i)), degreesToRadians(80), 0, 8, color.RGBA{255, 175, 175, uint8(alpha)}, func(int) float64 { return height })
			}
			screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
		}
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFiles[0])
	alpha := min(n, 9)*9 - max(0, n-9)
	if texture == nil || alpha <= 0 {
		return
	}
	vertices, indices := []render.Vertex3D{}, []uint16{}
	floorVertices, floorIndices := []render.Vertex3D{}, []uint16{}
	length, width := (24+.005*float64(n))/5, (3.2+.005*float64(n))/5
	height := 12 * math.Sin(degreesToRadians(float64(n)))
	for arm := 0; arm < 2; arm++ {
		corners := [4]modelPoint3{{x: -length, z: -width}, {x: length, z: -width}, {x: length, z: width}, {x: -length, z: width}}
		for i := range corners {
			if arm == 1 {
				corners[i].x, corners[i].z = -corners[i].z, corners[i].x
			}
			corners[i] = add3(origin, corners[i])
			corners[i].y += .2
		}
		for repeat := 0; repeat < 2; repeat++ {
			appendEffectQuad(&floorVertices, &floorIndices, texture, corners, color.RGBA{255, 255, 255, uint8(alpha / 2)})
		}
		for i, bottom := range corners {
			next := corners[(i+1)%4]
			appendEffectQuad(&vertices, &indices, texture,
				[4]modelPoint3{add3(bottom, modelPoint3{y: height}), add3(next, modelPoint3{y: height}), next, bottom},
				color.RGBA{255, 255, 255, uint8(alpha)})
		}
	}
	screen.DrawTriangles3DOwned(floorVertices, floorIndices, texture, texturedEffectBillboardDrawOptions(true, false))
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(false, false))
}

func (m *WorldMode) applyActorDevotion(ctx client.Context, notify network.ActorDevotion) {
	active := m.worldEffects[:0]
	for _, effect := range m.worldEffects {
		if effect.effectID != effectLineLink || effect.actorID != notify.SourceID {
			active = append(active, effect)
		}
	}
	m.worldEffects = active
	now := time.Now()
	for i, targetID := range notify.Targets {
		if targetID == 0 || targetID == notify.SourceID {
			continue
		}
		duplicate := false
		for _, previous := range notify.Targets[:i] {
			duplicate = duplicate || previous == targetID
		}
		if duplicate {
			continue
		}
		if m.addWorldEffectBetweenAt(ctx, effectLineLink, notify.SourceID, targetID, now) {
			m.worldEffects[len(m.worldEffects)-1].expires = time.Time{}
		}
	}
}

func (m *WorldMode) drawDevotionLinkEffect(screen *render.Frame, ctx client.Context, _ sceneProjection, component worldEffectComponent, effect worldEffect, now time.Time) {
	if _, ok, _ := actorForCombatID(ctx, effect.targetID); !ok {
		return
	}
	texture := m.effectFileTexture(ctx.Resources, component.textureFile)
	if texture == nil {
		return
	}
	source := effectWorldAnchor(ctx, effect, now)
	target := effectWorldAnchor(ctx, worldEffect{actorID: effect.targetID}, now)
	source.y += 1.6
	target.y += 1.6
	n := int(now.Sub(effect.starts)/db.EffectFrameDuration) + 1
	target = add3(source, mul3(sub3(target, source), math.Sin(degreesToRadians(float64(min(n, 15)*6)))))
	angle := math.Atan2(target.z-source.z, target.x-source.x) + math.Pi/2
	pulse := n % 10
	alpha := 45 + 2*min(pulse, 4) - 2*max(0, pulse-4)
	vertices, indices := []render.Vertex3D{}, []uint16{}
	for layer := 0; layer < 2; layer++ {
		radius := .08 * float64(layer+1)
		tint := color.RGBA{255, 255, 0, uint8(alpha)}
		if layer == 1 {
			tint = color.RGBA{0, 0, 255, uint8(alpha * 2)}
		}
		offset := func(j int) modelPoint3 {
			a := float64(j) * math.Pi / 5
			r := radius * math.Cos(a)
			return modelPoint3{x: r * math.Cos(angle), y: radius * math.Sin(a), z: r * math.Sin(angle)}
		}
		for j := 0; j < 10; j++ {
			a, b := offset(j), offset(j+1)
			appendEffectQuad(&vertices, &indices, texture, [4]modelPoint3{
				add3(target, a), add3(target, b), add3(source, b), add3(source, a),
			}, tint)
		}
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, texturedEffectBillboardDrawOptions(true, false))
}
