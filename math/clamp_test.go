// Copyright 2022 The Polyred Authors. All rights reserved.
// Use of this source code is governed by a GPLv3 license that
// can be found in the LICENSE file.

package math_test

import (
	"testing"

	"poly.red/math"
)

func TestClamp(t *testing.T) {
	t.Run("float64", func(t *testing.T) {
		got := math.Clamp[float64](128, 0, 255)
		if got != 128 {
			t.Fatalf("unexpected clamp, got %v, want 128", got)
		}

		got = math.Clamp[float64](-1, 0, 255)
		if got != 0 {
			t.Fatalf("unexpected clamp, got %v, want 0", got)
		}

		got = math.Clamp[float64](256, 0, 255)
		if got != 255 {
			t.Fatalf("unexpected clamp, got %v, want 255", got)
		}
	})

	t.Run("int", func(t *testing.T) {
		got := math.Clamp(128, 0, 255)
		if got != 128 {
			t.Fatalf("unexpected clamp, got %v, want 128", got)
		}

		got = math.Clamp(-1, 0, 255)
		if got != 0 {
			t.Fatalf("unexpected clamp, got %v, want 0", got)
		}

		got = math.Clamp(256, 0, 255)
		if got != 255 {
			t.Fatalf("unexpected clamp, got %v, want 255", got)
		}
	})

	t.Run("Vec4", func(t *testing.T) {
		v := math.Vec4[float32]{128, 128, 128, 128}
		want := math.Vec4[float32]{128, 128, 128, 128}
		got := math.ClampVec4(v, 0, 255)
		if got != want {
			t.Fatalf("unexpected clamp, got %v, want %v", got, want)
		}

		v = math.Vec4[float32]{-1, -1, -1, -1}
		want = math.Vec4[float32]{0, 0, 0, 0}
		got = math.ClampVec4(v, 0, 255)
		if got != want {
			t.Fatalf("unexpected clamp, got %v, want %v", got, want)
		}

		v = math.Vec4[float32]{256, 266, 265, 256}
		want = math.Vec4[float32]{255, 255, 255, 255}
		got = math.ClampVec4(v, 0, 255)
		if got != want {
			t.Fatalf("unexpected clamp, got %v, want 2%v55", got, want)
		}
	})

	t.Run("Vec3", func(t *testing.T) {
		v := math.Vec3[float32]{128, 128, 128}
		want := math.Vec3[float32]{128, 128, 128}
		got := math.ClampVec3(v, 0, 255)
		if got != want {
			t.Fatalf("unexpected clamp, got %v, want %v", got, want)
		}

		v = math.Vec3[float32]{-1, -1, -1}
		want = math.Vec3[float32]{0, 0, 0}
		got = math.ClampVec3(v, 0, 255)
		if got != want {
			t.Fatalf("unexpected clamp, got %v, want %v", got, want)
		}

		v = math.Vec3[float32]{256, 266, 265}
		want = math.Vec3[float32]{255, 255, 255}
		got = math.ClampVec3(v, 0, 255)
		if got != want {
			t.Fatalf("unexpected clamp, got %v, want 2%v55", got, want)
		}
	})

	t.Run("Vec2", func(t *testing.T) {
		v := math.Vec2[float32]{128, 128}
		want := math.Vec2[float32]{128, 128}
		got := math.ClampVec2(v, 0, 255)
		if got != want {
			t.Fatalf("unexpected clamp, got %v, want %v", got, want)
		}

		v = math.Vec2[float32]{-1, -1}
		want = math.Vec2[float32]{0, 0}
		got = math.ClampVec2(v, 0, 255)
		if got != want {
			t.Fatalf("unexpected clamp, got %v, want %v", got, want)
		}

		v = math.Vec2[float32]{256, 256}
		want = math.Vec2[float32]{255, 255}
		got = math.ClampVec2(v, 0, 255)
		if got != want {
			t.Fatalf("unexpected clamp, got %v, want 2%v55", got, want)
		}
	})
}

func BenchmarkClamp(b *testing.B) {
	v := float32(128.0)

	var bb float32
	for i := 0; i < b.N; i++ {
		bb = math.Clamp(v, 0, 255)
	}
	_ = bb
}

func BenchmarkClampInt(b *testing.B) {
	v := 128

	var bb int
	for i := 0; i < b.N; i++ {
		bb = math.Clamp(v, 0, 255)
	}
	_ = bb
}

func BenchmarkClampVec(b *testing.B) {
	b.Run("Vec4", func(b *testing.B) {
		v := math.Vec4[float32]{128, 128, 128, 255}

		var bb math.Vec4[float32]
		for i := 0; i < b.N; i++ {
			bb = math.ClampVec4(v, 0, 255)
		}
		_ = bb
	})

	b.Run("Vec3", func(b *testing.B) {
		v := math.Vec3[float32]{128, 128, 128}

		var bb math.Vec3[float32]
		for i := 0; i < b.N; i++ {
			bb = math.ClampVec3(v, 0, 255)
		}
		_ = bb
	})

	b.Run("Vec2", func(b *testing.B) {
		v := math.Vec2[float32]{128, 128}

		var bb math.Vec2[float32]
		for i := 0; i < b.N; i++ {
			bb = math.ClampVec2(v, 0, 255)
		}
		_ = bb
	})
}
