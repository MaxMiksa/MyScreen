//go:build windows

package desktop

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// These tests only create the hidden notification window. They NEVER enable
// input, show a feedback/mask window, call NativeOSD or write hardware/audio.
func TestIdleLifecycle(t *testing.T) {
	c, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.Configure(Options{Modifier: "CTRL"}); err != nil {
		t.Fatal(err)
	}
	if err := c.Configure(Options{Modifier: "win"}); err == nil {
		t.Fatal("unsupported modifier accepted")
	}
	if err := c.call(func() error {
		if c.hook != 0 || c.feedbackWindow != 0 || len(c.masks) != 0 {
			return errors.New("idle controller created an input/overlay resource")
		}
		if c.options.Modifier != "ctrl" {
			return errors.New("configure not applied on native thread")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Configure(Options{}); !errors.Is(err, ErrClosed) {
		t.Fatalf("Configure after Close: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if _, ok := <-c.Errors(); ok {
		t.Fatal("error stream not closed")
	}
	if _, ok := windowOwners.Load(c.window); ok {
		t.Fatal("native window owner retained after Close")
	}
}

func TestConcurrentIdleConfigureAndClose(t *testing.T) {
	c, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var wg sync.WaitGroup
	start := make(chan struct{})
	for n := 0; n < 24; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if err := c.Configure(Options{}); err != nil && !errors.Is(err, ErrClosed) {
				t.Errorf("Configure: %v", err)
			}
		}()
	}
	close(start)
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

func TestCloseWaitsForInFlightCallback(t *testing.T) {
	if !isConsole() {
		t.Skip("synthetic callback delivery is console-scoped")
	}
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock()
	c, err := New(func(Gesture) { close(started); <-release })
	if err != nil {
		t.Fatal(err)
	}
	// Feed the delivery queue directly: no hook or real input is involved.
	c.pending.Add(1)
	c.gestures <- delivery{generation: c.generation.Load()}
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		c.Close()
		t.Fatal("callback did not start")
	}
	closed := make(chan error, 1)
	go func() { closed <- c.Close() }()
	select {
	case <-c.done:
	case <-time.After(5 * time.Second):
		unblock()
		t.Fatal("native cleanup did not finish")
	}
	select {
	case <-closed:
		t.Fatal("Close returned before callback finished")
	default:
	}
	unblock()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not drain delivery")
	}
	if c.pending.Load() != 0 {
		t.Fatal("pending gesture count not drained")
	}
}

func TestReadOnlyDesktopSnapshot(t *testing.T) {
	if !isConsole() {
		t.Skip("desktop snapshot is console-scoped")
	}
	state, err := CaptureDesktopState()
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Displays) == 0 || state.Occupied == nil {
		t.Fatal("snapshot missing display/occupancy data")
	}
	secondary := map[string]bool{}
	for _, d := range state.Displays {
		if d.Display == "" || !d.Bounds.valid() || !d.WorkArea.valid() || d.DPI == 0 {
			t.Fatalf("invalid native display geometry: %+v", d)
		}
		if !d.Primary {
			secondary[d.Display] = true
		}
	}
	for name := range state.Occupied {
		if !secondary[name] {
			t.Fatal("occupancy contains a non-secondary display")
		}
	}
}

func TestFocusOccupancyIgnoresTransientOverlays(t *testing.T) {
	for _, tc := range []struct {
		name, class, exe string
		style            uint32
		ignored          bool
	}{
		{"computer use cursor", "CodexComputerUseCursorOverlay", "", 0x080800a8, true},
		{"snipping selection", "XamlWindow", "SnippingTool.exe", 0x00200188, true},
		{"snipping editor", "XamlWindow", "SnippingTool.exe", 0x00200100, false},
		{"another XAML window", "XamlWindow", "Application.exe", 0x00200188, false},
		{"unidentified XAML window", "XamlWindow", "", 0x00200188, false},
		{"interactive layered player", "Player", "", 0x08080088, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTransientFocusOverlay(tc.class, tc.style, func() string { return tc.exe }); got != tc.ignored {
				t.Fatalf("transient overlay = %v, want %v", got, tc.ignored)
			}
		})
	}
}

func TestCloseDrainsShellWorkerWithoutShowingOSD(t *testing.T) {
	c, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	// A synthetic job tests worker lifetime; shell COM is never invoked.
	c.submit.Lock()
	c.osdStarted = true
	go c.runOSD()
	c.osdJobs <- command{fn: func() error { close(started); <-release; return nil }, done: make(chan error, 1)}
	c.submit.Unlock()
	<-started
	closed := make(chan error, 1)
	go func() { closed <- c.Close() }()
	select {
	case <-c.done:
	case <-time.After(5 * time.Second):
		t.Fatal("native cleanup did not finish")
	}
	select {
	case <-closed:
		t.Fatal("Close did not wait for shell worker")
	default:
	}
	unblock()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("shell worker was not drained")
	}
}
