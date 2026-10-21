package game

import (
	"image/color"
	"math"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
)

const (
	mapPillarBands          = 4
	mapPillarSegments       = 20
	mapPillarCycleFrames    = 960.0
	mapPillarGrowStartFrame = 200.0
	mapPillarGrowEndFrame   = 290.0
	mapPillarShrinkFrame    = 800.0
	mapPillarWorldScale     = 0.2
	mapPillarBandRadiusStep = 0.5 * mapPillarWorldScale
	mapPillarRiseRadians    = 89.0 * math.Pi / 180.0
)

func (m *WorldMode) drawMapPillarEffect(screen *render.Frame, ctx client.Context, component worldEffectComponent, x, y, z, progress float64) {
	texture := m.effectTexture(ctx.Resources, component.textureName)
	if texture == nil || component.height <= 0 || component.bottomSize <= 0 {
		return
	}
	tint := effectComponentTint(component, component.alphaMax)
	frames := clampFloat(progress, 0, 0.999999) * mapPillarCycleFrames
	for band := 0; band < mapPillarBands; band++ {
		process := frames + float64(band)*30
		height := mapPillarBandHeight(process, component.height)
		if height <= 0 {
			continue
		}
		radius := component.bottomSize + float64(band)*mapPillarBandRadiusStep
		rotation := float64(band)*90 + frames
		drawMapPillarBand(screen, texture, x, y, z, radius, height, rotation, tint)
	}
}

func mapPillarBandHeight(process, maxHeight float64) float64 {
	if process < mapPillarGrowStartFrame || maxHeight <= 0 {
		return 0
	}
	if process >= mapPillarShrinkFrame {
		maxHeight -= (process - (mapPillarShrinkFrame - 1)) * mapPillarWorldScale
		if maxHeight <= 0 {
			return 0
		}
	}
	if process <= mapPillarGrowEndFrame {
		return maxHeight * math.Sin((process-mapPillarGrowStartFrame)*math.Pi/180)
	}
	return maxHeight
}

func drawMapPillarBand(screen *render.Frame, texture *render.Image, x, y, z, radius, height, rotationDegrees float64, tint color.RGBA) {
	if screen == nil || texture == nil || radius <= 0 || height <= 0 || tint.A == 0 {
		return
	}
	bounds := texture.Bounds()
	srcW, srcH := float32(bounds.Dx()), float32(bounds.Dy())
	center := modelPoint3{x: x, y: z, z: y}
	radialRise := math.Cos(mapPillarRiseRadians) * height
	verticalRise := math.Sin(mapPillarRiseRadians) * height
	vertices := make([]render.Vertex3D, 0, (mapPillarSegments+1)*2)
	indices := make([]uint16, 0, mapPillarSegments*6)
	for i := 0; i <= mapPillarSegments; i++ {
		u := float32(i) / float32(mapPillarSegments)
		angle := (rotationDegrees + float64(i)*360/float64(mapPillarSegments)) * math.Pi / 180
		cosine := math.Cos(angle)
		sine := math.Sin(angle)
		base := modelPoint3{
			x: center.x + radius*cosine,
			y: center.y,
			z: center.z + radius*sine,
		}
		top := modelPoint3{
			x: center.x + (radius+radialRise)*cosine,
			y: center.y + verticalRise,
			z: center.z + (radius+radialRise)*sine,
		}
		vertices = append(vertices,
			texturedSurfaceVertex3D(base, texturePoint{u: u, v: 1}, tint, srcW, srcH),
			texturedSurfaceVertex3D(top, texturePoint{u: u, v: 0}, tint, srcW, srcH),
		)
		if i == mapPillarSegments {
			continue
		}
		baseIndex := uint16(i * 2)
		indices = append(indices, baseIndex, baseIndex+1, baseIndex+2, baseIndex+1, baseIndex+3, baseIndex+2)
	}
	options := triangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	options.Blend = render.BlendLighter
	screen.DrawTriangles3D(vertices, indices, texture, options)
}
