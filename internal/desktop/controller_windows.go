//go:build windows

package desktop

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

type command struct {
	fn   func() error
	done chan error
}
type rawInput struct {
	target                 edgeTarget
	valid, enhanced, wheel bool
	delta                  int
	foreground             uintptr
	x, y                   int32
	at, generation, epoch  uint64
}
type delivery struct {
	gesture    Gesture
	generation uint64
}

// Controller owns its native windows/hook on a single locked OS thread.
// onGesture runs serially on a separate goroutine and may call Configure or
// ShowFeedback. It must return; Close waits for it. Do not call Close from it.
type Controller struct {
	submit          sync.Mutex
	closing         atomic.Bool
	generation      atomic.Uint64
	pending         atomic.Int32
	jobs            chan command
	osdJobs         chan command
	osdDone         chan struct{}
	osdStarted      bool // Protected by submit; the shell must not block the hook loop.
	raw             chan rawInput
	gestures        chan delivery
	done, delivered chan struct{}
	errors          chan error
	closeOnce       sync.Once
	closeErr        error
	wake            windows.Handle
	onGesture       func(Gesture)

	// Everything below is accessed only on the native thread.
	window, hook, feedbackWindow uintptr
	keyboard                     windowShortcut
	options                      Options
	displays                     []Display
	accumulator                  accumulator
	epoch, accumulatorEpoch      uint64
	masks                        map[string]uintptr
	feedback                     Feedback
	feedbackScale                float64
	feedbackVisible              bool
	stopping                     bool
}

// New starts an idle native loop with one hidden notification window.
// It does not install a hook or create/show any overlay.
func New(onGesture func(Gesture)) (*Controller, error) {
	c := &Controller{
		jobs: make(chan command, 64), raw: make(chan rawInput, 128), gestures: make(chan delivery, 128),
		osdJobs: make(chan command, 16), osdDone: make(chan struct{}),
		done: make(chan struct{}), delivered: make(chan struct{}), errors: make(chan error, 16),
		onGesture: onGesture, masks: make(map[string]uintptr), options: Options{Modifier: "alt"},
	}
	ready := make(chan error, 1)
	go c.run(ready)
	if err := <-ready; err != nil {
		<-c.done
		return nil, err
	}
	go c.deliver()
	return c, nil
}

// Errors reports asynchronous loop/topology/cleanup failures. Calls returning an
// error report directly. This bounded diagnostic channel never blocks the hook.
func (c *Controller) Errors() <-chan error { return c.errors }
func (c *Controller) report(err error) {
	if err != nil {
		select {
		case c.errors <- err:
		default:
		}
	}
}

func (c *Controller) run(ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(c.done)
	defer close(c.gestures)
	defer close(c.errors)
	// Thread-local per-monitor-v2 awareness keeps coordinates physical without
	// changing the host's process-wide DPI mode.
	previous, _, err := setDPIContext.Call(^uintptr(3)) // -4 = PER_MONITOR_AWARE_V2.
	if previous == 0 {
		ready <- winError("SetThreadDpiAwarenessContext", err)
		return
	}
	defer setDPIContext.Call(previous)
	if err := registerClass(); err != nil {
		ready <- err
		return
	}
	var msg message
	peekMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 0)
	c.window, err = makeWindow(c, true)
	if err != nil {
		ready <- err
		return
	}
	c.wake, err = windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		c.destroy(c.window)
		ready <- err
		return
	}
	defer windows.CloseHandle(c.wake)
	threadID := windows.GetCurrentThreadId()
	threadOwners.Store(threadID, c)
	defer threadOwners.Delete(threadID)
	defer func() { c.closeErr = errors.Join(c.closeErr, c.cleanup()) }()
	ready <- nil
	for !c.stopping && !c.closing.Load() {
		r, _, err := waitMessages.Call(1, uintptr(unsafe.Pointer(&c.wake)), 0xffffffff, 0x04ff, 0x0004)
		if uint32(r) == 0xffffffff {
			c.closeErr = winError("MsgWaitForMultipleObjectsEx", err)
			c.report(c.closeErr)
			break
		}
		if c.closing.Load() {
			break
		}
		c.runJobs()
		c.keyboard.deliver()
		c.processRaw()
		for count := 0; count < 64 && !c.closing.Load(); count++ {
			r, _, _ := peekMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 1)
			if r == 0 {
				break
			}
			if msg.Message == 0x0012 {
				c.stopping = true
				break
			}
			translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
	c.closing.Store(true)
	c.generation.Add(1)
}

func (c *Controller) call(fn func() error) error {
	j := command{fn, make(chan error, 1)}
	c.submit.Lock()
	if c.closing.Load() {
		c.submit.Unlock()
		return ErrClosed
	}
	select {
	case c.jobs <- j:
	case <-c.done:
		c.submit.Unlock()
		return ErrClosed
	}
	err := windows.SetEvent(c.wake)
	c.submit.Unlock()
	if err != nil {
		return err
	}
	select {
	case err := <-j.done:
		return err
	case <-c.done:
		return ErrClosed
	}
}
func (c *Controller) runJobs() {
	for n := 0; n < 64; n++ {
		select {
		case j := <-c.jobs:
			if c.closing.Load() {
				j.done <- ErrClosed
				continue
			}
			j.done <- j.fn()
			if c.stopping {
				return
			}
		default:
			return
		}
	}
	if len(c.jobs) > 0 {
		windows.SetEvent(c.wake)
	}
}

