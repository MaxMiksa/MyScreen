// Package domain implements product rules without hardware or host side effects.
// Use DefaultSettings (or DecodeSettings), rather than Settings{}, for defaults.
package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
)

// Field names intentionally match the PascalCase C#/React protocol. Do not add
// omitempty to ControlledDisplayIds: nil means all, [] means none.
type Settings struct {
	SchemaVersion                     int
	Revision                          int64
	Language                          string
	Linked                            bool
	LinkWeights                       map[string]float64
	BrightnessLinkCalibration         *BrightnessLinkCalibration
	TemperatureBrightnessCalibrations map[string]TemperatureBrightnessCalibration
	FocusEnabled                      bool
	ControlledDisplayIds              []string
	KeepSecondaryAwakeOnFullscreen    bool
	FocusDelaySeconds                 int
	ScheduleEnabled                   bool
	SolarLocation                     *SolarLocation `json:",omitempty"`
	ScheduleTransitionMinutes         *int           `json:",omitempty"`
	LaunchOnStartup                   bool
	WindowShortcut                    WindowShortcut
	TopEdgeBrightnessEnabled          bool
	BottomEdgeVolumeEnabled           bool
	EdgeWheelModifier                 string
	Scenes                            []Scene
	Schedule                          []PlanPoint
}

type DisplayState struct {
	Id              string
	Name            string
	Brightness      int
	CanAdjust       bool
	CanAdjustColor  bool
	Kelvin          *int
	Error           *string
	IsPrimary       bool
	BacklightTarget *int
}

type Snapshot struct {
	Displays                 []DisplayState
	Settings                 Settings
	FocusPaused              bool
	FocusPausedByComputerUse bool `json:",omitempty"`
	Source                   string
	Error                    *string
	Warning                  *string
	SystemLanguage           *string
	SolarTimes               *SolarTimes `json:",omitempty"`
}

type Request struct {
	Action   string
	Id       *string
	Value    *int
	Settings *Settings
}

type DisplayTarget struct {
	Brightness int
	Kelvin     *int
}
type Scene struct {
	Id          string
	Name        string
	DefaultName *string
	Targets     map[string]DisplayTarget
	Behavior    *SceneBehavior
}
type SceneBehavior struct {
	Linked               bool
	LinkWeights          map[string]float64
	FocusEnabled         bool
	ControlledDisplayIds []string
}
type PlanPoint struct {
	TimeKind          string `json:",omitempty"`
	OffsetMinutes     int    `json:",omitempty"`
	MinuteOfDay       int
	SceneId           string
	TransitionMinutes int
}

func DefaultSettings() Settings {
	return Settings{SchemaVersion: 1, Language: "system", LinkWeights: map[string]float64{},
		TemperatureBrightnessCalibrations: map[string]TemperatureBrightnessCalibration{},
		KeepSecondaryAwakeOnFullscreen:    true, FocusDelaySeconds: 180, EdgeWheelModifier: "alt",
		WindowShortcut: WindowShortcut{Enabled: true, Modifiers: 8, Key: 'W'},
		Scenes:         []Scene{}, Schedule: []PlanPoint{}}
}

// UnmarshalJSON supplies the same missing-field defaults as C# initializers.
// DecodeSettings additionally validates and fills absent scene behaviors.
func (s *Settings) UnmarshalJSON(data []byte) error {
	if err := rejectNullValues(data, reflect.TypeFor[Settings]()); err != nil {
		return err
	}
	type plain Settings
	v := plain(DefaultSettings())
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*s = Settings(v)
	return nil
}

// encoding/json silently accepts null into scalar/struct Go values; C# does
// not. Reject those before decoding so null cannot become a valid 0% target or
// silently turn a saved enabled flag into a default. Pointer/slice/map nulls
// retain their protocol semantics and collection validation runs separately.
func rejectNullValues(data []byte, typ reflect.Type) error {
	if typ == reflect.TypeFor[json.RawMessage]() {
		return nil
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		switch typ.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Map:
			return nil
		}
		return fmt.Errorf("null is not valid for %s", typ.Name())
	}
	if typ.Kind() == reflect.Pointer {
		return rejectNullValues(data, typ.Elem())
	}
	switch typ.Kind() {
	case reflect.Struct:
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil {
			return err
		}
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			for key, value := range fields {
				if strings.EqualFold(key, field.Name) {
					if err := rejectNullValues(value, field.Type); err != nil {
						return fmt.Errorf("%s: %w", field.Name, err)
					}
				}
			}
		}
	case reflect.Slice:
		var values []json.RawMessage
		if err := json.Unmarshal(data, &values); err != nil {
			return err
		}
		for _, value := range values {
			if err := rejectNullValues(value, typ.Elem()); err != nil {
				return err
			}
		}
	case reflect.Map:
		var values map[string]json.RawMessage
		if err := json.Unmarshal(data, &values); err != nil {
			return err
		}
		for key, value := range values {
			if err := rejectNullValues(value, typ.Elem()); err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
		}
	}
	return nil
}

