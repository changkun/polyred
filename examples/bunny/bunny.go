// Copyright 2021 Changkun Ou <changkun.de>. All rights reserved.
// Use of this source code is governed by a GPLv3 license that
// can be found in the LICENSE file.

package bunny

import (
	"image/color"

	"poly.red/camera"
	"poly.red/geometry/mesh"
	"poly.red/light"
	"poly.red/material"
	"poly.red/math"
	"poly.red/scene"
	"poly.red/texture"
	"poly.red/utils"
)

func NewBunnyScene(width, height int) interface{} {
	s := scene.NewScene()
	s.SetCamera(camera.NewPerspective(
		camera.WithPosition(math.NewVec3(-550, 194, 734)),
		camera.WithLookAt(
			math.NewVec3(-1000, 0, 0),
			math.NewVec3(0, 1, 1),
		),
		camera.WithPerspFrustum(
			45,
			float64(width)/float64(height),
			100, 600,
		),
	))
	s.Add(light.NewPoint(
		light.WithPointLightIntensity(200),
		light.WithPointLightColor(color.RGBA{255, 255, 255, 255}),
		light.WithPointLightPosition(math.NewVec3(-200, 250, 600)),
	), light.NewAmbient(
		light.WithAmbientIntensity(0.7),
	))

	var done func()

	// load a mesh
	done = utils.Timed("loading mesh")
	m, err := mesh.Load("../testdata/bunny-smooth.obj")
	if err != nil {
		panic(err)
	}
	done()

	done = utils.Timed("loading texture")
	data := texture.MustLoadImage("../testdata/bunny.png")
	tex := texture.NewTexture(
		texture.WithSource(data),
		texture.WithIsotropicMipMap(true),
	)
	done()

	mat := material.NewBlinnPhong(
		material.WithBlinnPhongTexture(tex),
		material.WithBlinnPhongFactors(0.6, 1),
		material.WithBlinnPhongShininess(150),
		material.WithBlinnPhongFlatShading(true),
	)
	m.SetMaterial(mat)
	m.Scale(1500, 1500, 1500)
	m.Translate(-700, -5, 350)
	s.Add(m)

	return s
}
