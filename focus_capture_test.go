package main

import (
	"fmt"
	"testing"
	"time"

	"golang.org/x/sys/windows"
	"myscreen/internal/desktop"
	"myscreen/internal/domain"
)

func TestComputerUsePausesFocusUntilIndicatorStaysGone(t *testing.T) {
	now := time.Now()
	left := int64(100)
	n := NativeBackend{
		focus:         map[string]domain.FocusState{"secondary": {LeftAt: &left}},
		lastDesktopAt: now,
	}
	if n.updateComputerUsePause(false, now) {
		t.Fatal("an idle desktop must not pause focus")
	}
	if !n.updateComputerUsePause(true, now) || len(n.focus) != 0 || !n.lastDesktopAt.IsZero() || n.paused {
		t.Fatal("session start must reset the focus delay and preserve the manual pause")
	}
	if !n.updateComputerUsePause(false, now.Add(computerUseEndGrace-time.Millisecond)) {
		t.Fatal("a brief gap in the indicator must not resume focus")
	}
	lastSeen := now.Add(computerUseEndGrace)
	if !n.updateComputerUsePause(true, lastSeen) || !n.updateComputerUsePause(false, lastSeen.Add(computerUseEndGrace-time.Millisecond)) {
		t.Fatal("a returning indicator must renew the grace period")
	}
	n.paused = true
	if n.updateComputerUsePause(false, lastSeen.Add(computerUseEndGrace)) || !n.paused || !n.lastDesktopAt.IsZero() {
		t.Fatal("a stable end must release only automatic pause and require a fresh capture")
	}
}

func TestFocusCaptureFailureAndRecovery(t *testing.T) {
	now := time.Now()
	left := int64(100)
	var events []map[string]any
	n := NativeBackend{
		lastDesktop: desktop.DesktopState{X: -400, Y: 700}, lastDesktopAt: now,
		focus: map[string]domain.FocusState{"secondary": {LeftAt: &left}},
		trace: func(_ string, fields map[string]any) { events = append(events, fields) },
	}
	denied := fmt.Errorf("GetCursorPos: %w", windows.ERROR_ACCESS_DENIED)
	for range 2 {
		if n.acceptFocusCapture(desktop.DesktopState{}, denied, now) {
			t.Fatal("failed read was accepted")
		}
	}
	if n.lastDesktop.X != 0 || !n.lastDesktopAt.IsZero() || len(n.focus) != 0 {
		t.Fatal("failed read retained stale cursor data or focus delay")
	}
	if n.currentWarning() != "暂时无法读取桌面状态，专注遮罩已暂停，恢复后自动继续。" || len(events) != 1 {
		t.Fatal("failure must show one scoped warning and avoid repeated diagnostic spam")
	}
	state := desktop.DesktopState{X: 20, Y: 40, Occupied: map[string]bool{}}
	if !n.acceptFocusCapture(state, nil, now) || n.lastDesktop.X != 20 || !n.lastDesktopAt.Equal(now) {
		t.Fatal("successful read did not replace the failed state")
	}
	if n.currentWarning() != "" || len(events) != 2 || events[1]["available"] != true {
		t.Fatal("desktop recovery left a stale warning")
	}
	// An independent warning must survive both a failed read and its recovery.
	n.warning = "音频设备不可用"
	n.acceptFocusCapture(desktop.DesktopState{}, denied, now)
	n.acceptFocusCapture(state, nil, now)
	if n.currentWarning() != "音频设备不可用" {
		t.Fatal("desktop recovery erased another subsystem's warning")
	}
}