func (b *SceneBehavior) UnmarshalJSON(data []byte) error {
	type plain SceneBehavior
	v := plain{LinkWeights: map[string]float64{}}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*b = SceneBehavior(v)
	return nil
}

func (s *Scene) UnmarshalJSON(data []byte) error {
	type plain Scene
	v := plain{Name: "新场景", Targets: map[string]DisplayTarget{}}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*s = Scene(v)
	return nil
}

func DecodeSettings(data []byte) (Settings, error) {
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, fmt.Errorf("decode settings: %w", err)
	}
	if err := Validate(s); err != nil {
		return Settings{}, err
	}
	return WithSceneDefaults(s), nil
}

// NewPlanPoint supplies the product's new-node fade default; persisted nodes
// with an explicit zero still switch immediately.
func NewPlanPoint(minuteOfDay int, sceneId string) PlanPoint {
	return PlanPoint{MinuteOfDay: minuteOfDay, SceneId: sceneId, TransitionMinutes: 30}
}

func SceneName(scene Scene, preference, systemLanguage string) string {
	if scene.DefaultName == nil || scene.Name != *scene.DefaultName || ResolveLanguage(preference, systemLanguage) == "zh-CN" {
		return scene.Name
	}
	switch scene.Name {
	case "日常":
		return "Day"
	case "早晨":
		return "Morning"
	case "中午":
		return "Midday"
	case "傍晚":
		return "Dusk"
	case "夜间":
		return "Night"
	case "专注":
		return "Focus"
	}
	return scene.Name
}

// Clone makes an independent draft, including pointers and nil/empty selections.
func (s Settings) Clone() Settings {
	s.ScheduleTransitionMinutes = clonePointer(s.ScheduleTransitionMinutes)
	s.SolarLocation = clonePointer(s.SolarLocation)
	s.LinkWeights = maps.Clone(s.LinkWeights)
	s.ControlledDisplayIds = slices.Clone(s.ControlledDisplayIds)
	if s.BrightnessLinkCalibration != nil {
		c := *s.BrightnessLinkCalibration
		c.Points = slices.Clone(c.Points)
		s.BrightnessLinkCalibration = &c
	}
	s.TemperatureBrightnessCalibrations = maps.Clone(s.TemperatureBrightnessCalibrations)
	for id, c := range s.TemperatureBrightnessCalibrations {
		c.Points = slices.Clone(c.Points)
		s.TemperatureBrightnessCalibrations[id] = c
	}
	s.Scenes = slices.Clone(s.Scenes)
	for i := range s.Scenes {
		s.Scenes[i].DefaultName = clonePointer(s.Scenes[i].DefaultName)
		s.Scenes[i].Targets = CloneTargets(s.Scenes[i].Targets)
		if b := s.Scenes[i].Behavior; b != nil {
			c := b.Clone()
			s.Scenes[i].Behavior = &c
		}
	}
	s.Schedule = slices.Clone(s.Schedule)
	return s
}

func clonePointer[T any](v *T) *T {
	if v == nil {
		return nil
	}
	c := *v
	return &c
}
func CloneTargets(targets map[string]DisplayTarget) map[string]DisplayTarget {
	r := maps.Clone(targets)
	for id, t := range r {
		t.Kelvin = clonePointer(t.Kelvin)
		r[id] = t
	}
	return r
}

func ResolveLanguage(preference, systemLanguage string) string {
	if preference == "zh-CN" || preference == "en" {
		return preference
	}
	if len(systemLanguage) >= 2 && strings.EqualFold(systemLanguage[:2], "zh") {
		return "zh-CN"
	}
	return "en"
}
