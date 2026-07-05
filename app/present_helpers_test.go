// Copyright 2026 The Polyred Authors. All rights reserved.
// Use of this source code is governed by a GPLv3 license that
// can be found in the LICENSE file.

//go:build linux || windows

package app

import (
	"image"
	"os"
	"testing"
)

// requireOrSkip turns a skip into a hard failure when POLYRED_REQUIRE_WINDOW is
// set. CI runs the windowed present tests in an environment where the display and
// the GL runtime are guaranteed present (Xvfb + Mesa on Linux; ANGLE + a desktop on
// Windows), so a skip there means the very thing the test exists to prove silently
// did not run. On a bare dev box the env var is unset and the test skips cleanly.
func requireOrSkip(t *testing.T, format string, args ...any) {
	t.Helper()
	if os.Getenv("POLYRED_REQUIRE_WINDOW") != "" {
		t.Fatalf("POLYRED_REQUIRE_WINDOW set but the windowed path is unavailable: "+format, args...)
	}
	t.Skipf(format, args...)
}

// solidRGBA returns a tightly-packed w*h RGBA image filled with c.
func solidRGBA(w, h int, c [4]byte) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c[0], c[1], c[2], c[3]
	}
	return img
}
