// Copyright 2021 Changkun Ou <changkun.de>. All rights reserved.
// Use of this source code is governed by a GPLv3 license that
// can be found in the LICENSE file.

//go:build darwin
// +build darwin

package gui

import (
	"fmt"
	"image"
	"unsafe"

	"github.com/go-gl/glfw/v3.3/glfw"

	"poly.red/internal/driver/mtl"
)

// driverInfo contains graphics driver informations.
type driverInfo struct {
	device mtl.Device
	ml     mtl.MetalLayer
	cq     mtl.CommandQueue
}

func (w *Window) initWinHints() {
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
}

func (w *Window) initDriver() {
	device, err := mtl.CreateSystemDefaultDevice()
	if err != nil {
		panic(fmt.Errorf("failed to initialize metal: %w", err))
	}
	dx := int(float32(w.width) * w.scaleX)
	dy := int(float32(w.height) * w.scaleY)

	ml := mtl.MakeMetalLayer()
	ml.SetDevice(device)
	ml.SetPixelFormat(mtl.PixelFormatBGRA8UNorm)
	ml.SetMaximumDrawableCount(3)
	ml.SetDrawableSize(dx, dy)
	ml.SetDisplaySyncEnabled(true)
	cv := mtl.NewWindow(
		unsafe.Pointer(w.win.GetCocoaWindow())).ContentView()
	cv.SetLayer(ml)
	cv.SetWantsLayer(true)
	cq := device.MakeCommandQueue()
	w.driverInfo = driverInfo{device: device, ml: ml, cq: cq}
}

func (w *Window) initContext() {
	// Nothing needs to be done on Metal.
}

// flush flushes the containing pixel buffer of the given image to the
// hardware frame buffer for display prupose. The given image is assumed
// to be non-nil pointer.
func (w *Window) flush(buf *frameBuf) error {
	// Make sure the previous frame is complete.
	if buf.done != nil {
		<-buf.done
	}
	buf.done = make(chan struct{})

	dx, dy := buf.img.Bounds().Dx(), buf.img.Bounds().Dy()
	drawable, err := w.ml.NextDrawable()
	if err != nil {
		return fmt.Errorf("gui: couldn't get the next drawable: %w", err)
	}

	// We create a new texture for every draw call. A temporary texture
	// is needed since ReplaceRegion tries to sync the pixel data between
	// CPU and GPU, and doing it on the existing texture is inefficient.
	// The texture cannot be reused until sending the pixels finishes,
	// then create new ones for each call.
	tex := w.device.MakeTexture(mtl.TextureDescriptor{
		PixelFormat: mtl.PixelFormatBGRA8UNorm,
		Width:       dx,
		Height:      dy,
		StorageMode: mtl.StorageModeManaged,
	})
	region := mtl.RegionMake2D(0, 0, dx, dy)
	tex.ReplaceRegion(region, 0, &buf.img.Pix[0], uintptr(4*dx))
	cb := w.cq.MakeCommandBuffer()
	bce := cb.MakeBlitCommandEncoder()
	bce.CopyFromTexture(tex, 0, 0, mtl.Origin{},
		mtl.Size{Width: dx, Height: dy, Depth: 1},
		drawable.Texture(), 0, 0, mtl.Origin{})
	bce.EndEncoding()
	cb.PresentDrawable(drawable)

	// We need a synchornization here. Similar to glFinish,
	// instead of glFlush. See a general discussion about CPU, GPU
	// and display synchornization here:
	//
	// Working with Metal: Fundamentals, 21:28
	// https://developer.apple.com/videos/play/wwdc2014/604/
	//
	// We may not need such an wait, if we are doing perfect timing.
	// See: https://golang.design/research/ultimate-channel/
	cb.AddCompletedHandler(func() { close(buf.done) })

	cb.Commit()
	return nil
}

// resetBuffers assign new buffers to the caches window buffers (w.bufs)
// Note: with Metal, we always use BGRA pixel format.
func (w *Window) resetBufs(r image.Rectangle) {
	// SetDrawableSize later so that all buffers are prepared for flushing.
	w.ml.SetDrawableSize(r.Dx(), r.Dy())
}
