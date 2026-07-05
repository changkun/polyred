// Copyright 2026 The Polyred Authors. All rights reserved.
// Use of this source code is governed by a GPLv3 license that
// can be found in the LICENSE file.

//go:build windows

package app

import (
	"os"
	"runtime"
	"testing"

	"poly.red/gpu"
	"poly.red/gpu/syscall/windows"
)

// TestWin32WindowedPresent drives the cgo-free Win32 + ANGLE windowed present path
// end to end, the Windows analog of TestX11WindowedPresent: register the window
// class, create + show a real HWND at a known client size, open the GL device (ANGLE
// libEGL/libGLESv2) on its HDC, bind an on-screen Surface, then present several
// frames across a resize, reading the presented pixels back each time. It is the
// runtime proof of the Win32 present path AND the thread/context-ownership model
// (all GL/EGL on the backend's single locked thread while the app drives present
// from another). A single clear-and-readback would miss a marshaling bug, hence
// multiple frames + a resize.
//
// It runs only where ANGLE and a desktop window station are guaranteed
// (POLYRED_REQUIRE_WINDOW); on a runner/box without the ANGLE runtime it skips
// cleanly. gpu.Open / CreateWindowSurface failing under POLYRED_REQUIRE_WINDOW is a
// hard failure -- that is the break this test exists to catch.
func TestWin32WindowedPresent(t *testing.T) {
	if os.Getenv("POLYRED_REQUIRE_WINDOW") == "" {
		t.Skip("windowed present runs only where ANGLE + a desktop are guaranteed (POLYRED_REQUIRE_WINDOW)")
	}
	// Win32 windows are thread-affine; pin this goroutine like run() does. (The GL
	// backend owns its own locked thread; present marshals onto it.)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := initResources(); err != nil {
		t.Fatalf("initResources: %v", err)
	}

	const cw, ch = 64, 48
	dwStyle := uint32(windows.WS_OVERLAPPEDWINDOW | windows.WS_CLIPSIBLINGS | windows.WS_CLIPCHILDREN)
	dwExStyle := uint32(windows.WS_EX_APPWINDOW | windows.WS_EX_WINDOWEDGE)

	// Size the window so its CLIENT area is cw x ch (the surface size must match the
	// window the swapchain is bound to).
	r := windows.Rect{Left: 0, Top: 0, Right: cw, Bottom: ch}
	windows.AdjustWindowRectEx(&r, dwStyle, 0, dwExStyle)
	hwnd, err := windows.CreateWindowEx(dwExStyle,
		resources.class, "",
		dwStyle,
		windows.CW_USEDEFAULT, windows.CW_USEDEFAULT,
		r.Right-r.Left, r.Bottom-r.Top,
		0, 0, resources.handle, 0)
	if err != nil {
		t.Fatalf("CreateWindowEx: %v", err)
	}
	defer windows.DestroyWindow(hwnd)
	windows.ShowWindow(hwnd, windows.SW_SHOWDEFAULT)

	hdc, err := windows.GetDC(hwnd)
	if err != nil {
		t.Fatalf("GetDC: %v", err)
	}
	defer windows.ReleaseDC(hdc)

	// Use the actual client rect for the surface size (robust against border math).
	var cr windows.Rect
	windows.GetClientRect(hwnd, &cr)
	w, h := int(cr.Right), int(cr.Bottom)
	if w <= 0 || h <= 0 {
		w, h = cw, ch
	}

	// Open the GL device (ANGLE) on the window's HDC, then bind an on-screen Surface
	// to the HWND, mirroring run(). ANGLE's eglCreateWindowSurface takes the HWND.
	dev, err := gpu.Open(gpu.WithDriver(gpu.DriverGL), gpu.WithNativeDisplay(uintptr(hdc)))
	if err != nil {
		requireOrSkip(t, "no GL device (ANGLE libEGL/libGLESv2/driver missing): %v", err)
	}
	defer dev.Close()

	surf, err := dev.CreateWindowSurface(gpu.WindowSurfaceDescriptor{
		Display: uintptr(hdc),
		Window:  uintptr(hwnd),
		Width:   w,
		Height:  h,
		Format:  gpu.RGBA8Unorm,
	})
	if err != nil {
		requireOrSkip(t, "CreateWindowSurface failed (ANGLE / HWND swapchain): %v", err)
	}
	defer surf.Release()

	red := [4]byte{255, 0, 0, 255}

	// presentAndCheck presents a solid-red frame of size sw x sh and asserts the
	// presented pixels read back red. Driving this several times and across a resize
	// exercises the present loop and the resize realloc on the backend thread.
	presentAndCheck := func(sw, sh int) {
		img := solidRGBA(sw, sh, red)
		if err := surf.PresentImage(img); err != nil {
			t.Fatalf("PresentImage(%dx%d) failed: %v", sw, sh, err)
		}
		pix := surf.PresentedPixels()
		if len(pix) != sw*sh*4 {
			t.Fatalf("PresentedPixels len=%d, want %d", len(pix), sw*sh*4)
		}
		off := ((sh/2)*sw + sw/2) * 4
		got := [4]byte{pix[off], pix[off+1], pix[off+2], pix[off+3]}
		for i := range red {
			if diff := int(got[i]) - int(red[i]); diff < -2 || diff > 2 {
				t.Fatalf("presented center pixel=%v, want ~%v (gl present/blit marshaling)", got, red)
			}
		}
	}

	for range 4 {
		presentAndCheck(w, h)
	}

	// Resize the swapchain and present several more frames; surf.Resize reallocates
	// the upload/blit texture on the backend thread.
	w2, h2 := w/2, h/2
	if err := surf.Resize(w2, h2); err != nil {
		t.Fatalf("surface Resize failed: %v", err)
	}
	for range 4 {
		presentAndCheck(w2, h2)
	}
}
