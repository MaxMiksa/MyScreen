//go:build windows

package desktop

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var findWindowEx = u32.NewProc("FindWindowExW")

var computerUseIndicators = [...]struct {
	class *uint16
	style uint32
}{
	{windows.StringToUTF16Ptr("CodexComputerUseSwiftOverlay"), (focusPassiveOverlay &^ 0x00080000) | 0x00200000},
	{windows.StringToUTF16Ptr("CodexComputerUseCursorOverlay"), focusPassiveOverlay},
}

// ComputerUseVisible is a read-only probe for the native session indicator.
// The helper can remain running after a session; only a shown, uncloaked
// cursor overlay on this console is evidence of active use.
func (c *Controller) ComputerUseVisible() (bool, error) {
	var active bool
	err := c.call(func() error {
		if !isConsole() {
			return ErrNotConsole
		}
		for _, indicator := range computerUseIndicators {
			var previous uintptr
			for range 64 { // Window creation or destruction must not prolong the tick.
				h, _, _ := findWindowEx.Call(0, previous, uintptr(unsafe.Pointer(indicator.class)), 0)
				if h == 0 || h == previous {
					break
				}
				previous = h
				shown, _, _ := focusIsWindowVisible.Call(h)
				if shown == 0 {
					continue
				}
				style, _, _ := focusGetExStyle.Call(h, ^uintptr(19)) // GWL_EXSTYLE
				if uint32(style)&indicator.style != indicator.style {
					continue
				}
				var cloaked uint32
				hr, _, _ := dwmAttribute.Call(h, 14, uintptr(unsafe.Pointer(&cloaked)), unsafe.Sizeof(cloaked))
				if int32(hr) >= 0 && cloaked == 0 {
					active = true
					return nil
				}
			}
		}
		return nil
	})
	return active, err
}
