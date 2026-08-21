// Copyright 2026 The Polyred Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shader_test

import (
	"strings"
	"testing"

	"poly.red/gpu/shader"
	"poly.red/gpu/shader/gpumath/kernels"
)

// helperSrc exercises the shape brick 2 needs: a helper calling another helper,
// truncation via Trunc, and a kernel calling into them. See
// specs/foundations/gpu-material-texture-sampling.md.
const helperSrc = `
package kernels

//gpu:helper
func lerp8(a float32, b float32, t float32) float32 {
	return Trunc(a + (b-a)*t)
}

//gpu:helper
func bilerp8(p1 float32, p2 float32, p3 float32, p4 float32, fx float32, fy float32) float32 {
	i1 := lerp8(p1, p2, fx)
	i2 := lerp8(p3, p4, fx)
	return lerp8(i1, i2, fy)
}

func HelperSample(gid uint, a []float32, out []float32) {
	b := gid * 6
	out[gid] = bilerp8(a[b], a[b+1], a[b+2], a[b+3], a[b+4], a[b+5])
}
`

// TestHelperNotAnEntryPoint is the discriminating check for helper lowering.
// Before //gpu:helper existed, compileAll turned every top-level func into its
// own entry point, so this source produced three kernels (and the helpers failed
// to compile as kernels at all, since they return a value and take non-buffer
// params). Exactly one kernel must come back now.
func TestHelperNotAnEntryPoint(t *testing.T) {
	for _, tc := range []struct {
		name    string
		compile func(string) (map[string]*shader.Kernel, error)
	}{
		{"MSL", shader.Compile},
		{"GLSL", shader.CompileGLSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ks, err := tc.compile(helperSrc)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if len(ks) != 1 {
				var names []string
				for n := range ks {
					names = append(names, n)
				}
				t.Fatalf("got %d kernels %v, want exactly 1 (HelperSample); helpers must not be entry points", len(ks), names)
			}
			if _, ok := ks["HelperSample"]; !ok {
				t.Fatalf("entry point HelperSample missing")
			}
			for _, h := range []string{"lerp8", "bilerp8"} {
				if _, ok := ks[h]; ok {
					t.Errorf("helper %q was emitted as an entry point", h)
				}
			}
		})
	}
}

// TestHelperEmittedBeforeCaller checks the helper bodies land in the kernel's
// module ahead of the entry point that calls them: both MSL and GLSL require a
// function to be defined (or declared) before use.
func TestHelperEmittedBeforeCaller(t *testing.T) {
	msl, err := shader.Compile(helperSrc)
	if err != nil {
		t.Fatalf("compile MSL: %v", err)
	}
	src := msl["HelperSample"].MSL
	iLerp := strings.Index(src, "static float lerp8(")
	iBi := strings.Index(src, "static float bilerp8(")
	iKernel := strings.Index(src, "kernel void HelperSample(")
	if iLerp < 0 || iBi < 0 || iKernel < 0 {
		t.Fatalf("missing definitions in MSL:\n%s", src)
	}
	if !(iLerp < iBi && iBi < iKernel) {
		t.Errorf("MSL order lerp8=%d bilerp8=%d kernel=%d, want each helper before its caller", iLerp, iBi, iKernel)
	}
	if !strings.Contains(src, "trunc(") {
		t.Errorf("Trunc did not lower to trunc():\n%s", src)
	}

	glsl, err := shader.CompileGLSL(helperSrc)
	if err != nil {
		t.Fatalf("compile GLSL: %v", err)
	}
	g := glsl["HelperSample"].GLSL
	gLerp := strings.Index(g, "float lerp8(")
	gMain := strings.Index(g, "void main()")
	if gLerp < 0 || gMain < 0 {
		t.Fatalf("missing definitions in GLSL:\n%s", g)
	}
	if gLerp > gMain {
		t.Errorf("GLSL emits lerp8 after main(); helpers must precede their caller")
	}
	if strings.Contains(g, "static ") {
		t.Errorf("GLSL must not carry MSL's static keyword:\n%s", g)
	}
}

// TestExistingKernelsUnaffectedByHelpers guards the blast radius: helper support
// changed entry-point selection for every kernel in the tree, so the kernels that
// declare no helper must still compile to exactly one entry point with no
// prelude. Byte-identity of their generated source was verified against a
// pre-change dump when this landed; this keeps the shape locked.
func TestExistingKernelsUnaffectedByHelpers(t *testing.T) {
	for name, src := range map[string]string{
		"Shade": kernels.ShadeSrc, "SRGB": kernels.SRGBSrc,
		"Shadow": kernels.ShadowSrc, "AO": kernels.AOSrc,
	} {
		t.Run(name, func(t *testing.T) {
			msl, err := shader.Compile(src)
			if err != nil {
				t.Fatalf("MSL: %v", err)
			}
			if len(msl) != 1 {
				t.Fatalf("MSL: got %d kernels, want 1", len(msl))
			}
			glsl, err := shader.CompileGLSL(src)
			if err != nil {
				t.Fatalf("GLSL: %v", err)
			}
			if len(glsl) != 1 {
				t.Fatalf("GLSL: got %d kernels, want 1", len(glsl))
			}
			for _, k := range msl {
				if strings.Contains(k.MSL, "static ") {
					t.Errorf("MSL gained a helper prelude it never declared:\n%s", k.MSL)
				}
			}
		})
	}
}

// TestHelperRejectsBufferParam pins the documented limit: GLSL ES 3.1 cannot pass
// an SSBO block to a function, so buffer indexing stays in the entry point.
func TestHelperRejectsBufferParam(t *testing.T) {
	const src = `
package kernels

//gpu:helper
func fetch(a []float32, i int) float32 { return a[i] }

func K(gid uint, a []float32, out []float32) { out[gid] = fetch(a, int(gid)) }
`
	if _, err := shader.Compile(src); err == nil {
		t.Fatal("want an error for a helper taking a storage buffer, got nil")
	} else if !strings.Contains(err.Error(), "scalar or vector") {
		t.Fatalf("error should name the restriction, got: %v", err)
	}
}
