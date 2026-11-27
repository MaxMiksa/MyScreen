package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"myscreen/internal/audio"
	"myscreen/internal/desktop"
	"myscreen/internal/domain"
	"myscreen/internal/startup"
	"myscreen/internal/win32"
)

func NewNativeBackend(writes bool, roots ...string) (*NativeBackend, error) {
	root := profilePath(false)
	if len(roots) > 0 {
		root = roots[0]
	}
	store := domain.NewStore(filepath.Join(root, "native-settings.json"))
	settings, err := store.Load()
	if err != nil {
		return nil, err
	}
	n, err := newHardwareBackend(writes)
	if err != nil {
		return nil, err
	}
	n.store = store
	n.settings = settings
	n.targets = map[string]domain.DisplayTarget{}
	n.displayErrors = map[string]string{}
	n.backlights = map[string]int{}
	n.focus = map[string]domain.FocusState{}
	n.gestures = make(chan desktop.Gesture, 128)
	n.source = "手动调节"
	n.started = time.Now()
	for _, d := range n.worker.Displays {
		n.targets[d.Id] = domain.DisplayTarget{Brightness: d.Brightness}
		n.backlights[d.Id] = d.Brightness
	}
	if writes {
		if err = n.worker.ConfigureRecovery(filepath.Join(root, "native-colors.json")); err != nil {
			n.closeHardware()
			return nil, err
		}
		n.audio, err = audio.New()
		if err != nil {
			n.closeHardware()
			return nil, fmt.Errorf("native audio startup: %w", err)
		}
		n.desktop, err = desktop.New(func(g desktop.Gesture) {
			select {
			case n.gestures <- g:
			default:
			}
		})
		if err != nil {
			n.audio.Close()
			n.closeHardware()
			return nil, err
		}
		if err = n.configureDesktop(); err != nil {
			n.desktop.Close()
			n.audio.Close()
			n.closeHardware()
			return nil, err
		}
	}
	if len(n.settings.Scenes) == 0 {
		states := n.snapshotStates()
		n.settings.Scenes = domain.DefaultScenes(n.settings, states, [4]string{"default-morning", "default-midday", "default-dusk", "default-night"})
	}
	n.lastTopology, _ = win32.DisplaySignature()
	n.lastTopologyAt = time.Now()
	return n, nil
}
func (n *NativeBackend) SetTrace(trace func(string, map[string]any)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.trace = trace
	n.worker.Trace = trace
}
func (n *NativeBackend) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.stop != nil || !n.writes {
		return
	}
	n.stop = make(chan struct{})
	n.done = make(chan struct{})
	go n.run()
}
func (n *NativeBackend) run() {
	defer close(n.done)
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	var pending *desktop.Gesture
	process := func(g desktop.Gesture) {
		// Merge only adjacent gestures in the same direction and for the same
		// target. Reversals remain barriers so clamping at 0/100 is preserved.
	drain:
		for i := 0; i < cap(n.gestures); i++ {
			select {
			case next := <-n.gestures:
				if sameGesture(g, next) {
					g.Steps += next.Steps
					g.X, g.Y = next.X, next.Y
				} else {
					pending = &next
					break drain
				}
			default:
				break drain
			}
		}
		n.mu.Lock()
		if !n.closed && !n.closing {
			n.gesture(g)
		}
		n.mu.Unlock()
	}
	for {
		if pending != nil {
			select {
			case <-n.stop:
				return
			default:
			}
			g := *pending
			pending = nil
			process(g)
			continue
		}
		select {
		case <-n.stop:
			return
		case g := <-n.gestures:
			process(g)
		case err := <-n.desktop.Errors():
			if err != nil {
				n.mu.Lock()
				n.warning = err.Error()
				n.mu.Unlock()
			}
		case <-timer.C:
			n.mu.Lock()
			if !n.closed && !n.closing {
				n.worker.TraceID = fmt.Sprintf("timer-%d", time.Now().UnixNano())
				if err := n.tick(); err != nil {
					n.warning = err.Error()
				}
			}
			interval := time.Second
			if n.settings.FocusEnabled && !n.paused {
				interval = 50 * time.Millisecond
			}
			n.mu.Unlock()
			timer.Reset(interval)
		}
	}
}
func sameGesture(a, b desktop.Gesture) bool {
	return a.Display == b.Display && a.Brightness == b.Brightness && a.Enhanced == b.Enhanced && a.Foreground == b.Foreground && ((a.Steps > 0 && b.Steps > 0) || (a.Steps < 0 && b.Steps < 0))
}
func (n *NativeBackend) available() []string {
	ids := []string{}
	for _, d := range n.worker.Displays {
		if d.Method != "none" && d.Error == "" {
			ids = append(ids, d.Id)
		}
	}
	return ids
}
func (n *NativeBackend) configureDesktop() error {
	return n.configureDesktopSettings(n.settings)
}

