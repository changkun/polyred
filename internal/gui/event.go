// Copyright 2021 Changkun Ou <changkun.de>. All rights reserved.
// Use of this source code is governed by a GPLv3 license that
// can be found in the LICENSE file.

package gui

import "github.com/go-gl/glfw/v3.3/glfw"

type Event interface{}

type EventName int

const (
	OnResize EventName = iota
	OnCursor
	OnMouseUp
	OnMouseDown
	OnScroll
	OnKeyUp
	OnKeyDown
	OnKeyRepeat
)

// SizeEvent describers a window size changed event
type SizeEvent struct {
	Width  int
	Height int
}

// MouseEvent describes a mouse event over the window
type MouseEvent struct {
	Xpos   float64
	Ypos   float64
	Button MouseButton
	Mods   ModifierKey
}

// CursorEvent describes a cursor position change event
type CursorEvent struct {
	Xpos, Ypos float64
	Mods       ModifierKey
}

// ScrollEvent describes a scroll event
type ScrollEvent struct {
	Xoffset float64
	Yoffset float64
	Mods    ModifierKey
}

// KeyEvent describes a window key event
type KeyEvent struct {
	Key  Key
	Mods ModifierKey
}

// Mouse buttons
const (
	MouseButtonLeft   = MouseButton(glfw.MouseButtonLeft)
	MouseButtonRight  = MouseButton(glfw.MouseButtonRight)
	MouseButtonMiddle = MouseButton(glfw.MouseButtonMiddle)
)
