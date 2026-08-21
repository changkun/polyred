---
title: "GPU material texture sampling + seam option B (forward->deferred, no CPU round-trip)"
status: drafted (architecture locked: transliterate Query, no hardware samplers)
depends_on:
  - foundations/gpu-forward-raster.md
affects:
  - render
  - gpu
  - gpu/shader
  - buffer
created: 2026-07-01
updated: 2026-08-21
author: changkun
effort: xlarge
dispatched_task_id: null
---

# GPU material texture sampling + seam option B

## Why this exists as its own brick

`gpu-forward-raster.md` (brick 3b) sketched a "step 3": *remove the round-trip (seam
option B), keep the G-buffer on GPU textures into the deferred pass, no CPU
FragmentBuffer round-trip.* That step cannot live in the rasterizer brick because it
depends on an item that brick lists as **out of scope**: GPU-side material texture
sampling.

The rasterizer brick is complete: the GPU forward pass rasterizes the full G-buffer
(world position, normal, uv + du/dv, material id, depth) on the GPU by default on both
GL and Metal, gated by measured parity. What remains is the *seam* between forward and
deferred.

## The round-trip today (seam A)

1. `gpuForwardPass` rasterizes the G-buffer into GPU **textures** (RGBA32F MRT + depth).
2. It **reads those textures back** to the CPU `*buffer.FragmentBuffer` (`buf.Set`).
3. `passDeferred` -> `gpuDeferredShade(dev, buf, ...)` reads the CPU buffer and
   **re-uploads** normals/worldpos/basecol/matidx to GPU **storage buffers**, then runs
   the Blinn-Phong lighting compute kernel (`kernels.Shade`).

So the G-buffer makes a GPU-textures -> CPU -> GPU-buffers round-trip every frame.

## Why seam B is blocked on GPU texture sampling

The "GPU deferred" pass is a **hybrid**, not fully on the GPU:

- Material `basecol` is sampled on the **CPU**: `render/gpudeferred.go:184`,
  `bp.Texture.Query(lod, info.U, 1-info.V)`, using the FragmentBuffer's `U/V/Du/Dv`.
- Only the Blinn-Phong lighting math runs in the compute kernel, and that kernel reads
  storage **buffers**, not texture samplers.

For **flat** (non-textured) materials `basecol = bp.Diffuse`, already a constant in the
`materials[]` table, so seam B needs no sampling at all. For **textured** materials
(the bunny and the scenes that actually matter) the deferred pass cannot stop reading
the CPU FragmentBuffer until sampling moves onto the GPU.

## Architecture decision (locked 2026-08-21): transliterate `Query`, do not use hardware samplers

Two routes were weighed. **Route B, a storage-buffer mipmap atlas plus a
transliteration of `Texture.Query` as an author-once Go kernel, is chosen.** Route A,
real GPU textures and sampler objects with hardware trilinear filtering, is rejected
for now and stays available later as an opt-in fast path once a tolerance band is
acceptable.

`buffer.Texture.Query` is **not expressible as a hardware sampler**. Three independent
reasons, each verified in the source:

1. **Addressing is `u*(dx-1)`** (`buffer/texture.go`, `queryL0` and `queryBilinear`),
   not the half-texel `u*dx-0.5` convention every GPU sampling unit implements. Edge
   handling duplicates `p1` rather than clamping the coordinate, and the `p4` corner
   fallback is `RGBAAt(i, j)`, not `(i+1, j)`.
2. **LOD carries an off-by-one**: `if lod <= 1 { return queryBilinear(0, u, v) }`, then
   `lod -= 1` before the floor/frac trilinear split.
3. **It truncates to uint8 at every lerp.** `math.LerpC` (`math/interpolate.go:55`)
   converts with `uint8(Lerp(...))`, a truncating conversion, not rounding.
   `queryBilinear` performs three such lerps, `queryTrilinear` a fourth on top. A
   float-throughout GPU sampler is off by one almost everywhere.

Route A would additionally have to build new machinery before anything is provable:
`gpu/shader/compile.go:520` rejects `Texture2D`/`Sampler` for the **GLSL** backend, and
GL on Mesa llvmpipe is the CI oracle (the standing strategy is author backend-agnostic,
GL as CI oracle, Metal as darwin runtime). Route A is currently half-built on Metal
only, which would prove the feature on the runtime while leaving the oracle blind.

Route B needs **no new backend features**: it rides the storage-buffer path already
proven by `kernels.Shade` on Metal, GL, and Vulkan, works on GL and Metal from day one,
and is **bit-exact by construction** rather than by tuned tolerance.

**Scope reduction that only route B allows:** upload the CPU's **already-built** mipmap
pyramid verbatim. `NewTexture` resamples each level from level 0 with
`imageutil.Resize` at dims `dx/2^i` by `dy/2^i`, level count `log2(max(dx,dy))+1`. If
the pyramid bytes ship as-is, mipmap **generation** parity stops being a problem
entirely and the brick owes only **sampling** parity.

