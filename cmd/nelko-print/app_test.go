package main

import (
	"bytes"
	"errors"
	"image/png"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"nelko-print/internal/label"
	"nelko-print/internal/settings"
)

func testApp(t *testing.T) *App {
	t.Helper()
	fa := test.NewApp()
	w := fa.NewWindow("ASK test")
	a := &App{fyneApp: fa, window: w}
	a.loadSettings()
	w.SetContent(a.buildUI())
	a.renderActivePreview()
	t.Cleanup(func() { a.cleanup(); w.Close(); fa.Quit() })
	return a
}

func pngBytes(t *testing.T, a *App) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, a.sourceImg); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestDesignerTextEditingKeepsFocusAndPersists(t *testing.T) {
	a := testApp(t)
	a.tabs.SelectIndex(2)
	el := label.NewText("ASK")
	a.designCanvas.Add(el)
	form := a.designProps.Objects[0].(*widget.Form)
	var entry *widget.Entry
	for _, item := range form.Items {
		if e, ok := item.Widget.(*widget.Entry); ok {
			entry = e
		}
	}
	if entry == nil {
		t.Fatal("no editable text field")
	}
	before := pngBytes(t, a)
	a.window.Canvas().Focus(entry)
	entry.SetText("")
	test.Type(entry, "ASK – ÆØÅ\nP21")
	if el.Text != "ASK – ÆØÅ\nP21" {
		t.Fatalf("text not edited: %q", el.Text)
	}
	if a.window.Canvas().Focused() != entry {
		t.Fatal("typing lost keyboard focus")
	}
	if a.designProps.Objects[0] != form {
		t.Fatal("typing replaced the property editor")
	}
	if bytes.Equal(before, pngBytes(t, a)) {
		t.Fatal("edit did not change rendered label")
	}
	restored, err := settings.Load(a.fyneApp.Preferences())
	if err != nil {
		t.Fatal(err)
	}
	doc, err := label.UnmarshalDocument(restored.Design)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Elements[0].(*label.TextElement).Text != el.Text {
		t.Fatal("edit not autosaved")
	}
	// Selection, loading and continued editing must also work.
	a.designCanvas.SetDocument(doc)
	a.designCanvas.selectElement(doc.Elements[0])
	form = a.designProps.Objects[0].(*widget.Form)
	for _, item := range form.Items {
		if e, ok := item.Widget.(*widget.Entry); ok {
			entry = e
		}
	}
	entry.CursorRow = 1
	entry.CursorColumn = 3
	test.Type(entry, "!")
	if doc.Elements[0].(*label.TextElement).Text != el.Text+"!" {
		t.Fatal("loaded label cannot be edited")
	}
}

func TestRestartAndLanguageSwitch(t *testing.T) {
	a := testApp(t)
	a.textEntry.SetText("Dansk æøå")
	a.fontName = "Go Mono"
	a.fontSize = 13
	a.density = 7
	a.labelSize.Name = "14x50mm"
	a.saveSettings()
	s, err := settings.Load(a.fyneApp.Preferences())
	if err != nil {
		t.Fatal(err)
	}
	a.applySettings(s)
	a.window.SetContent(a.buildUI())
	a.renderActivePreview()
	if a.textEntry.Text != "Dansk æøå" || a.fontName != "Go Mono" || a.fontSize != 13 || a.labelSize.Name != "14x50mm" || a.density != 7 {
		t.Fatal("settings lost on rebuild")
	}
	a.changeLanguage("en")
	if a.tabs.Items[1].Text != "Text" || a.tr("Connect") != "Connect" {
		t.Fatal("English not applied")
	}
	if a.textEntry.Text != "Dansk æøå" {
		t.Fatal("language switch lost text")
	}
	a.changeLanguage("da")
	if a.tabs.Items[1].Text != "Tekst" || a.tr("Connect") != "Tilslut" {
		t.Fatal("Danish not applied")
	}
	// Translated select values must still call the original enum callbacks.
	textTab := a.tabs.Items[1].Content.(*fyne.Container)
	form := textTab.Objects[1].(*widget.Form)
	orientation := form.Items[1].Widget.(*widget.Select)
	orientation.SetSelected("Lodret")
	if a.orientation != 1 {
		t.Fatal("translated orientation did not update")
	}
}

func TestSwitchingModesClearsStalePreview(t *testing.T) {
	a := testApp(t)
	a.textEntry.SetText("Text to print")
	if a.sourceImg == nil {
		t.Fatal("missing text preview")
	}
	a.tabs.SelectIndex(0)
	if a.sourceImg != nil {
		t.Fatal("image tab retains text to print")
	}
	a.tabs.SelectIndex(1)
	a.textEntry.SetText("")
	if a.sourceImg != nil {
		t.Fatal("clearing text leaves stale print image")
	}
}

