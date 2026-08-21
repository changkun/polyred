# polyred [![Go Reference](https://pkg.go.dev/badge/poly.red.svg)](https://pkg.go.dev/poly.red) [![polyred](https://github.com/polyred/polyred/actions/workflows/polyred.yml/badge.svg?branch=main)](https://github.com/polyred/polyred/actions/workflows/polyred.yml) ![](https://changkun.de/urlstat?mode=github&repo=polyred/polyred)

3D graphics in Go.

```go
import "poly.red"
```

_Warning: under experiment, expect to break at anytime._

## GPU compute and rendering, in Go

`poly.red/gpu` is a backend-agnostic GPU abstraction, a WebGPU-style
`Device`/`Queue`/`Buffer`/`Pipeline`/`CommandEncoder` API for running **compute**
and **rendering** pipelines, with swappable drivers underneath (Metal, OpenGL
ES, Vulkan today). It is **cgo-free**: the Metal/Objective-C runtime, EGL/GLES
and the Vulkan loader are all reached through
[ebitengine/purego](https://github.com/ebitengine/purego), so it builds with
`CGO_ENABLED=0`.

Shaders are written **in Go** and compiled to the backend's shading language by
`poly.red/gpu/shader`: compute, vertex, and fragment kernels, with varyings,
uniforms, vector math, and control flow.

```go
dev, _ := gpu.Open()                 // Metal on darwin
defer dev.Close()

// Shaders authored in Go, compiled to MSL.
ks, _ := shader.Compile(`
package kernels
type Vec4 struct{ X, Y, Z, W float32 }
type VOut struct {
    Pos   Vec4 ` + "`gpu:\"position\"`" + `
    Color Vec4
}
//gpu:vertex
func VMain(vid uint, pos []float32, col []float32) VOut {
    return VOut{Vec4{pos[vid*2], pos[vid*2+1], 0, 1}, Vec4{col[vid*3], col[vid*3+1], col[vid*3+2], 1}}
}
//gpu:fragment
func FMain(in VOut) Vec4 { return in.Color }
`)
// ... build a render pipeline, render to a texture, read pixels back.
```

See the runnable end-to-end example:

```sh
go run ./cmd/gpudemo -o triangle.png    # Go shaders -> Metal -> PNG, cgo-free
```

The design, decisions, and roadmap live in
[`docs/gpu-abstraction.md`](docs/gpu-abstraction.md); implementation specs are in
[`specs/`](specs/README.md).

### Status

| Capability | State |
| --- | --- |
| `Device` API (buffers, bind groups, compute + render pipelines, passes, textures, samplers) | working |
| Metal backend (compute + render), cgo-free via purego | working |
| Go→shader compiler (compute + vertex/fragment, varyings, uniforms, vector + matrix math, swizzle, texture sampling, trig, control flow) | working |
| **Renderer deferred pass fully offloaded to the GPU**: point + directional lights, multi-material, shadow maps (one and many casting lights), ambient occlusion, gamma | working, CPU-parity verified |
| **Renderer forward rasterizer on the GPU** (vertex transform, back-face cull, depth test, three-target G-buffer), the default pass on GL and Metal | working, parity-gated |
| **OpenGL ES + Vulkan backends** (cgo-free, via purego): compute through the Device API verified in CI on Mesa (llvmpipe / lavapipe, software, headless); GL also does render-to-texture, depth + MRT, and on-screen window surfaces | working |
| **On-screen windowed present** through the Device/Surface API: X11 (Xvfb + Mesa) and Win32 (ANGLE) proven on screen in CI; darwin drives the same present path in a test, but offscreen (no `CAMetalLayer` drawable yet) | working |
| DirectX 12 backend, Vulkan render path | planned (the device and compute probes are green in CI) |

The renderer is GPU by default: `NewRenderer` acquires a device itself (Metal
today) and falls back to the CPU per pass if that fails. Pass `render.GPU(dev)`
to hand it a device of your own, or `render.CPU()` to force the CPU path:

```go
dev, _ := gpu.Open()
img := render.NewRenderer(
    render.Camera(cam), render.Size(w, h), render.Scene(s),
    render.ShadowMap(true),
    render.GPU(dev), // forward raster + deferred shading run on this device
).Render()
```

cgo-free build/test of the Metal GPU stack on darwin:

```sh
CGO_ENABLED=0 go test ./gpu ./gpu/mtl ./gpu/shader ./gpu/tests
```

CI runs build + vet + tests on macOS, Linux, and Windows, plus dedicated jobs
for the GL, Vulkan and D3D12 backends and for the X11 and Win32 windowed
present; all green. Per-spec status is in
[`specs/README.md`](specs/README.md).
