// Copyright 2021 Changkun Ou <changkun.de>. All rights reserved.
// Use of this source code is governed by a GPLv3 license that
// can be found in the LICENSE file.

package primitive_test

import (
	"testing"

	"changkun.de/x/ddd/geometry/primitive"
	"changkun.de/x/ddd/math"
)

func TestNewAABB(t *testing.T) {
	v1 := math.NewVector(1, 0, 0, 1)
	v2 := math.NewVector(0, 1, 0, 1)
	v3 := math.NewVector(0, 0, 1, 1)

	aabb := primitive.NewAABB(v1, v2, v3)

	if !aabb.Min.Eq(math.NewVector(0, 0, 0, 1)) {
		t.Fatal("not equal")
	}
	if !aabb.Max.Eq(math.NewVector(1, 1, 1, 1)) {
		t.Fatal("not equal")
	}

}

func TestAABB_Intersect(t *testing.T) {

	v1 := math.NewVector(1, 0, 0, 1)
	v2 := math.NewVector(0, 1, 0, 1)
	v3 := math.NewVector(0, 0, 1, 1)

	aabb1 := primitive.NewAABB(v1, v2, v3)

	v4 := math.NewVector(-1, -0.5, -0.5, 1)
	v5 := math.NewVector(-0.5, -1, -0.5, 1)
	v6 := math.NewVector(-0.5, -0.5, -1, 1)

	aabb2 := primitive.NewAABB(v4, v5, v6)

	if aabb1.Intersect(aabb2) {
		t.Fatalf("intersect")
	}
	v7 := math.NewVector(0.5, 0, 0, 1)
	v8 := math.NewVector(0, 0.5, 0, 1)
	v9 := math.NewVector(0, 0, 0.5, 1)

	aabb3 := primitive.NewAABB(v7, v8, v9)

	if !aabb1.Intersect(aabb3) {
		t.Fatalf("not intersect")
	}

	v10 := math.NewVector(-1, 0, 0, 1)
	v11 := math.NewVector(0, -1, 0, 1)
	v12 := math.NewVector(0, 0, -1, 1)

	aabb4 := primitive.NewAABB(v10, v11, v12)

	if !aabb1.Intersect(aabb4) {
		t.Fatalf("not intersect")
	}
}

func TestAABB_Add(t *testing.T) {

	v1 := math.NewVector(1, 0, 0, 1)
	v2 := math.NewVector(0, 1, 0, 1)
	v3 := math.NewVector(0, 0, 1, 1)

	aabb := primitive.NewAABB(v1, v2, v3)

	v4 := math.NewVector(-1, -0.5, -0.5, 1)
	v5 := math.NewVector(-0.5, -1, -0.5, 1)
	v6 := math.NewVector(-0.5, -0.5, -1, 1)

	aabb.Add(primitive.NewAABB(v4, v5, v6))
	want := primitive.NewAABB(v1, v2, v3, v4, v5, v6)
	if !aabb.Eq(want) {
		t.Fatalf("AABB add does not work")
	}
}

func BenchmarkVertexAABB(b *testing.B) {
	v := primitive.NewRandomVertex()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.AABB()
	}
}