## The contract to reproduce

The kernel is a **transliteration**. Reproducing behavior that looks wrong is the
requirement, not a defect. `queryL0`'s own comment calls its addressing "very coarse
approximation"; that approximation is part of the contract. Any change to the CPU
sampler is a separate, deliberate change to both sides at once, never a silent
"fix" inside the kernel.

Ordered, from `buffer/texture.go`:

- **Wrap**: `iu, u := Modf(u)`; `if iu != 0 && u == 0 { u = 1 }`; `if u < 0 { u = 1-u }`.
  Same for `v`. Note this is not a standard repeat.
- **No mipmap** (`!useMipmap`): `queryL0`, nearest at
  `x = floor(u*(dx-1))`, `y = floor(v*(dy-1))`, with a 1x1 short circuit.
- **LOD clamp**: `lod < 0 -> 0`; `lod >= len(mipmap) -> len(mipmap)-1`.
- **`lod <= 1`** -> `queryBilinear(0, u, v)`. Otherwise `lod -= 1`.
- **Level split**: `h = floor(lod)`, `l = h+1`; if `l >= len(mipmap)` ->
  `queryBilinear(h, ...)`; `p = lod - h`; if `ApproxEq(p, 0, Epsilon)` ->
  `queryBilinear(h, ...)`; else `queryTrilinear`.
- **`queryBilinear(lod, u, v)`**: 1x1 short circuit; `x = u*(dx-1)`, `y = v*(dy-1)`;
  `i = int(floor(x))`, `j = int(floor(y))`; taps `p1=(i,j)`,
  `p2=(i+1,j)` else `p1`, `p3=(i,j+1)` else `p1`, `p4=(i+1,j+1)` else `(i,j)`;
  `interpo1 = LerpC(p1,p2,x-x0)`, `interpo2 = LerpC(p3,p4,x-x0)`,
  result `LerpC(interpo1, interpo2, y-y0)`, **each truncating to uint8**.
- **`queryTrilinear(h,l,p,u,v)`**: `LerpC(queryBilinear(h), queryBilinear(l), p)`.

Two convention lines that belong to the **caller**, not the sampler, and must move with
it (`render/gpudeferred.go`):

- The **`1-V` flip**: `Query(lod, info.U, 1-info.V)`.
- The **LOD formula**: `lod = log2(max(1, Texture.Size() * sqrt(max(Du, Dv))))`, and
  `lod = 0` when `!UseMipmap()`. `Size()` returns level-0 **width**.

## Data layout: the mipmap atlas

One storage buffer holding every material texture's full pyramid, plus a descriptor
table. Sketch, to be pinned down in brick 2:

- `atlas []float32` (or `[]uint32` packed RGBA8): all levels of all textures, level 0
  first, concatenated in material-table order.
- `texdesc []float32`: per material, `[offset, dx, dy, levelCount, useMipmap, size]`,
  with per-level offsets derivable from `dx/2^i, dy/2^i` exactly as `NewTexture` builds
  them.

Texel fetch is an index computation into `atlas`, so no sampler, no texture object, and
no new backend surface. Storage as RGBA8-packed keeps the uint8 truncation semantics
honest and shrinks the upload; the kernel unpacks per tap.

## Where sampling lives: a standalone kernel, `Shade` untouched

`basecol` stays a storage-buffer **input** to `kernels.Shade`, and `Shade` stays
**byte-unchanged**. Sampling goes in a new author-once kernel, `kernels.SampleBasecol`,
that **produces** the `basecol` buffer from `uv/dudv/matidx` plus the atlas.

The alternative, giving `Shade` atlas parameters and sampling inside it, would widen
`TestDeferredShadingEquivalence`, `deferredSelfCheck`, `TestGPUDeferredMultiMaterial`,
and every full-`Render()` GPU-vs-CPU gate at once. That is exactly the failure recorded
when GPU-forward went default and folded the forward parity band into six darwin
deferred/gamma gates (53% fail). Keeping the new kernel standalone means the
author-once equivalence lock stays intact and the new code gets its own dedicated gate.

## Bricks

Each brick is bounded, lands with its own gate, and is CI-green before the next starts.

### Brick 0: seam B for flat materials

No sampling at all. Prove the GPU-G-buffer to deferred plumbing while the risk is zero.

The forward pass writes **textures**; `Shade` reads storage **buffers**. Closing the
seam without a CPU hop therefore needs an on-device texture-to-buffer copy. Add
`CopyTextureToBuffer` to the `gpu` API and both backends (Metal blit encoder,
GL pixel-buffer path). This keeps `Shade`'s storage-buffer contract unchanged and is
reusable well beyond this brick.

Gate: a flat-material scene rendered with seam B matches seam A. Because no
interpolation convention changes, this gate is **exact**.

