package game

import (
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

const rsmModelDepthBias = scene.GroundDecalDepthBias

const rsmHorizontalDepthBiasLayers = 64

const rsmHorizontalDepthBiasStep = scene.GroundDecalDepthBias / 256

func worldOpaqueTriangleDrawOptions(filter render.Filter, address render.Address) *render.DrawTrianglesOptions {
	options := scene.TriangleDrawOptions(filter, address)
	options.DepthWrite = true
	return options
}

func groundTextureDrawOptions() *render.DrawTrianglesOptions {
	return worldOpaqueTriangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
}

func rsmModelDrawOptions(filter render.Filter, address render.Address) *render.DrawTrianglesOptions {
	options := worldOpaqueTriangleDrawOptions(filter, address)
	options.DepthBias = rsmModelDepthBias
	return options
}

func rsmHorizontalSurfaceDrawOptions(filter render.Filter, address render.Address, placementIndex int) *render.DrawTrianglesOptions {
	options := rsmModelDrawOptions(filter, address)
	if placementIndex < 0 {
		return options
	}
	layer := placementIndex % rsmHorizontalDepthBiasLayers
	options.DepthBias += float32(layer) * rsmHorizontalDepthBiasStep
	return options
}