func (n *NativeBackend) configureWindowShortcut(s domain.WindowShortcut) error {
	if n.desktop == nil || n.togglePanel == nil {
		return nil
	}
	if !s.Enabled {
		s.Key = 0
	}
	return n.desktop.ConfigureWindowShortcut(s.Key, s.Modifiers, n.togglePanel)
}

func (n *NativeBackend) configureDesktopSettings(settings domain.Settings) error {
	if n.desktop == nil {
		return nil
	}
	displays := []desktop.Display{}
	for _, d := range n.worker.Displays {
		displays = append(displays, desktop.Display{Display: d.GDI, Brightness: d.Method != "none" && d.Error == "", Temperature: d.ColorMode.SDR && d.ColorError == ""})
	}
	return n.desktop.Configure(desktop.Options{TopEnabled: settings.TopEdgeBrightnessEnabled, BottomEnabled: settings.BottomEdgeVolumeEnabled, Modifier: settings.EdgeWheelModifier, Displays: displays})
}
func (n *NativeBackend) saveSettings(next domain.Settings) error {
	// Lab saves preserve a copied preference but never register a startup entry.
	if !n.production && next.LaunchOnStartup != n.settings.LaunchOnStartup {
		return errors.New("验收期间不修改登录自启；正式切换由部署步骤统一处理")
	}
	// Validate and configure native input first. A hook failure must not be
	// presented as a failed save after the settings have already been committed.
	if err := domain.Validate(next); err != nil {
		return err
	}
	if err := n.configureDesktopSettings(next); err != nil {
		return err
	}
	shortcutChanged := next.WindowShortcut != n.settings.WindowShortcut
	rollbackInput := func() error {
		err := n.configureDesktop()
		if shortcutChanged {
			err = errors.Join(err, n.configureWindowShortcut(n.settings.WindowShortcut))
		}
		return err
	}
	if shortcutChanged {
		if err := n.configureWindowShortcut(next.WindowShortcut); err != nil {
			return errors.Join(err, rollbackInput())
		}
	}
	var startupBefore *startup.State
	if n.production && next.LaunchOnStartup != n.settings.LaunchOnStartup {
		state, err := startup.Snapshot()
		if err != nil {
			return errors.Join(err, rollbackInput())
		}
		executable, err := os.Executable()
		if err == nil {
			err = startup.Configure(executable, next.LaunchOnStartup)
		}
		if err != nil {
			return errors.Join(err, rollbackInput())
		}
		startupBefore = &state
	}
	saved, err := n.store.Save(next)
	if err != nil {
		if startupBefore != nil {
			err = errors.Join(err, startup.Restore(*startupBefore))
		}
		return errors.Join(err, rollbackInput())
	}
	n.schedule = domain.ReconcileSchedule(n.schedule, n.settings, saved)
	n.settings = saved
	n.warning = ""
	return nil
}
func (n *NativeBackend) holdManual() {
	n.schedule = domain.HoldManual(n.settings, n.schedule, time.Now())
	n.source = "手动调节"
	if n.schedule.ManualUntil != nil {
		n.source = "手动 · 保持至 " + n.schedule.ManualUntil.Format("15:04")
	}
}
func (n *NativeBackend) setBrightness(id string, value int) error {
	values, err := domain.LinkedBrightness(n.settings, n.available(), n.targets, id, value)
	if err != nil {
		return err
	}
	for target, brightness := range values {
		state := n.targets[target]
		state.Brightness = brightness
		n.targets[target] = state
	}
	n.holdManual()
	n.focus = map[string]domain.FocusState{}
	return nil
}
func (n *NativeBackend) setTemperature(id string, value *int) error {
	found := false
	for _, d := range n.worker.Displays {
		if id != "" && d.Id != id {
			continue
		}
		found = true
		if value != nil && (!d.ColorMode.SDR || d.ColorError != "") {
			return errors.New("色温暂不可用，可在设置中重新检测")
		}
		target := n.targets[d.Id]
		target.Kelvin = domain.QuantizeKelvin(value)
		n.targets[d.Id] = target
	}
	if !found {
		return errors.New("显示器不存在")
	}
	n.holdManual()
	return nil
}
func (n *NativeBackend) apply(forceColor bool) error {
	if !n.writes {
		return nil
	}
	if !win32.Console() || win32.ExistingCore() {
		return errors.New("失去物理控制台独占控制，拒绝写入")
	}
	var failures []error
	for _, d := range n.worker.Displays {
		target, ok := n.targets[d.Id]
		if !ok {
			continue
		}
		var failure error
		activeKelvin, _ := d.Kelvin().(int)
		requested := 6500
		if target.Kelvin != nil {
			requested = *domain.QuantizeKelvin(target.Kelvin)
		}
		// Withdraw excess backlight before moving toward cooler/original colors.
		if activeKelvin != 0 && requested > activeKelvin {
			resolved := domain.ResolveBrightnessTargets(n.settings, domain.CompensationScope{Id: d.Id, IsInternal: d.ColorMode.Internal, IsPrimary: d.Primary, IdentityConfirmed: true, SupportsColor: d.ColorMode.SDR && d.ColorError == "", ColorActive: requested < 6500}, target.Brightness, target.Kelvin)
			if d.Method != "none" && d.Brightness > resolved.BacklightTarget {
				if err := n.worker.Set(d.Id, resolved.BacklightTarget); err != nil {
					n.displayErrors[d.Id] = err.Error()
					failures = append(failures, err)
					continue
				}
			}
		}
		if requested < 6500 {
			if d.ColorConflict() {
				failure = errors.New("外部颜色已变化，重新检测后再调色")
			} else if forceColor || activeKelvin != requested {
				failure = n.worker.Temperature(d.Id, requested)
			}
		} else if activeKelvin != 0 {
			failure = n.worker.Temperature(d.Id, 6500)
		}
		colorActive := failure == nil && d.Kelvin() != nil
		resolved := domain.ResolveBrightnessTargets(n.settings, domain.CompensationScope{Id: d.Id, IsInternal: d.ColorMode.Internal, IsPrimary: d.Primary, IdentityConfirmed: true, SupportsColor: d.ColorMode.SDR && d.ColorError == "", ColorActive: colorActive}, target.Brightness, target.Kelvin)
		n.backlights[d.Id] = resolved.BacklightTarget
		if d.Method != "none" && d.Brightness != resolved.BacklightTarget {
			failure = errors.Join(failure, n.worker.Set(d.Id, resolved.BacklightTarget))
		}
		if failure != nil {
			n.displayErrors[d.Id] = failure.Error()
			failures = append(failures, failure)
		} else {
			delete(n.displayErrors, d.Id)
		}
		if resolved.Limited {
			n.warning = "内屏背光已达上限，当前色温下无法完全补偿亮度"
		}
	}
	return errors.Join(failures...)
}
func (n *NativeBackend) rescan() error {
	if n.desktop != nil {
		if err := n.desktop.Configure(desktop.Options{Modifier: n.settings.EdgeWheelModifier}); err != nil {
			return err
		}
		n.desktop.SetMasks(nil)
	}
	err := n.worker.Rescan()
	if err != nil {
		return err
	}
	for _, d := range n.worker.Displays {
		if _, ok := n.original[d.Id]; !ok {
			n.original[d.Id] = d.Brightness
		}
		if _, ok := n.targets[d.Id]; !ok {
			n.targets[d.Id] = domain.DisplayTarget{Brightness: d.Brightness}
		}
	}
	n.focus = map[string]domain.FocusState{}
	n.lastTopology, _ = win32.DisplaySignature()
	return errors.Join(err, n.configureDesktop(), n.apply(true))
}
func (n *NativeBackend) tick() error {
	now := time.Now()
	resume := !n.lastTick.IsZero() && now.Sub(n.lastTick) > 5*time.Second
	n.lastTick = now
	if !win32.Console() {
		if n.desktop != nil {
			n.desktop.Configure(desktop.Options{Modifier: n.settings.EdgeWheelModifier})
			n.desktop.SetMasks(nil)
		}
		return errors.New("不在物理控制台，已撤销输入和遮罩")
	}
	if resume || now.Sub(n.lastTopologyAt) >= 2*time.Second {
		n.lastTopologyAt = now
		signature, err := win32.DisplaySignature()
		if err != nil {
			return err
		}
		if resume || signature != n.lastTopology {
			if err = n.rescan(); err != nil {
				return err
			}
		}
	}
	plan, err := domain.EvaluateSchedule(n.settings, n.schedule, now, n.available(), n.targets)
	if err != nil {
		return err
	}
	if plan.BehaviorChanged {
		if err = n.saveSettings(plan.Settings); err != nil {
			return err
		}
	}
	if plan.ApplyBehavior {
		n.focus = map[string]domain.FocusState{}
	}
	n.schedule = plan.State
	n.targets = plan.Targets
	if plan.Plan != nil && !plan.Held {
		n.source = "时间计划 · " + plan.Plan.Scene.Name
	}
	check := now.Sub(n.lastColorCheck) >= 3*time.Second
	if check {
		n.lastColorCheck = now
	}
	return errors.Join(n.tickFocus(time.Now()), n.apply(check))
}
func (n *NativeBackend) snapshotStates() []domain.DisplayState {
	states := []domain.DisplayState{}
	for _, d := range n.worker.Displays {
		target := n.targets[d.Id]
		var text *string
		if v := n.displayErrors[d.Id]; v != "" {
			copy := v
			text = &copy
		} else if d.Error != "" {
			copy := d.Error
			text = &copy
		}
		backlight := n.backlights[d.Id]
		states = append(states, domain.DisplayState{Id: d.Id, Name: d.Name, Brightness: target.Brightness, CanAdjust: n.writes && d.Method != "none" && d.Error == "", CanAdjustColor: n.writes && d.ColorMode.SDR && d.ColorError == "", Kelvin: target.Kelvin, Error: text, IsPrimary: d.Primary, BacklightTarget: &backlight})
	}
	return states
}
func (n *NativeBackend) snapshot() (map[string]any, error) {
	var warning *string
	if text := n.currentWarning(); text != "" {
		copy := text
		warning = &copy
	}
	if !n.writes {
		copy := "Go 原生只读 · 未启用硬件写入"
		warning = &copy
	}
	language := "zh-CN"
	data, err := json.Marshal(domain.Snapshot{
		Displays: n.snapshotStates(), Settings: n.settings, FocusPaused: n.paused,
		FocusPausedByComputerUse: n.settings.FocusEnabled && !n.paused && !n.lastComputerUse.IsZero(),
		Source:                   n.source, Warning: warning, SystemLanguage: &language, SolarTimes: n.schedule.Solar.Today(n.settings.SolarLocation, time.Now()),
	})
	if err != nil {
		return nil, err
	}
	var result map[string]any
	err = json.Unmarshal(data, &result)
	return result, err
}
func (n *NativeBackend) Request(r Request) (map[string]any, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.closed || n.closing {
		return nil, errors.New("正在退出")
	}
	n.worker.TraceID = r.TraceID
	revisionBefore := n.settings.Revision
	if r.Action != "snapshot" && !n.writes {
		return nil, errors.New("Go 原生只读，拒绝控制操作")
	}
	var err error
	switch r.Action {
	case "snapshot":
	case "brightness":
		if r.Value == nil || *r.Value < 0 || *r.Value > 100 {
			return nil, errors.New("无效亮度")
		}
		err = n.setBrightness(r.Id, int(math.RoundToEven(*r.Value)))
	case "temperature":
		if r.Value == nil || *r.Value < 2700 || *r.Value > 6500 {
			return nil, errors.New("无效色温")
		}
		value := int(*r.Value)
		err = n.setTemperature(r.Id, &value)
	case "restore-color":
		err = n.setTemperature(r.Id, nil)
	case "pause":
		n.paused = !n.paused
	case "resume-plan":
		n.schedule = domain.ResumePlan()
		n.source = "手动调节"
		err = n.tick()
	case "link":
		var next domain.Settings
		var targets map[string]domain.DisplayTarget
		next, targets, err = domain.SetLinked(n.settings, r.Value != nil && *r.Value == 1, n.available(), n.targets)
		if err == nil {
			err = n.saveSettings(next)
			if err == nil {
				n.targets = targets
			}
		}
	case "scene":
		found := false
		for _, scene := range n.settings.Scenes {
			if scene.Id != r.Id {
				continue
			}
			found = true
			var next domain.Settings
			var targets map[string]domain.DisplayTarget
			var changed bool
			next, targets, changed, err = domain.ApplyScene(n.settings, scene, n.available(), n.targets)
			if err == nil && changed {
				err = n.saveSettings(next)
			}
			if err == nil {
				n.targets = targets
				n.focus = map[string]domain.FocusState{}
				n.holdManual()
				n.source = "场景 · " + scene.Name
			}
			break
		}
		if !found {
			err = errors.New("场景不存在")
		}
	case "rescan":
		err = n.rescan()
	case "save":
		if r.Settings == nil {
			return nil, errors.New("缺少设置")
		}
		var data []byte
		data, err = json.Marshal(r.Settings)
		if err == nil {
			var next domain.Settings
			next, err = domain.DecodeSettings(data)
			if err == nil {
				if next.Revision != n.settings.Revision {
					err = errors.New("配置已变化，保留草稿并重新载入")
				} else {
					old := n.settings
					err = n.saveSettings(next)
					if err == nil && (old.Linked != next.Linked || !equalCalibration(old, next)) {
						n.targets, err = domain.ApplyCalibratedLink(n.settings, n.available(), n.targets, nil)
					}
				}
			}
		}
	default:
		err = fmt.Errorf("不支持的操作：%s", r.Action)
	}
	if err != nil {
		return nil, err
	}
	if r.Action != "snapshot" {
		if err = n.apply(false); err != nil {
			if n.settings.Revision != revisionBefore {
				n.warning = "设置已保存；应用失败：" + err.Error()
				return n.snapshot()
			}
			return nil, err
		}
		if n.desktop != nil && (r.Action == "pause" || r.Action == "save" || r.Action == "scene") {
			if err = n.tick(); err != nil {
				if n.settings.Revision != revisionBefore {
					n.warning = "设置已保存；应用失败：" + err.Error()
					return n.snapshot()
				}
				return nil, err
			}
		}
	}
	return n.snapshot()
}
func equalCalibration(a, b domain.Settings) bool {
	x, _ := json.Marshal(a.BrightnessLinkCalibration)
	y, _ := json.Marshal(b.BrightnessLinkCalibration)
	return string(x) == string(y)
}
func (n *NativeBackend) gesture(g desktop.Gesture) {
	if !n.writes || !win32.Console() {
		return
	}
	if (g.Brightness && !n.settings.TopEdgeBrightnessEnabled) || (!g.Brightness && !n.settings.BottomEdgeVolumeEnabled) {
		return
	}
	traceID := fmt.Sprintf("wheel-%d", time.Now().UnixNano())
	n.worker.TraceID = traceID
	start := time.Now()
	var err error
	feedbackHandled := false
	if n.trace != nil {
		n.trace("gesture-received", map[string]any{"traceId": traceID, "brightness": g.Brightness, "enhanced": g.Enhanced, "steps": g.Steps})
	}
	if g.Brightness {
		id := ""
		for _, d := range n.worker.Displays {
			if d.GDI == g.Display {
				id = d.Id
				break
			}
		}
		if id == "" {
			return
		}
		target := n.targets[id]
		if g.Enhanced {
			value := 6500
			if target.Kelvin != nil {
				value = *target.Kelvin
			}
			value = max(2700, min(6500, value+g.Steps*100))
			if n.settings.Linked {
				err = n.setTemperature("", &value)
			} else {
				err = n.setTemperature(id, &value)
			}
			if err == nil {
				n.desktop.ShowFeedback(desktop.Feedback{Label: "色温", Kelvin: value, Display: g.Display, X: g.X, Y: g.Y})
			}
		}
		if !g.Enhanced {
			err = n.setBrightness(id, max(0, min(100, target.Brightness+g.Steps*2)))
		}
		if err == nil {
			err = n.apply(false)
		}
		if err == nil && !g.Enhanced {
			err = n.desktop.NativeOSD(true)
		}
	} else if n.audio != nil {
		var state audio.State
		if g.Enhanced {
			state, err = n.audio.ApplicationVolume(g.Foreground)
			if err == nil {
				state, err = n.audio.SetApplicationVolume(g.Foreground, max(0, min(100, state.Percent+float64(g.Steps*2))))
			}
			// Discovery usually takes a few milliseconds. Show its final result
			// once: repeatedly opening a pending bar then hiding it on NoTarget
			// creates a flashing zero-looking indicator while the panel is closed.
			if missingApplicationAudio(err) && audio.ForegroundWindow() == g.Foreground && n.notify != nil {
				n.desktop.HideFeedback()
				if !time.Now().Before(n.nextAudioNotice) {
					n.nextAudioNotice = time.Now().Add(5 * time.Second)
					n.notify("当前软件没有可调节的音频会话，请先播放音频。")
				}
				feedbackHandled = true
			} else if (err == nil || missingApplicationAudio(err)) && audio.ForegroundWindow() == g.Foreground {
				feedback := applicationVolumeFeedback(state, err)
				feedback.Display, feedback.X, feedback.Y = g.Display, g.X, g.Y
				feedbackErr := n.desktop.ShowFeedback(feedback)
				feedbackHandled = feedbackErr == nil
				err = errors.Join(err, feedbackErr)
			}
		}
		if !g.Enhanced {
			state, err = n.audio.SystemVolume()
			if err == nil {
				_, err = n.audio.SetSystemVolume(max(0, min(100, state.Percent+float64(g.Steps*2))))
			}
			if err == nil {
				err = n.desktop.NativeOSD(false)
			}
		}
	}
	if err != nil {
		n.warning = err.Error()
		if missingApplicationAudio(err) {
			n.warning = "当前软件没有可调节的音频会话，请先播放音频。"
		}
		if !feedbackHandled {
			n.desktop.HideFeedback()
		}
	} else if n.warning == "当前软件没有可调节的音频会话，请先播放音频。" {
		n.warning = ""
	}
	if n.trace != nil {
		fields := map[string]any{"traceId": traceID, "elapsedMs": time.Since(start).Milliseconds(), "feedbackHandled": feedbackHandled}
		if err != nil {
			fields["error"] = err.Error()
		}
		n.trace("gesture-complete", fields)
	}
}
func (n *NativeBackend) Close() error {
	n.closeOnce.Do(func() {
		n.mu.Lock()
		n.closing = true
		if n.stop != nil {
			close(n.stop)
		}
		done := n.done
		n.mu.Unlock()
		if done != nil {
			<-done
		}
		n.mu.Lock()
		defer n.mu.Unlock()
		var failures []error
		if n.desktop != nil {
			failures = append(failures, n.desktop.Close())
		}
		if n.audio != nil {
			failures = append(failures, n.audio.Close())
		}
		if n.production && n.writes {
			for id, target := range n.targets {
				n.original[id] = target.Brightness
			}
		}
		failures = append(failures, n.closeHardware())
		n.closeError = errors.Join(failures...)
	})
	return n.closeError
}