### Brick 1: compiler helper functions and `gpumath` gaps

`compileAll` (`gpu/shader/compile.go:160`) compiles **every** top-level func in a
kernel source as its **own entry point**. There is no helper-function concept, so
adding `queryBilinear` beside `Query` today would either produce a bogus entry point or
fail to compile (it returns a value and takes non-buffer params).

`Query` is naturally four functions and `queryTrilinear` calls `queryBilinear` twice.
Hand-flattening that into one function with duplicated bodies is precisely where
transliteration bugs come from, which would forfeit the bit-exactness that justifies
route B. So: **add helper-function lowering to the compiler.** Non-entry funcs emit as
device/static functions (MSL) and plain functions (GLSL) into each kernel's module.
Entry-point selection becomes explicit rather than "every func".

Also fill the `gpumath` gaps the transliteration needs: `Trunc`, a `Modf` equivalent,
`Log2`, and a `Quant8` helper carrying the `uint8(...)` truncation semantics.

Gate: device-free compiler tests (all platforms) that a multi-function source compiles
to one entry point plus helpers on both MSL and GLSL, and a parity test that a helper
using kernel matches its Go-as-CPU run.

### Brick 2: the atlas and `kernels.SampleBasecol`, bit-exact

Build the atlas upload, write the transliterated kernel, and lock it against `Query`.

Gate: **bit-exact**, and independent of any rasterization. Sample a known texture at a
fixed list of `(u, v, lod)` triples chosen to hit every branch above (`lod <= 1`, the
`lod -= 1` seam, `p ~= 0`, `l >= len(mipmap)`, edge taps, 1x1, `useMipmap` off,
negative and out-of-range `u/v`) and assert equality with `Texture.Query` on CPU, GPU
as GL, and GPU as Metal. This is the fails-without / passes-with test: run it against a
float-throughout sampler first and confirm it fails.

### Brick 3: seam B for textured materials

Wire `SampleBasecol` into the GPU deferred path so the textured G-buffer never touches
the CPU. Move the `1-V` flip and the LOD formula into the GPU path.

Gate: **measured band**, not exact, and it needs its own metric (see below).

## Testing strategy: two gates, two strictnesses

Route B puts no hardware in the sampling loop, so the sampler gate is exact. The
end-to-end gate cannot be, and the reason is specific:

In seam B, `du/dv` come from the forward pass's hardware `dFdx/dFdy`, while the CPU
derives them analytically per triangle. That divergence already sits inside the
accepted 4.38%@>8 forward band. But in seam B it now feeds **discrete mip-level
selection**, so at some pixels it flips a whole level and produces a visible color
step, not an epsilon.

So the end-to-end gate reports a **discrete, attributable metric**: the **count of
pixels selecting a different mip level** between CPU and GPU, alongside the usual
channel-delta percentage. Stating this split up front is what stops the exact sampler
gate from later being loosened to hide an LOD problem.

Existing gates that must stay green and unchanged: `TestDeferredShadingEquivalence`,
`TestGPUDeferredMultiMaterial`, `TestGLDeferredRender`, `TestGPUForwardPassUV`,
`TestGPUForwardDeferredIntegration`, `TestGPUForwardMetal`, and the darwin
deferred/gamma parity gates.

## Risks

- **Transliteration drift.** The CPU sampler and the kernel are two copies of one
  algorithm. Mitigate as the repo already does for `Shade`: the kernel runs as Go on the
  CPU in the equivalence test, so a divergence fails a test rather than shipping.
  Consider whether `buffer.Texture.Query` can eventually **call** the kernel, making the
  kernel the single authority, the same "authority plus locked equivalence" shape as
  `render-shading-equivalence.md`.
- **Performance.** No texture cache, no filtering units, and a dependent index chain per
  tap. Route B optimizes for provable correctness first. Measure before assuming this
  matters, and keep route A as the later opt-in fast path.
- **Atlas size.** Full pyramids for every material, uploaded per frame in the naive
  version. Upload once and cache by material identity; the material table is already
  tabulated per frame in `gpuDeferredShade`.
- **Compiler blast radius.** Brick 1 changes entry-point selection for **all** existing
  kernels. It must be behavior-preserving for `Shade`/`Shadow`/`AO`/`SRGB`, verified by
  byte-comparing the generated MSL and GLSL before and after.

## Out of scope

- MSAA on the GPU raster (inherited from the rasterizer brick).
- Anisotropic filtering (the CPU path is isotropic).
- Hardware samplers and GLSL `Texture2D`/`Sampler` support (route A, deferred).
- Changing `buffer.Texture.Query`'s behavior in any way.

## Deliverable

The forward -> deferred -> present pipeline stays on the GPU with no CPU round-trip for
textured scenes, with GPU material sampling proven bit-exact against
`buffer.Texture.Query` on both GL and Metal, and the end-to-end result gated by a
measured band with mip-level selection reported as its own metric.
