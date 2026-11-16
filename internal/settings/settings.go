// Package settings stores ASK Edition preferences and the label library through
// Fyne's platform preferences backend (%APPDATA% on Windows).
package settings

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"nelko-print/internal/fonts"
	"nelko-print/internal/label"
	"nelko-print/internal/tspl"
)

const SettingsKey = "ask.settings.v1"
const LibraryKey = "ask.library.v1"

type Preferences interface {
	String(string) string
	SetString(string, string)
}

type Settings struct {
	Language        string          `json:"language"`
	Font            string          `json:"font"`
	FontSize        float64         `json:"fontSize"`
	LabelSize       string          `json:"labelSize"`
	Density         int             `json:"density"`
	Threshold       uint8           `json:"threshold"`
	Copies          int             `json:"copies"`
	Text            string          `json:"text"`
	Orientation     int             `json:"orientation"`
	Invert          bool            `json:"invert"`
	TextInvert      bool            `json:"textInvert"`
	WordBreakOnly   bool            `json:"wordBreakOnly"`
	LastPrinter     string          `json:"lastPrinter"`
	LastPrinterName string          `json:"lastPrinterName"`
	ConnectionMode  string          `json:"connectionMode"`
	AutoReconnect   bool            `json:"autoReconnect"`
	Mode            string          `json:"mode"`
	Design          json.RawMessage `json:"design,omitempty"`
}

func Default() Settings {
	return Settings{Language: "da", Font: fonts.Names[0], FontSize: 24, LabelSize: tspl.Label14x40.Name, Density: 10, Threshold: 128, Copies: 1, AutoReconnect: true, Mode: "text"}
}

func (s *Settings) Normalize() {
	if s.Language != "en" {
		s.Language = "da"
	}
	if !fonts.Valid(s.Font) {
		s.Font = fonts.Names[0]
	}
	if math.IsNaN(s.FontSize) || math.IsInf(s.FontSize, 0) || s.FontSize < 4 || s.FontSize > 72 {
		s.FontSize = 24
	}
	validSize := false
	for _, size := range tspl.AllSizes {
		if size.Name == s.LabelSize {
			validSize = true
		}
	}
	if !validSize {
		s.LabelSize = tspl.Label14x40.Name
	}
	if s.Density < 0 || s.Density > 15 {
		s.Density = 10
	}
	if s.Copies < 1 || s.Copies > 999 {
		s.Copies = 1
	}
	if s.Orientation != 1 {
		s.Orientation = 0
	}
	if s.Mode != "designer" && s.Mode != "image" {
		s.Mode = "text"
	}
	if s.ConnectionMode != "bluetooth" {
		s.ConnectionMode = "port"
	}
}

func Load(p Preferences) (Settings, error) {
	s := Default()
	if data := p.String(SettingsKey); data != "" {
		if err := json.Unmarshal([]byte(data), &s); err != nil {
			return Default(), fmt.Errorf("read settings: %w", err)
		}
	}
	s.Normalize()
	return s, nil
}

func Save(p Preferences, s Settings) error {
	s.Normalize()
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	p.SetString(SettingsKey, string(data))
	return nil
}

type SavedLabel struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Template bool     `json:"template"`
	Settings Settings `json:"settings"`
}

type Library struct {
	Items []SavedLabel `json:"items"`
}

func LoadLibrary(p Preferences) (Library, error) {
	var l Library
	if data := p.String(LibraryKey); data != "" {
		if err := json.Unmarshal([]byte(data), &l); err != nil {
			return Library{}, fmt.Errorf("read library: %w", err)
		}
	}
	seen := make(map[string]bool)
	for i := range l.Items {
		item := &l.Items[i]
		if item.ID == "" || seen[item.ID] || strings.TrimSpace(item.Name) == "" {
			return Library{}, fmt.Errorf("invalid library entry")
		}
		seen[item.ID] = true
		item.Settings.Normalize()
		if item.Settings.Mode == "designer" {
			if _, err := label.UnmarshalDocument(item.Settings.Design); err != nil {
				return Library{}, fmt.Errorf("read %s: %w", item.Name, err)
			}
		}
	}
	return l, nil
}

func (l *Library) Save(p Preferences) error {
	data, err := json.Marshal(l)
	if err != nil {
		return err
	}
	p.SetString(LibraryKey, string(data))
	return nil
}

func (l *Library) Put(item SavedLabel) error {
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" || item.ID == "" {
		return fmt.Errorf("name and ID are required")
	}
	for i := range l.Items {
		if l.Items[i].ID == item.ID {
			l.Items[i] = item
			return nil
		}
	}
	l.Items = append(l.Items, item)
	return nil
}

func (l *Library) Delete(id string) bool {
	for i, item := range l.Items {
		if item.ID == id {
			l.Items = append(l.Items[:i], l.Items[i+1:]...)
			return true
		}
	}
	return false
}
