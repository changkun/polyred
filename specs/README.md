# polyred specs

Design specs for non-trivial work, written **before** implementation. Each spec
captures the problem, current state, architecture, and testing strategy so the
implementation has a reviewable target.

Workflow: write/iterate a spec here → implement against it → diff the result
back. The high-level GPU architecture and locked decisions live in
[`docs/gpu-abstraction.md`](../docs/gpu-abstraction.md); per-phase implementation
specs live here.

## Tracks

- **foundations**: core abstraction interfaces the rest of the engine builds on
  (e.g. the GPU `Device` abstraction).

## foundations

| Spec | Status | Deliverable |
| --- | --- | --- |
| [gpu-phase1-foundation.md](foundations/gpu-phase1-foundation.md) | **Done** | cgo-free Metal compute via purego, the `Device` API, and the matrix demo through it |
| [gpu-phase2-goshader.md](foundations/gpu-phase2-goshader.md) | **Done** | Go→shader compiler (compute + vertex/fragment → MSL): varyings, uniforms, swizzle, vector/matrix math, texture sampling, trig |
| [gpu-phase3-render.md](foundations/gpu-phase3-render.md) | **Done** | Render pipelines + the renderer's full deferred pass offloaded to the GPU: lights, multi-material, shadow maps (N lights), ambient occlusion, gamma; CPU-parity verified |
| [windows-present-port.md](foundations/windows-present-port.md) | **Done, runtime CI-proven** | Windows window present on the modern textured-quad GLES blit; `TestWin32WindowedPresent` creates a real HWND, presents frames across a resize and reads them back, run on every push by the dedicated `windows-present` job (ANGLE on WARP, `POLYRED_REQUIRE_WINDOW=1` so a skip fails the job) |
| [gpu-gl-backend.md](foundations/gpu-gl-backend.md) | **Done, CI-verified, engine-integrated** | cgo-free GLES 3.1 backend behind the `backend` interface: compute (storage + UBO), render-to-texture (FBO), depth + MRT, and on-screen window surfaces, all through the Device API and verified on Mesa llvmpipe in CI; the renderer's forward and deferred passes are exercised on it there (`TestGPUForward`, `TestGLDeferredRender`). Follow-up: the raster vertex/fragment shaders are still hand-written GLSL/MSL rather than authored once in Go |
| [gpu-windowed-present.md](foundations/gpu-windowed-present.md) | **Done, CI-verified on screen** | backend-agnostic swapchain (`gpu/surface.go`): acquire/present/resize, verified headless and against real windows (X11 on Xvfb, Win32 on ANGLE). On darwin the present path itself is exercised by `TestBlitPresentNoUseAfterFree`, but offscreen: `metalBackend.newWindowSurface` still returns `ErrUnsupported`, so a `CAMetalLayer` drawable is the open piece |
| [cgo-free-windowed-present.md](foundations/cgo-free-windowed-present.md) | **Done** | windowed present ported off the cgo windowing toy to purego/objc on all three platforms; every backend is purego and `CGO_ENABLED=0 go build ./...` is green |
| [gl-windowed-present-cleanup.md](foundations/gl-windowed-present-cleanup.md) | **Done, CI-proven** | linux and windows present routed through the one Device/Surface seam, and the duplicate standalone `gpu/gl` + `gpu/ctx/egl` stacks deleted |
| [gpu-vulkan-backend.md](foundations/gpu-vulkan-backend.md) | **Compute backend done, CI-verified** | cgo-free Vulkan compute wired behind the `backend` interface: `gpu.Open(DriverVulkan)` runs kernels through the Device API on Mesa lavapipe (SPIR-V via glslang), matched to CPU. Remaining: render, Go-to-SPIR-V, Windows; DX12 separate |
| [gpu-dx12-backend.md](foundations/gpu-dx12-backend.md) | **Viability proven (probe green), backend not built** | cgo-free D3D12 device created in CI on windows-latest via WARP/Basic Render Driver (syscall, no cgo). Remaining: COM command/pipeline/dispatch (HLSL via D3DCompile), then wire behind the interface |
| [unified-renderer.md](foundations/unified-renderer.md) | **Broken down; the slices below shipped** | unify CPU + GPU renderers: author passes once as Go kernels (run as Go on CPU, compiled to MSL/GLSL/SPIR-V on GPU), GPU by default with CPU fallback. Phases 1-2 landed; the rasterizer arc continues in the bricks below |
| [author-once-kernels.md](foundations/author-once-kernels.md) | **Done** | a `gpumath` library + compiler lowering of method/free-func form, so one Go kernel runs as Go on the CPU and compiles to GPU; proven on the Blinn-Phong kernel by parity |
| [render-pass-runner.md](foundations/render-pass-runner.md) | **Done** | `runPass(name, gpu, cpu)` + a per-pass path record in `render/raster.go`; forward, deferred and gamma all route through it, each falling back to CPU on error |
| [gpu-by-default.md](foundations/gpu-by-default.md) | **Done** | `NewRenderer` acquires a device itself (CPU fallback, closed in its finalizer) and `render.CPU()` forces the CPU path. Auto-acquisition is still Metal-only; other drivers reach the renderer via `render.GPU(dev)` |
| [render-deferred-author-once.md](foundations/render-deferred-author-once.md) | **Done** | the GPU deferred pass compiles `kernels.ShadeSrc`; the duplicate private kernel string in `render/gpudeferred.go` is gone |
| [render-shading-equivalence.md](foundations/render-shading-equivalence.md) | **Done** | `render/shading_equiv_test.go` pins the CPU default (`shader.FragmentShader`) equivalent to the author-once `kernels.Shade` |
| [render-multibackend-kernels.md](foundations/render-multibackend-kernels.md) | **Done, CI-verified** | render compiles each kernel for `dev.Driver()` (MSL for Metal, GLSL for GL), so the render GPU passes run on a GL device too; exercised end to end by the gl-probe job |
| [author-once-postprocess-kernels.md](foundations/author-once-postprocess-kernels.md) | **Done** | shadow, AO and sRGB moved into `gpu/shader/gpumath/kernels` as author-once Go; no kernel DSL string is left in `render/` |
| [shading-path-cleanup.md](foundations/shading-path-cleanup.md) | **Done** | dead `BlinnShader` removed, the misnamed `blinn_old.go` is now `blinn_cpu.go`, and the compiler test reads the canonical kernel sources instead of a copy |
| [archive-programmable-shader.md](foundations/archive-programmable-shader.md) | **Done** | the 2022-era `shader.Program` pipeline and the `AttrSmooth`/`AttrFlat` attribute-map varyings retired in favor of typed fragments + the author-once kernels |
| [material-ownership.md](foundations/material-ownership.md) | **Done** | the process-wide material pool is deleted: materials are geometry-owned and tabulated per frame by the renderer, with `material.Default()` the only shared instance |
| [gpu-render-depth.md](foundations/gpu-render-depth.md) | **Done, CI-verified** | depth attachments in the render pipeline/pass (rasterizer brick 1), Metal first then GL (`TestGLRenderDepthOcclusion`) |
| [gpu-render-mrt.md](foundations/gpu-render-mrt.md) | **Done, CI-verified** | multiple color attachments, the G-buffer prerequisite (rasterizer brick 2), Metal first then GL (`TestGLRenderMRT`) |
| [gpu-forward-raster.md](foundations/gpu-forward-raster.md) | **Done, parity-gated** | the GPU forward rasterizer is the default `passForward` on GL (CI) and Metal (darwin): vertex transform, back-face cull, depth test and a three-target G-buffer, gated by measured parity against the CPU pass |
| [gpu-material-texture-sampling.md](foundations/gpu-material-texture-sampling.md) | **Drafted, architecture locked; not started** | GPU-side material texture sampling by transliterating `buffer.Texture.Query` into an author-once kernel over a mipmap atlas (no hardware samplers), and with it seam option B: keep the G-buffer on the GPU into the deferred pass instead of today's textures → CPU `FragmentBuffer` → storage-buffers round-trip |

The GPU abstraction's Metal-backend phases are complete, and the renderer now
runs both forward rasterization and deferred shading on the GPU, cgo-free, with
the shading kernels authored once in Go; GL is CI-verified and windowed present
is proven on all three platforms. What is left is no longer only breadth.
Breadth: the Vulkan render path and the DX12 backend, both simply unbuilt (their
device and compute probes are green in CI, so what remains is code, not an
environment). Depth: GPU material texture sampling (which unblocks deleting the
forward-to-deferred CPU round-trip), the raster vertex/fragment shaders still
hand-written per language instead of authored once in Go, and automatic device
acquisition still limited to Metal.