func (c *Controller) deliver() {
	defer close(c.delivered)
	for d := range c.gestures {
		valid := !c.closing.Load() && d.generation == c.generation.Load() && isConsole()
		if d.gesture.Enhanced && !d.gesture.Brightness && foreground() != d.gesture.Foreground {
			valid = false
		}
		if valid && c.onGesture != nil {
			c.onGesture(d.gesture)
		}
		c.pending.Add(-1)
	}
}

// Configure is synchronous. It invalidates queued gestures and partial detents.
// A callback which has already started may finish. An empty supported list never
// intercepts the top edge; bottom-edge audio does not need hardware capability.
func (c *Controller) Configure(o Options) error {
	o.Modifier = strings.ToLower(strings.TrimSpace(o.Modifier))
	if o.Modifier == "" {
		o.Modifier = "alt"
	}
	if o.Modifier != "alt" && o.Modifier != "ctrl" && o.Modifier != "shift" {
		return errors.New("modifier must be alt, ctrl or shift")
	}
	o.Displays = append([]Display(nil), o.Displays...)
	seen := map[string]bool{}
	for _, d := range o.Displays {
		name := strings.ToUpper(d.Display)
		if name == "" || seen[name] {
			return errors.New("supported displays require distinct GDI names")
		}
		seen[name] = true
	}
	return c.call(func() error {
		if (o.TopEnabled || o.BottomEnabled) && !isConsole() {
			return ErrNotConsole
		}
		displays := append([]Display(nil), c.displays...)
		var err error
		if o.TopEnabled || o.BottomEnabled {
			displays, err = captureDisplays()
			if err != nil {
				return err
			}
		}
		c.generation.Add(1)
		c.accumulator.reset()
		c.epoch++
		c.options = o
		c.acceptDisplays(withCapabilities(displays, o))
		c.hideFeedback()
		if !o.TopEnabled && !o.BottomEnabled {
			err = c.removeHook()
		} else if c.hook == 0 {
			// Resolve the few read-only APIs before entering the low-level callback.
			for _, p := range []*windows.LazyProc{asyncKey, getForeground, nextHook, tickCount, consoleSession, getSystemMetrics} {
				if err = p.Find(); err != nil {
					return err
				}
			}
			module, _, _ := getModule.Call(0)
			h, _, e := setHook.Call(14, mouseCallback, module, 0)
			if h == 0 {
				return winError("SetWindowsHookExW", e)
			}
			c.hook = h
		}
		if timerErr := c.updateScopeTimer(); timerErr != nil {
			c.options.TopEnabled, c.options.BottomEnabled = false, false
			c.generation.Add(1)
			return errors.Join(err, timerErr, c.removeHook())
		}
		return err
	})
}

func withCapabilities(displays []Display, o Options) []Display {
	for i := range displays {
		displays[i].Brightness, displays[i].Temperature = false, false
		for _, cap := range o.Displays {
			if strings.EqualFold(displays[i].Display, cap.Display) {
				displays[i].Brightness, displays[i].Temperature = cap.Brightness, cap.Temperature
				break
			}
		}
	}
	return displays
}

// enqueueMouse performs only bounded input classification/snapshot/enqueue.
// Accumulation, user callbacks, COM, painting and all writes run elsewhere.
// Saturation passes the wheel through and starts a new accumulation epoch.
func (c *Controller) enqueueMouse(msg uintptr, m mouseInput) bool {
	if c.closing.Load() || (msg != wmMouseWheel && msg != wmMouseMove) {
		return false
	}
	enhanced, valid := mode(readKeys(), c.options.Modifier)
	target, hit := hitTest(c.displays, m.Point.X, m.Point.Y, c.options, enhanced)
	valid = valid && hit && m.Flags&3 == 0 && isConsole()
	fg := foreground()
	valid = valid && fg != 0
	if !valid {
		c.epoch++
	}
	wheel := msg == wmMouseWheel
	if wheel && (!valid || c.pending.Load() >= 128) {
		c.epoch++
		return false
	}
	r := rawInput{target: target, valid: valid, enhanced: enhanced, wheel: wheel,
		delta: int(int16(m.MouseData >> 16)), foreground: fg, x: m.Point.X, y: m.Point.Y,
		at: ticks(), generation: c.generation.Load(), epoch: c.epoch}
	if wheel {
		c.pending.Add(1)
	}
	select {
	case c.raw <- r:
		windows.SetEvent(c.wake)
		return wheel
	default:
		if wheel {
			c.pending.Add(-1)
		}
		c.epoch++
		return false
	}
}

