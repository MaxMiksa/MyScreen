//go:build windows

package desktop

import (
	"errors"
	"math"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	focusIsWindowVisible = u32.NewProc("IsWindowVisible")
	focusIsIconic        = u32.NewProc("IsIconic")
	focusGetClassNameW   = u32.NewProc("GetClassNameW")
	focusIsZoomed        = u32.NewProc("IsZoomed")
	focusGetWindowRect   = u32.NewProc("GetWindowRect")
	focusGetExStyle      = u32.NewProc("GetWindowLongW")
	focusGetWindowPID    = u32.NewProc("GetWindowThreadProcessId")

	enumerations      sync.Map
	enumerationID     atomic.Uint64
	monitorCallback   = windows.NewCallback(enumMonitor)
	occupancyCallback = windows.NewCallback(enumOccupancy)
)

type enumeration struct {
	displays []Display
	occupied map[string]bool
	err      error
}

const (
	// Click-through, layered tool windows that cannot activate (e.g. Computer
	// Use's cursor effects) are not applications occupying the screen.
	focusPassiveOverlay = uint32(exOverlay)
	// The Windows Snipping Tool selection layer is a topmost XAML tool window
	// without a redirection bitmap. Its normal editor is not a selection layer.
	focusSnippingOverlay = 0x00000008 | 0x00000080 | 0x00200000
)

func isTransientFocusOverlay(class string, style uint32, exe func() string) bool {
	return style&focusPassiveOverlay == focusPassiveOverlay ||
		(class == "XamlWindow" && style&focusSnippingOverlay == focusSnippingOverlay &&
			strings.EqualFold(exe(), "SnippingTool.exe"))
}

func focusWindowExe(hwnd uintptr) string {
	var pid uint32
	if r, _, _ := focusGetWindowPID.Call(hwnd, uintptr(unsafe.Pointer(&pid))); r == 0 || pid == 0 {
		return ""
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)
	var buf [1024]uint16
	n := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &n); err != nil {
		return ""
	}
	return filepath.Base(windows.UTF16ToString(buf[:n]))
}

func markFocusOccupancy(e *enumeration, d Display, hwnd uintptr, class string) {
	style, _, _ := focusGetExStyle.Call(hwnd, ^uintptr(19)) // GWL_EXSTYLE (-20), a 32-bit value.
	if !isTransientFocusOverlay(class, uint32(style), func() string { return focusWindowExe(hwnd) }) {
		e.occupied[d.Display] = true
	}
}

func enumMonitor(handle, dc, rect, token uintptr) uintptr {
	v, ok := enumerations.Load(token)
	if !ok {
		return 0
	}
	e := v.(*enumeration)
	var mi monitorInfo
	mi.Size = uint32(unsafe.Sizeof(mi))
	if err := boolCall(getMonitorInfo, handle, uintptr(unsafe.Pointer(&mi))); err != nil {
		e.err = err
		return 0
	}
	d := Display{Display: windows.UTF16ToString(mi.Device[:]), Bounds: mi.Bounds, WorkArea: mi.Work, Primary: mi.Flags&1 != 0, DPI: 96}
	var x, y uint32
	if hr, _, _ := getDPI.Call(handle, 0, uintptr(unsafe.Pointer(&x)), uintptr(unsafe.Pointer(&y))); int32(hr) >= 0 && x != 0 {
		d.DPI = x
	}
	e.displays = append(e.displays, d)
	return 1
}
func captureDisplays() ([]Display, error) {
	e := &enumeration{}
	id := uintptr(enumerationID.Add(1))
	enumerations.Store(id, e)
	defer enumerations.Delete(id)
	err := boolCall(enumMonitors, 0, 0, monitorCallback, id)
	if e.err != nil {
		return nil, e.err
	}
	if err != nil {
		return nil, err
	}
	if len(e.displays) == 0 {
		return nil, errors.New("no desktop displays")
	}
	return e.displays, nil
}

// CaptureDesktopState is read-only: no windows/hooks are created. It examines all
// top-level windows, independently of foreground, and reports secondary monitors.
// DWM-cloaked windows include windows on inactive virtual desktops.
func CaptureDesktopState() (DesktopState, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	previous, _, err := setDPIContext.Call(^uintptr(3))
	if previous == 0 {
		return DesktopState{}, winError("SetThreadDpiAwarenessContext", err)
	}
	defer setDPIContext.Call(previous)
	return captureState()
}
func (c *Controller) CaptureDesktopState() (DesktopState, error) {
	var state DesktopState
	err := c.call(func() error { var err error; state, err = captureState(); return err })
	return state, err
}
func captureState() (DesktopState, error) {
	if !isConsole() {
		return DesktopState{}, ErrNotConsole
	}
	var p point
	if err := boolCall(getCursor, uintptr(unsafe.Pointer(&p))); err != nil {
		return DesktopState{}, err
	}
	displays, err := captureDisplays()
	if err != nil {
		return DesktopState{}, err
	}
	e := &enumeration{displays: displays, occupied: make(map[string]bool)}
	id := uintptr(enumerationID.Add(1))
	enumerations.Store(id, e)
	defer enumerations.Delete(id)
	if err := boolCall(enumWindows, occupancyCallback, id); err != nil {
		return DesktopState{}, err
	}
	return DesktopState{X: p.X, Y: p.Y, Foreground: foreground(), Displays: displays, Occupied: e.occupied}, nil
}

