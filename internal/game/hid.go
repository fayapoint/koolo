package game

import "github.com/lxn/win"

type HID struct {
	gr *MemoryReader
	gi *MemoryInjector
}

func NewHID(gr *MemoryReader, gi *MemoryInjector) *HID {
	return &HID{
		gr: gr,
		gi: gi,
	}
}

// GetHWND returns the window handle for the game
func (hid *HID) GetHWND() win.HWND {
	return hid.gr.HWND
}

// CalculateLParam calculates the lParam for keyboard messages
func (hid *HID) CalculateLParam(keyCode byte, down bool) uintptr {
	return hid.calculatelParam(keyCode, down)
}
