package main

import (
	"fmt"
	"io"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"nelko-print/internal/tspl"
)

func (a *App) buildDiagnosticsTab() fyne.CanvasObject {
	a.diagLogEntry = widget.NewMultiLineEntry()
	a.diagLogEntry.TextStyle = fyne.TextStyle{Monospace: true}
	a.diagLogEntry.Wrapping = fyne.TextWrapOff
	a.diagLogEntry.Disable()
	a.diagStatus = widget.NewLabel(a.tr("Ready for diagnostics"))
	a.diagStatus.Wrapping = fyne.TextWrapWord
	battery := widget.NewButton("Test battery", func() {
		a.runDiagnostic("Test battery", false, func(p printerConnection) error {
			if !a.updateBattery(p) {
				return fmt.Errorf("%s", a.tr("No valid battery reply; see communication log"))
			}
			return nil
		})
	})
	cancel := widget.NewButton("Cancel Pause", func() {
		a.runDiagnostic("Cancel Pause", true, func(p printerConnection) error { return p.CancelPause() })
	})
	testPrint := widget.NewButton("Reference test print", func() {
		a.runDiagnostic("Reference test print", true, func(p printerConnection) error { return p.PrintChunks(tspl.ReferenceTestParts()) })
	})
	a.diagButtons = []*widget.Button{battery, cancel, testPrint}
	for _, button := range a.diagButtons {
		if a.diagBusy {
			button.Disable()
		}
	}
	poll := widget.NewCheck("Automatic battery polling", func(b bool) {
		if a.loading {
			return
		}
		a.prefs.PollBattery = b
		a.saveSettings()
	})
	poll.SetChecked(a.prefs.PollBattery)
	summary := widget.NewLabel(a.tr("Test label: 14x40 mm, density 15, one label. Uses the connected port."))
	summary.Wrapping = fyne.TextWrapWord
	save := widget.NewButton("Save communication log", func() { a.saveCommunicationLog() })
	refresh := widget.NewButton("Refresh log", func() { a.uiMu.Lock(); defer a.uiMu.Unlock(); a.refreshDiagnosticLogLocked() })
	a.refreshDiagnosticLogLocked()
	return container.NewBorder(container.NewVBox(summary, poll, container.NewGridWithColumns(2, battery, cancel), testPrint, a.diagStatus), container.NewGridWithColumns(2, refresh, save), nil, nil, a.diagLogEntry)
}

func (a *App) refreshDiagnosticLogLocked() {
	if a.diagLogEntry == nil {
		return
	}
	text, revision := a.commLog.Snapshot()
	a.diagLogEntry.SetText(text)
	a.diagRevision = revision
}

func (a *App) startDiagnosticsMonitor() {
	a.diagStop = make(chan struct{})
	a.diagDone = make(chan struct{})
	go func() {
		defer close(a.diagDone)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-a.diagStop:
				return
			case <-ticker.C:
				a.uiMu.Lock()
				if a.commLog.Revision() != a.diagRevision {
					a.refreshDiagnosticLogLocked()
				}
				a.uiMu.Unlock()
			}
		}
	}()
}

func (a *App) runDiagnostic(name string, reconnectOnError bool, action func(printerConnection) error) {
	p := a.currentPrinter()
	if p == nil {
		a.showError(fmt.Errorf("%s", a.tr("not connected to printer")))
		return
	}
	a.uiMu.Lock()
	if a.diagBusy {
		a.uiMu.Unlock()
		return
	}
	a.diagBusy = true
	for _, button := range a.diagButtons {
		button.Disable()
	}
	a.diagStatus.SetText(a.tr(name) + ": " + a.tr("Running..."))
	a.uiMu.Unlock()
	a.commLog.Record("app", "ACTION", nil, name+" started")
	go func() {
		err := action(p)
		note := name + " completed"
		if err != nil {
			note = name + ": " + err.Error()
		}
		a.commLog.Record("app", "ACTION", nil, note)
		a.uiMu.Lock()
		defer a.uiMu.Unlock()
		a.diagBusy = false
		for _, button := range a.diagButtons {
			button.Enable()
		}
		if err != nil {
			a.diagStatus.SetText(a.tr(name) + ": " + err.Error())
			if reconnectOnError {
				a.connectionFailed(p)
			}
		} else {
			a.diagStatus.SetText(a.tr(name) + ": " + a.tr("Command sent; check printer and log"))
		}
		a.refreshDiagnosticLogLocked()
	}()
}

func (a *App) communicationLogText() string {
	text, _ := a.commLog.Snapshot()
	return fmt.Sprintf("%s v%s\nTX/RX = bytes accepted/returned by serial API; not a physical printer acknowledgement.\nTimestamps: RFC3339 with timezone offset.\n\n%s", AppName, AppVersion, text)
}

func (a *App) saveCommunicationLog() {
	fd := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil {
			a.showError(err)
			return
		}
		if w == nil {
			return
		}
		text := a.communicationLogText()
		n, writeErr := io.WriteString(w, text)
		if writeErr == nil && n != len(text) {
			writeErr = io.ErrShortWrite
		}
		closeErr := w.Close()
		if writeErr != nil {
			a.showError(writeErr)
		} else if closeErr != nil {
			a.showError(closeErr)
		}
	}, a.window)
	a.localizeFileDialog(fd, true)
	fd.SetFileName("NELKO-P21-ASK-v" + AppVersion + "-" + time.Now().Format("20060102-150405") + ".log")
	fd.Show()
}