func enumOccupancy(hwnd, token uintptr) uintptr {
	v, ok := enumerations.Load(token)
	if !ok {
		return 0
	}
	e := v.(*enumeration)
	visible, _, _ := focusIsWindowVisible.Call(hwnd)
	iconic, _, _ := focusIsIconic.Call(hwnd)
	if visible == 0 || iconic != 0 {
		return 1
	}
	if _, ours := windowOwners.Load(hwnd); ours {
		return 1
	}
	var class [256]uint16
	n, _, _ := focusGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&class[0])), uintptr(len(class)))
	if n == 0 {
		return 1
	}
	className := windows.UTF16ToString(class[:n])
	switch className {
	case "Progman", "WorkerW", "Shell_TrayWnd", "Shell_SecondaryTrayWnd", "MyScreen.Desktop.Native":
		return 1
	}
	var cloaked uint32
	hr, _, _ := dwmAttribute.Call(hwnd, 14, uintptr(unsafe.Pointer(&cloaked)), unsafe.Sizeof(cloaked))
	if int32(hr) < 0 || cloaked != 0 {
		return 1
	}
	maximized, _, _ := focusIsZoomed.Call(hwnd)
	if maximized != 0 {
		h, _, _ := monitorFromWindow.Call(hwnd, 0)
		var mi monitorInfo
		mi.Size = uint32(unsafe.Sizeof(mi))
		if h != 0 && boolCall(getMonitorInfo, h, uintptr(unsafe.Pointer(&mi))) == nil {
			for _, d := range e.displays {
				if !d.Primary && strings.EqualFold(d.Display, windows.UTF16ToString(mi.Device[:])) {
					markFocusOccupancy(e, d, hwnd, className)
				}
			}
		}
		return 1
	}
	var bounds Rect
	hr, _, _ = dwmAttribute.Call(hwnd, 9, uintptr(unsafe.Pointer(&bounds)), unsafe.Sizeof(bounds))
	if int32(hr) < 0 && boolCall(focusGetWindowRect, hwnd, uintptr(unsafe.Pointer(&bounds))) != nil {
		return 1
	}
	for _, d := range e.displays {
		if !d.Primary && matchesFullscreen(bounds, d.Bounds) {
			markFocusOccupancy(e, d, hwnd, className)
		}
	}
	return 1
}

// SetMasks represents the complete desired set, not a patch. Only exact current
// secondary rectangles are accepted; nil removes every mask. Windows are reused.
func (c *Controller) SetMasks(masks []Mask) error {
	masks = append([]Mask(nil), masks...)
	return c.call(func() error {
		if len(masks) == 0 {
			return c.clearMasks()
		}
		if !isConsole() {
			return errors.Join(ErrNotConsole, c.clearMasks())
		}
		displays, err := captureDisplays()
		if err != nil {
			return errors.Join(err, c.clearMasks())
		}
		wanted := make(map[string]Mask, len(masks))
		for _, m := range masks {
			if math.IsNaN(m.Opacity) || math.IsInf(m.Opacity, 0) || m.Opacity < 0 || m.Opacity > 1 {
				return errors.New("mask opacity must be 0..1")
			}
			found := false
			for _, d := range displays {
				if strings.EqualFold(m.Display, d.Display) && !d.Primary && m.Bounds == d.Bounds && m.Bounds.valid() {
					m.Display = d.Display
					found = true
					break
				}
			}
			if !found {
				return errors.Join(errors.New("mask must match an attached secondary display"), c.clearMasks())
			}
			if _, exists := wanted[m.Display]; exists {
				return errors.New("duplicate mask display")
			}
			wanted[m.Display] = m
		}
		// Topology observed here is also used by the watchdog.
		c.acceptDisplays(withCapabilities(displays, c.options))
		for name, h := range c.masks {
			m, keep := wanted[name]
			if keep && m.Opacity > 0 {
				continue
			}
			if err := c.destroy(h); err != nil {
				return errors.Join(err, c.clearMasks())
			}
			delete(c.masks, name)
		}
		for name, m := range wanted {
			if m.Opacity == 0 {
				continue
			}
			h := c.masks[name]
			if h == 0 {
				h, err = makeWindow(c, false)
				if err != nil {
					return errors.Join(err, c.clearMasks())
				}
				c.masks[name] = h
			}
			alpha := uintptr(math.Round(m.Opacity * 255))
			if alpha > 254 {
				alpha = 254
			}
			if alpha < 1 {
				alpha = 1
			}
			if err = boolCall(setLayered, h, 0, alpha, 2); err != nil {
				return errors.Join(err, c.clearMasks())
			}
			b := m.Bounds
			if err = boolCall(setWindowPos, h, ^uintptr(0), uintptr(b.Left), uintptr(b.Top), uintptr(b.Right-b.Left), uintptr(b.Bottom-b.Top), swpShow); err != nil {
				return errors.Join(err, c.clearMasks())
			}
		}
		if err := c.updateScopeTimer(); err != nil {
			return errors.Join(err, c.clearMasks())
		}
		if len(c.masks) == 0 {
			return c.stopMaskOrderWatch()
		}
		return c.startMaskOrderWatch()
	})
}
func (c *Controller) clearMasks() error {
	var result error
	for name, h := range c.masks {
		if err := c.destroy(h); err != nil {
			result = errors.Join(result, err)
		} else {
			delete(c.masks, name)
		}
	}
	c.stopScopeIfIdle()
	return errors.Join(result, c.stopMaskOrderWatch())
}
