// Copyright 2021 Changkun Ou <changkun.de>. All rights reserved.
// Use of this source code is governed by a GPLv3 license that
// can be found in the LICENSE file.

package render_test

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"

	"poly.red/camera"
	"poly.red/geometry/mesh"
	"poly.red/geometry/primitive"
	"poly.red/light"
	"poly.red/material"
	"poly.red/math"
	"poly.red/object"
	"poly.red/render"
	"poly.red/scene"
	"poly.red/texture"
	"poly.red/texture/imageutil"
)

var (
	s *scene.Scene
	c camera.Interface
	r *render.Renderer
)

func init() {
	w, h, msaa := 1920, 1080, 2
	s, c = newscene(w, h)
	r = render.NewRenderer(
		render.Size(w, h),
		render.MSAA(msaa),
		render.Scene(s),
		render.Background(color.RGBA{0, 127, 255, 255}),
	)
}

func newscene(w, h int) (*scene.Scene, camera.Interface) {
	s := scene.NewScene()

	s.Add(light.NewPoint(
		light.Intensity(5),
		light.Color(color.RGBA{0, 0, 0, 255}),
		light.Position(math.NewVec3(-2, 2.5, 6)),
	), light.NewAmbient(
		light.Intensity(0.5),
	))

	m, err := mesh.Load("../internal/testdata/bunny.obj")
	if err != nil {
		panic(err)
	}

	data := imageutil.MustLoadImage("../internal/testdata/bunny.png")
	mat := material.NewBlinnPhong(
		material.Texture(texture.NewTexture(
			texture.Image(data),
			texture.IsoMipmap(true),
		)),
		material.Kdiff(0.8), material.Kspec(1),
		material.Shininess(100),
		material.AmbientOcclusion(true),
	)
	m.SetMaterial(mat)
	m.Rotate(math.NewVec3(0, 1, 0), -math.Pi/6)
	m.Scale(4, 4, 4)
	m.Translate(0.1, 0, -0.2)
	s.Add(m)
	return s, camera.NewPerspective(
		camera.Position(math.NewVec3(0, 1.5, 1)),
		camera.LookAt(
			math.NewVec3(0, 0, -0.5),
			math.NewVec3(0, 1, 0),
		),
		camera.ViewFrustum(45, float64(w)/float64(h), 0.1, 3),
	)
}

func TestRender(t *testing.T) {
	w, h, msaa := 1920, 1080, 2
	s, c := newscene(w, h)
	r := render.NewRenderer(
		render.Camera(c),
		render.Size(w, h),
		render.MSAA(msaa),
		render.Scene(s),
		render.Background(color.RGBA{0, 127, 255, 255}),
	)

	f, err := os.Create("cpu.pprof")
	if err != nil {
		t.Fatal(err)
	}
	mem, err := os.Create("mem.pprof")
	if err != nil {
		panic(err)
	}

	var buf *image.RGBA
	pprof.StartCPUProfile(f)
	for i := 0; i < 10; i++ {
		buf = r.Render()
	}
	pprof.StopCPUProfile()
	runtime.GC()
	pprof.WriteHeapProfile(mem)
	mem.Close()
	f.Close()

	path := "../internal/testdata/render.png"
	fmt.Printf("render saved at: %s\n", path)
	imageutil.Save(buf, path)
}

func BenchmarkRasterizer(b *testing.B) {
	for block := 1; block <= 1024; block *= 2 {
		r.Options(render.BatchSize(int32(block)), render.Camera(c))
		b.Run(fmt.Sprintf("concurrent-size %d", block), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				r.Render()
			}
		})
	}
}

func BenchmarkForwardPass(b *testing.B) {
	for block := 1; block <= 1024; block *= 2 {
		r.Options(render.BatchSize(int32(block)), render.Camera(c))
		b.Run(fmt.Sprintf("concurrent-size %d", block), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				render.PassForward(r)
			}
		})
	}
}

func BenchmarkDeferredPass(b *testing.B) {
	for block := 1; block <= 1024; block *= 2 {
		r.Options(render.BatchSize(int32(block)), render.Camera(c))
		b.Run(fmt.Sprintf("concurrent-size %d", block), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				render.PassDeferred(r)
			}
		})
	}
}

func BenchmarkAntiAliasingPass(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		render.PassAntiAliasing(r)
	}
}

func BenchmarkAntiGammaCorrection(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		render.PassGammaCorrect(r)
	}
}

func BenchmarkDraw(b *testing.B) {
	for block := 1; block <= 1024; block *= 2 {
		matView := c.ViewMatrix()
		matProj := c.ProjMatrix()
		matVP := math.ViewportMatrix(1920, 1080)

		var (
			m        mesh.Mesh
			modelMat math.Mat4
		)
		s.IterObjects(func(o object.Object, modelMatrix math.Mat4) bool {
			if o.Type() == object.TypeMesh {
				m = o.(mesh.Mesh)
				modelMat = modelMatrix
				return false
			}
			return true
		})

		uniforms := map[string]interface{}{
			"matModel":   modelMat,
			"matView":    matView,
			"matViewInv": matView.Inv(),
			"matProj":    matProj,
			"matProjInv": matProj.Inv(),
			"matVP":      matVP,
			"matVPInv":   matVP.Inv(),
			"matNormal":  modelMat.Inv().T(),
		}

		b.Run(fmt.Sprintf("concurrent-size %d", block), func(b *testing.B) {
			var (
				ts  = []*primitive.Triangle{}
				mat material.Material
				nt  = m.NumTriangles()
			)

			m.Faces(func(f primitive.Face, m material.Material) bool {
				mat = m
				f.Triangles(func(t *primitive.Triangle) bool {
					ts = append(ts, t)
					return true
				})
				return true
			})

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				f := ts[i%int(nt)]
				render.Draw(r, uniforms, f, modelMat, mat)
			}
		})
	}
}