// A blocked serial open can complete after Disconnect. It must be closed and
// must never become the app's active printer.
type fakePrinter struct {
	closed     chan struct{}
	queried    chan struct{}
	battery    int
	batteryErr error
}

func (p *fakePrinter) Alive() bool { return true }
func (p *fakePrinter) GetBattery() (int, error) {
	if p.queried != nil {
		select {
		case p.queried <- struct{}{}:
		default:
		}
	}
	return p.battery, p.batteryErr
}
func (p *fakePrinter) Print([]byte) error { return nil }
func (p *fakePrinter) Close() error       { close(p.closed); return nil }

func TestDisconnectCancelsInFlightOpen(t *testing.T) {
	a := testApp(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	p := &fakePrinter{closed: make(chan struct{})}
	a.dialPrinter = func(string) (printerConnection, error) { close(entered); <-release; return p, nil }
	a.startConnection("COM12", "P21", "port")
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("open did not start")
	}
	a.disconnect()
	close(release)
	select {
	case <-p.closed:
	case <-time.After(time.Second):
		t.Fatal("late connection was not closed")
	}
	if a.isConnected() {
		t.Fatal("late connection revived cancelled session")
	}
	s, err := settings.Load(a.fyneApp.Preferences())
	if err != nil {
		t.Fatal(err)
	}
	if s.LastPrinter != "COM12" || s.ConnectionMode != "port" {
		t.Fatal("printer not remembered")
	}
}

func TestLibraryTabAvailable(t *testing.T) {
	a := testApp(t)
	if len(a.library.Items) != 2 || !a.library.Items[0].Template {
		t.Fatal("starter templates missing")
	}
	a.tabs.SelectIndex(3)
	if a.tabs.SelectedIndex() != 3 {
		t.Fatal("library tab inaccessible")
	}
	// Ensure its layout has enough structure to display the scrollable list.
	if _, ok := a.tabs.Items[3].Content.(*fyne.Container); !ok {
		t.Fatal("library unavailable")
	}
}

func TestReconnectAfterPrintFailureAndExplicitDisconnect(t *testing.T) {
	a := testApp(t)
	first := &fakePrinter{closed: make(chan struct{}), queried: make(chan struct{}, 1)}
	second := &fakePrinter{closed: make(chan struct{}), queried: make(chan struct{}, 1)}
	attempts := make(chan string, 3)
	count := 0
	a.dialPrinter = func(port string) (printerConnection, error) {
		attempts <- port
		count++
		if count == 1 {
			return first, nil
		}
		return second, nil
	}
	wait := func(ch <-chan struct{}) {
		t.Helper()
		select {
		case <-ch:
		case <-time.After(8 * time.Second):
			t.Fatal("connection event timed out")
		}
	}
	a.startConnection("COM12", "P21", "port")
	wait(first.queried)
	// Exercise rebuilding the visible UI while connection status is being read.
	a.changeLanguage("en")
	a.connectionFailed(first)
	wait(first.closed)
	wait(second.queried)
	if !a.isConnected() {
		t.Fatal("printer did not reconnect")
	}
	a.disconnect()
	wait(second.closed)
	if a.isConnected() {
		t.Fatal("Disconnect left an active printer")
	}
	if len(attempts) != 2 {
		t.Fatalf("unexpected opens: %d", len(attempts))
	}
	// Cancelled sessions cannot retry, including a pending retry timer.
	a.connMu.Lock()
	active := a.connection != nil
	a.connMu.Unlock()
	if active {
		t.Fatal("Disconnect did not cancel retries")
	}
}

func TestBatteryUnavailableAndValid(t *testing.T) {
	a := testApp(t)
	p := &fakePrinter{closed: make(chan struct{}), battery: 73}
	s := &connectionSession{stop: make(chan struct{}), failed: make(chan struct{}, 1), p: p}
	a.connMu.Lock()
	a.connection = s
	a.connMu.Unlock()
	if !a.updateBattery(p) || a.batteryLabel.Text != "Batteri: 73%" {
		t.Fatal("battery percentage unavailable")
	}
	p.battery = 255
	if a.updateBattery(p) || a.batteryLabel.Text != "Batteri: ukendt" {
		t.Fatal("invalid battery shown as a percentage")
	}
	p.batteryErr = errors.New("unsupported")
	if a.updateBattery(p) || a.batteryLabel.Text != "Batteri: ukendt" {
		t.Fatal("unsupported query shown as a percentage")
	}
	a.stopConnection()
}
