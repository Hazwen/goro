package testutil

import (
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func EffectRenderTestSprite() *scene.SpriteView {
	return &scene.SpriteView{
		Spr: &res.SPR{},
		Act: &res.ACT{Actions: []res.ACTAction{{DelayMS: 100, Animations: []res.ACTAnimation{{}}}}},
		Billboards: map[scene.SpriteBillboardKey]*scene.Billboard{
			{}: {Image: render.NewImage(8, 8), AnchorX: 4, AnchorY: 8},
		},
	}
}
