package settings

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"nelko-print/internal/label"
)

type memoryPreferences map[string]string

func (p memoryPreferences) String(k string) string { return p[k] }
func (p memoryPreferences) SetString(k, v string)  { p[k] = v }

func TestSettingsRoundTrip(t *testing.T) {
	p := memoryPreferences{}
	s := Default()
	s.Language = "en"
	s.Font = "Go Mono"
	s.FontSize = 17
	s.LabelSize = "14x75mm"
	s.Density = 15
	s.Text = "Æble\nØl & År"
	s.LastPrinter = "COM12"
	s.ConnectionMode = "bluetooth"
	s.LastPrinterName = "P21"
	s.AutoReconnect = false
	s.Mode = "designer"
	doc := &label.Document{Elements: []label.Element{label.NewText("ASK")}}
	s.Design, _ = label.MarshalDocument(doc)
	if err := Save(p, s); err != nil {
		t.Fatal(err)
	}
	got, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	var compact bytes.Buffer
	json.Compact(&compact, s.Design)
	s.Design = compact.Bytes()
	if !reflect.DeepEqual(s, got) {
		t.Fatalf("settings changed on restart: got %+v, want %+v", got, s)
	}
}

func TestDefaultsAndInvalidSettings(t *testing.T) {
	p := memoryPreferences{}
	got, err := Load(p)
	if err != nil || got.Language != "da" || !got.AutoReconnect {
		t.Fatalf("defaults: %+v %v", got, err)
	}
	p[SettingsKey] = `{"font":"missing.ttf","fontSize":-1,"labelSize":"bad","density":200,"copies":0,"language":"xx"}`
	got, err = Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Font != "Go Regular" || got.FontSize != 24 || got.LabelSize != "14x40mm" || got.Density != 10 || got.Copies != 1 {
		t.Fatalf("invalid defaults: %+v", got)
	}
	p[SettingsKey] = "broken"
	if _, err = Load(p); err == nil {
		t.Fatal("corruption must be reported")
	}
	if p[SettingsKey] != "broken" {
		t.Fatal("load modified corrupt preferences")
	}
}

func TestLibraryCRUDAndRestart(t *testing.T) {
	p := memoryPreferences{}
	l := Library{}
	s := Default()
	s.Mode = "designer"
	el := label.NewText("Før")
	el.FontName = "Go Bold"
	el.FontSize = 13
	s.Design, _ = label.MarshalDocument(&label.Document{Elements: []label.Element{el}})
	item := SavedLabel{ID: "a", Name: "  Test  ", Template: true, Settings: s}
	if err := l.Put(item); err != nil {
		t.Fatal(err)
	}
	item = l.Items[0]
	item.Name = "Omdøbt"
	el.Text = "Efter"
	item.Settings.Design, _ = label.MarshalDocument(&label.Document{Elements: []label.Element{el}})
	if err := l.Put(item); err != nil {
		t.Fatal(err)
	}
	if err := l.Save(p); err != nil {
		t.Fatal(err)
	}
	got, err := LoadLibrary(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].Name != "Omdøbt" || !got.Items[0].Template {
		t.Fatalf("library changed: %+v", got)
	}
	doc, err := label.UnmarshalDocument(got.Items[0].Settings.Design)
	if err != nil {
		t.Fatal(err)
	}
	text := doc.Elements[0].(*label.TextElement)
	if text.Text != "Efter" || text.FontName != "Go Bold" || text.FontSize != 13 {
		t.Fatalf("lost edit: %+v", text)
	}
	if !got.Delete("a") || got.Delete("missing") {
		t.Fatal("delete result")
	}
	if err := got.Save(p); err != nil {
		t.Fatal(err)
	}
	got, err = LoadLibrary(p)
	if err != nil || len(got.Items) != 0 {
		t.Fatalf("deleted entry returned: %+v %v", got, err)
	}
	if err := got.Put(SavedLabel{ID: "b", Name: " "}); err == nil {
		t.Fatal("accepted empty name")
	}
}

func TestLibraryCorruptionPreserved(t *testing.T) {
	for _, data := range []string{"invalid", `{"items":[{"id":"a","name":"Bad","settings":{"mode":"designer","design":{}}}]}`} {
		p := memoryPreferences{LibraryKey: data}
		if _, err := LoadLibrary(p); err == nil {
			t.Fatal("accepted corruption")
		}
		if p[LibraryKey] != data {
			t.Fatal("modified corrupt data")
		}
	}
}
