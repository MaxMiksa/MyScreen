package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"myscreen/internal/audio"
	"myscreen/internal/desktop"
	"myscreen/internal/domain"
	"myscreen/internal/win32"
	"os"
	"sync"
	"time"
)

type NativeBackend struct {
	mu              sync.Mutex
	worker          *win32.Worker
	writes          bool
	original        map[string]int
	closed          bool
	closing         bool
	closeOnce       sync.Once
	closeError      error
	settings        domain.Settings
	store           *domain.Store
	targets         map[string]domain.DisplayTarget
	displayErrors   map[string]string
	backlights      map[string]int
	focus           map[string]domain.FocusState
	schedule        domain.ScheduleState
	paused          bool
	source          string
	warning         string
	desktop         *desktop.Controller
	audio           *audio.Controller
	gestures        chan desktop.Gesture
	stop            chan struct{}
	done            chan struct{}
	started         time.Time
	lastTopology    string
	lastTopologyAt  time.Time
	lastTick        time.Time
	lastColorCheck  time.Time
	trace           func(string, map[string]any)
	production      bool
	togglePanel     func()
	notify          func(string)
	nextAudioNotice time.Time
	lastDesktop     desktop.DesktopState
	lastDesktopAt   time.Time
	focusCaptureErr string
	lastComputerUse time.Time
}

func newHardwareBackend(writes bool) (*NativeBackend, error) {
	if writes && win32.ExistingCore() {
		return nil, errors.New("现用 C# 核心仍在运行，拒绝硬件接管")
	}
	w, err := win32.NewWorker()
	if err != nil {
		return nil, err
	}
	if writes {
		if err = w.AcquireHardware(); err != nil {
			w.Close()
			return nil, err
		}
	}
	n := &NativeBackend{worker: w, writes: writes, original: map[string]int{}}
	for _, d := range w.Displays {
		n.original[d.Id] = d.Brightness
	}
	return n, nil
}
func (n *NativeBackend) closeHardware() error {
	if n.closed {
		return nil
	}
	n.closed = true
	var errs []error
	// This experiment restores brightness to its takeover baseline (unlike product exit).
	if n.writes {
		if win32.ExistingCore() {
			errs = append(errs, errors.New("C# core appeared; refused restoration to avoid concurrent writes"))
		} else {
			// Remove compensation before restoring original colors, avoiding a
			// transient bright flash. Do not raise any backlight in this phase.
			for _, d := range n.worker.Displays {
				if value, ok := n.original[d.Id]; ok && value >= 0 && d.Brightness > value {
					if err := n.worker.Set(d.Id, value); err != nil {
						errs = append(errs, err)
					}
				}
			}
			if err := n.worker.RestoreColors(); err != nil {
				errs = append(errs, err)
			}
			for id, value := range n.original {
				if value >= 0 {
					if err := n.worker.Set(id, value); err != nil {
						errs = append(errs, err)
					}
				}
			}
		}
	}
	n.worker.Close()
	return errors.Join(errs...)
}
func runProbe() {
	w, err := win32.NewWorker()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer w.Close()
	json.NewEncoder(os.Stdout).Encode(map[string]any{"PhysicalConsole": win32.Console(), "ExistingCSharpCore": win32.ExistingCore(), "Displays": w.Displays})
}
