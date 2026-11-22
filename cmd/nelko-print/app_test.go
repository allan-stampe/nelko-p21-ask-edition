package main

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"strings"
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
func (p *fakePrinter) PrintChunks([][]byte) error { return nil }
func (p *fakePrinter) Print([]byte) error         { return nil }
func (p *fakePrinter) CancelPause() error         { return nil }
func (p *fakePrinter) Close() error               { close(p.closed); return nil }

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
	a.prefs.PollBattery = true
	a.saveSettings()
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

func TestDiagnosticsLogExportAndLanguage(t *testing.T) {
	a := testApp(t)
	if len(a.tabs.Items) != 5 || a.tabs.Items[4].Text != "Diagnostik" {
		t.Fatal("diagnostics tab missing")
	}
	oldMode := a.mode
	a.tabs.SelectIndex(4)
	if a.mode != oldMode {
		t.Fatal("diagnostics changed active label")
	}
	a.commLog.Record("COM5", "TX", []byte{0x1b, 0x21, 0x6f}, "")
	a.commLog.Record("COM5", "RX", []byte("BATTERY"), "")
	a.uiMu.Lock()
	a.refreshDiagnosticLogLocked()
	a.uiMu.Unlock()
	if !a.diagLogEntry.Disabled() {
		t.Fatal("communication view can be edited")
	}
	text := a.communicationLogText()
	for _, want := range []string{"v1.0.2", "COM5", "TX 3 bytes", "RX 7 bytes", "ASCII:", "HEX:", "1b 21 6f"} {
		if !strings.Contains(text, want) {
			t.Fatalf("export missing %q", want)
		}
	}
	a.changeLanguage("en")
	if a.tabs.Items[4].Text != "Diagnostics" {
		t.Fatal("diagnostics not translated")
	}
	if a.diagLogEntry.Text == "" {
		t.Fatal("language switch lost log")
	}
	if a.prefs.PollBattery || a.fyneApp.Preferences().Bool("ask.pollBattery") {
		t.Fatal("unrequested battery polling enabled")
	}
}

func TestDiagnosticsMonitorAndActionSurviveUIRebuild(t *testing.T) {
	a := testApp(t)
	a.startDiagnosticsMonitor()
	a.commLog.Record("COM5", "TX", []byte("TEST"), "")
	waitFor := func(check func() bool) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			a.uiMu.Lock()
			ok := check()
			a.uiMu.Unlock()
			if ok {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatal("diagnostic UI event timed out")
	}
	waitFor(func() bool { return strings.Contains(a.diagLogEntry.Text, "TEST") })
	p := &fakePrinter{closed: make(chan struct{})}
	s := &connectionSession{stop: make(chan struct{}), failed: make(chan struct{}, 1), p: p}
	a.connMu.Lock()
	a.connection = s
	a.connMu.Unlock()
	entered := make(chan struct{})
	release := make(chan struct{})
	a.runDiagnostic("Cancel Pause", true, func(printerConnection) error { close(entered); <-release; return nil })
	<-entered
	a.changeLanguage("en")
	a.uiMu.Lock()
	busy := a.diagBusy && a.diagButtons[0].Disabled()
	a.uiMu.Unlock()
	if !busy {
		close(release)
		t.Fatal("in-flight action enabled duplicate diagnostic commands")
	}
	close(release)
	waitFor(func() bool { return !a.diagBusy })
	a.uiMu.Lock()
	completed := !a.diagButtons[0].Disabled() && strings.Contains(a.diagLogEntry.Text, "Cancel Pause completed")
	a.uiMu.Unlock()
	if !completed {
		t.Fatal("action result missing after UI rebuild")
	}
}

// A fresh app must use the saved COM target, including when enumeration is
// temporarily missing it. Reconnection is opt-in and must not select COM5.
func TestSavedCOMPortRestoredAcrossRestart(t *testing.T) {
	a := testApp(t)
	first := &fakePrinter{closed: make(chan struct{})}
	a.portSelect.Options = []string{"COM1", "COM4", "COM5"}
	a.portSelect.SetSelected("COM4")
	a.dialPrinter = func(port string) (printerConnection, error) {
		if port != "COM4" {
			return nil, fmt.Errorf("unexpected target %s", port)
		}
		return first, nil
	}
	a.connectManualPort()
	deadline := time.Now().Add(2 * time.Second)
	for !a.isConnected() && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !a.isConnected() {
		t.Fatal("first connection did not open")
	}
	a.disconnect()
	select {
	case <-first.closed:
	case <-time.After(2 * time.Second):
		t.Fatal("first connection not closed")
	}
	saved := a.fyneApp.Preferences().String(settings.SettingsKey)

	restarted := testApp(t)
	restarted.fyneApp.Preferences().SetString(settings.SettingsKey, saved)
	restarted.loadSettings()
	restarted.window.SetContent(restarted.buildUI())
	if restarted.prefs.LastPrinter != "COM4" || restarted.prefs.ConnectionMode != "port" || restarted.portSelect.Selected != "COM4" {
		t.Fatal("saved port was lost or replaced on restart")
	}
	restarted.prefs.AutoReconnect = false
	restarted.saveSettings()
	restarted.restoreConnection()
	restarted.connMu.Lock()
	active := restarted.connection != nil
	restarted.connMu.Unlock()
	if active {
		t.Fatal("auto-reconnect off still opened a session")
	}

	targets := make(chan string, 1)
	release := make(chan struct{})
	second := &fakePrinter{closed: make(chan struct{})}
	restarted.dialPrinter = func(port string) (printerConnection, error) { targets <- port; <-release; return second, nil }
	restarted.prefs.AutoReconnect = true
	restarted.saveSettings()
	restarted.restoreConnection()
	select {
	case target := <-targets:
		if target != "COM4" {
			t.Fatalf("restart opened %s instead of COM4", target)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("saved target was not reopened")
	}
	restarted.disconnect()
	close(release)
	select {
	case <-second.closed:
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled restart connection not closed")
	}
}
