// Copyright 2022 Changkun Ou <changkun.de>. All rights reserved.
// Use of this source code is governed by a GPLv3 license that
// can be found in the LICENSE file.

package geometry

import (
	"poly.red/geometry/primitive"
	"poly.red/math"
)

type BezierCurve struct {
	controlPoints []primitive.Vertex
}

func NewBezierCurve[T math.Float](cp ...*primitive.Vertex) *BezierCurve {
	bc := &BezierCurve{
		controlPoints: make([]primitive.Vertex, len(cp)),
	}
	for i := range cp {
		bc.controlPoints[i] = *cp[i]
	}
	return bc
}

func (bc *BezierCurve) At(t float32) math.Vec4[float32] {
	n := len(bc.controlPoints)

	tc := make([]math.Vec4[float32], n)
	for i := range bc.controlPoints {
		tc[i] = bc.controlPoints[i].Pos
	}

	// The de Casteljau algorithm.
	for j := 0; j < n; j++ {
		for i := 0; i < n-j-1; i++ {
			b01 := math.LerpVec4(tc[i], tc[i+1], t)
			tc[i].X = b01.X
			tc[i].Y = b01.Y
		}
	}
	return tc[0]
}