func (c *Controller) processRaw() {
	console := isConsole()
	for {
		select {
		case r := <-c.raw:
			if r.epoch != c.accumulatorEpoch {
				c.accumulator.reset()
				c.accumulatorEpoch = r.epoch
			}
			if !console || r.generation != c.generation.Load() || c.closing.Load() {
				if r.wheel {
					c.pending.Add(-1)
				}
				continue
			}
			if !r.wheel {
				c.accumulator.observe(r.target, r.valid, r.enhanced, r.foreground)
				continue
			}
			steps := c.accumulator.add(r.target, r.valid, r.delta, r.at, r.enhanced, r.foreground)
			if steps == 0 {
				c.pending.Add(-1)
				continue
			}
			// Foreground can change between hook delivery and this message.
			if r.enhanced && !r.target.brightness && foreground() != r.foreground {
				c.accumulator.reset()
				c.pending.Add(-1)
				continue
			}
			c.gestures <- delivery{Gesture{r.target.display, r.target.brightness, r.enhanced, steps, r.foreground, r.x, r.y}, r.generation}
		default:
			return
		}
	}
}

func (c *Controller) removeHook() error {
	if c.hook == 0 {
		return nil
	}
	if err := boolCall(unhook, c.hook); err != nil {
		return err
	}
	c.hook = 0
	return nil
}

// Close prevents new input, removes the hook, destroys native windows, discards
// stale queued gestures, and waits for an in-flight onGesture to return.
func (c *Controller) Close() error {
	c.closeOnce.Do(func() {
		c.submit.Lock()
		c.closing.Store(true)
		c.generation.Add(1)
		if c.osdStarted {
			close(c.osdJobs)
		}
		var signalErr error
		select {
		case <-c.done:
		default:
			signalErr = windows.SetEvent(c.wake)
		}
		c.submit.Unlock()
		<-c.done
		<-c.delivered
		if c.osdStarted {
			<-c.osdDone
		}
		c.closeErr = errors.Join(c.closeErr, signalErr)
	})
	return c.closeErr
}

func (c *Controller) cleanup() error {
	var result error
	result = errors.Join(result, c.removeHook(), c.keyboard.remove())
	c.hideFeedback()
	killTimer.Call(c.window, scopeTimer)
	for _, h := range c.masks {
		result = errors.Join(result, c.destroy(h))
	}
	if c.feedbackWindow != 0 {
		result = errors.Join(result, c.destroy(c.feedbackWindow))
		c.feedbackWindow = 0
	}
	c.processRaw()
	for {
		select {
		case j := <-c.jobs:
			j.done <- ErrClosed
		default:
			return errors.Join(result, c.destroy(c.window))
		}
	}
}
func (c *Controller) destroy(hwnd uintptr) error {
	if hwnd == 0 {
		return nil
	}
	err := boolCall(destroyWindow, hwnd)
	if err == nil {
		windowOwners.Delete(hwnd)
	}
	return err
}

func (c *Controller) updateScopeTimer() error {
	if c.hook == 0 && len(c.masks) == 0 && !c.feedbackVisible {
		killTimer.Call(c.window, scopeTimer)
		return nil
	}
	r, _, err := setTimer.Call(c.window, scopeTimer, 1000, 0)
	if r == 0 {
		return winError("SetTimer(scope)", err)
	}
	return nil
}
func (c *Controller) stopScopeIfIdle() {
	if c.hook == 0 && len(c.masks) == 0 && !c.feedbackVisible {
		killTimer.Call(c.window, scopeTimer)
	}
}
func (c *Controller) checkScope() {
	c.processRaw()
	if !isConsole() {
		c.generation.Add(1)
		c.accumulator.reset()
		c.epoch++
		c.hideFeedback()
		for _, h := range c.masks {
			showWindow.Call(h, 0)
		}
		return
	}
	c.refreshDisplays()
}
func (c *Controller) refreshDisplays() {
	displays, err := captureDisplays()
	if err != nil {
		c.report(err)
		c.hideFeedback()
		c.report(c.clearMasks())
		c.displays = nil
		c.generation.Add(1)
		c.accumulator.reset()
		return
	}
	c.acceptDisplays(withCapabilities(displays, c.options))
}
func (c *Controller) acceptDisplays(displays []Display) {
	changed := len(displays) != len(c.displays)
	if !changed {
		for i := range displays {
			a, b := displays[i], c.displays[i]
			a.Brightness, a.Temperature, b.Brightness, b.Temperature = false, false, false, false
			if a != b {
				changed = true
				break
			}
		}
	}
	if !changed {
		c.displays = displays
		return
	}
	c.generation.Add(1)
	c.accumulator.reset()
	c.epoch++
	c.displays = displays
	c.hideFeedback()
	// Never let an old rectangle obscure a newly assigned primary display.
	for name, h := range c.masks {
		if err := c.destroy(h); err != nil {
			c.report(fmt.Errorf("remove stale mask %s: %w", name, err))
		} else {
			delete(c.masks, name)
		}
	}
}
