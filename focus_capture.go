package main

import (
	"time"

	"myscreen/internal/desktop"
	"myscreen/internal/domain"
)

// The indicator can briefly disappear while Computer Use replaces its windows.
// Release only after a short grace period, then start the user's focus delay
// from a fresh desktop sample. This is separate from the manual pause.
const computerUseEndGrace = time.Second

func (n *NativeBackend) updateComputerUsePause(visible bool, now time.Time) bool {
	wasPaused := !n.lastComputerUse.IsZero()
	if visible {
		n.lastComputerUse = now
		if !wasPaused {
			clear(n.focus)
			n.lastDesktopAt = time.Time{}
		}
	} else if wasPaused && now.Sub(n.lastComputerUse) >= computerUseEndGrace {
		n.lastComputerUse = time.Time{}
	}
	return !n.lastComputerUse.IsZero()
}

func (n *NativeBackend) tickFocus(now time.Time) error {
	if n.desktop == nil {
		return nil
	}
	if !n.settings.FocusEnabled || n.paused {
		clear(n.focus)
		n.lastDesktopAt = time.Time{}
		n.focusCaptureErr = ""
		n.lastComputerUse = time.Time{}
		return n.desktop.SetMasks(nil)
	}
	visible, err := n.desktop.ComputerUseVisible()
	if err != nil {
		n.acceptFocusCapture(desktop.DesktopState{}, err, now)
		return n.desktop.SetMasks(nil)
	}
	if n.updateComputerUsePause(visible, now) {
		return n.desktop.SetMasks(nil)
	}
	if now.Sub(n.lastDesktopAt) >= 250*time.Millisecond {
		state, err := n.desktop.CaptureDesktopState()
		if !n.acceptFocusCapture(state, err, now) {
			return n.desktop.SetMasks(nil)
		}
	}
	state := n.lastDesktop
	var masks []desktop.Mask
	for _, d := range n.worker.Displays {
		bounds := d.Bounds
		on := state.X >= bounds.Left && state.X < bounds.Right && state.Y >= bounds.Top && state.Y < bounds.Bottom
		focus, dim := domain.EvaluateFocus(n.settings, n.focus[d.Id], domain.FocusInput{DisplayId: d.Id, IsPrimary: d.Primary, CursorOnDisplay: on, HasFullscreenWindow: state.Occupied[d.GDI], NowMillis: now.Sub(n.started).Milliseconds()})
		n.focus[d.Id] = focus
		if dim {
			masks = append(masks, desktop.Mask{Display: d.GDI, Bounds: desktop.Rect{Left: bounds.Left, Top: bounds.Top, Right: bounds.Right, Bottom: bounds.Bottom}, Opacity: .999})
		}
	}
	return n.desktop.SetMasks(masks)
}

// Called under the backend lock. A failed desktop read must not reuse cursor
// coordinates or a partially elapsed focus delay. Keep this warning separate
// from hardware/audio warnings so recovery clears only the desktop read issue.
func (n *NativeBackend) acceptFocusCapture(state desktop.DesktopState, err error, now time.Time) bool {
	previous := n.focusCaptureErr
	n.focusCaptureErr = ""
	if err != nil {
		n.focusCaptureErr = err.Error()
		n.lastDesktop = desktop.DesktopState{}
		n.lastDesktopAt = time.Time{}
		n.focus = map[string]domain.FocusState{}
	} else {
		n.lastDesktop = state
		n.lastDesktopAt = now
	}
	if previous != n.focusCaptureErr && n.trace != nil {
		n.trace("focus-capture-status", map[string]any{"available": err == nil, "error": n.focusCaptureErr})
	}
	return err == nil
}

func (n *NativeBackend) currentWarning() string {
	if n.warning != "" {
		return n.warning
	}
	if n.focusCaptureErr != "" {
		return "暂时无法读取桌面状态，专注遮罩已暂停，恢复后自动继续。"
	}
	return ""
}
