// Copyright 2021 Changkun Ou <changkun.de>. All rights reserved.
// Use of this source code is governed by a GPLv3 license that
// can be found in the LICENSE file.

package material

import (
	"image/color"

	"poly.red/geometry/primitive"
	"poly.red/light"
	"poly.red/math"
	"poly.red/texture"
)

// Material is an interface that represents a mesh material
type Material interface {
	ReceiveShadow() bool
	AmbientOcclusion() bool
	Texture() *texture.Texture
	VertexShader(
		v primitive.Vertex,
		uniforms map[string]interface{},
	) primitive.Vertex
	FragmentShader(col color.RGBA, x, n, fn, camera math.Vec4, ls []light.Source, es []light.Environment) color.RGBA
}
