package main

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2/dialog"
	"nelko-print/internal/printer"
)

// One session owns all opens, health checks and closes. Cancelling prevents a
// late open from reviving a connection after Disconnect or window close.
type printerConnection interface {
	Alive() bool
	GetBattery() (int, error)
	Print([]byte) error
	PrintChunks([][]byte) error
	CancelPause() error
	Close() error
}

type connectionSession struct {
	stop   chan struct{}
	failed chan struct{}
	once   sync.Once
	p      printerConnection
}

func (a *App) currentPrinter() printerConnection {
	a.connMu.Lock()
	defer a.connMu.Unlock()
	if a.connection == nil {
		return nil
	}
	return a.connection.p
}
func (a *App) isConnected() bool { return a.currentPrinter() != nil }

func (a *App) stopConnection() {
	a.connMu.Lock()
	session := a.connection
	a.connection = nil
	a.connMu.Unlock()
	if session != nil {
		session.once.Do(func() { close(session.stop) })
	}
}

func (a *App) disconnect() {
	a.stopConnection()
	a.resetConnectionUI()
}

func (a *App) resetConnectionUI() {
	a.uiMu.Lock()
	defer a.uiMu.Unlock()
	a.connMu.Lock()
	active := a.connection != nil
	a.connMu.Unlock()
	if active {
		return
	}
	a.connectBtn.SetText(a.tr("Connect"))
	a.connectBtn.Enable()
	a.btDeviceSelect.Enable()
	a.refreshBTBtn.Enable()
	a.printBtn.Disable()
	a.statusLabel.SetText(a.tr("Disconnected"))
	a.batteryLabel.SetText(a.tr("Battery: unknown"))
}

func (a *App) restoreConnection() {
	if a.prefs.AutoReconnect && a.prefs.LastPrinter != "" {
		a.startConnection(a.prefs.LastPrinter, a.prefs.LastPrinterName, a.prefs.ConnectionMode)
	}
}

func (a *App) connectBluetooth() {
	a.connMu.Lock()
	active := a.connection != nil
	a.connMu.Unlock()
	if active {
		a.disconnect()
		return
	}
	device := a.getSelectedBluetoothDevice()
	if device == nil {
		dialog.ShowInformation(a.tr("Connect"), a.tr("Select a printer first"), a.window)
		return
	}
	if err := printer.CheckRFCOMMInstalled(); err != nil {
		a.showError(err)
		return
	}
	if printer.CheckPrivilegeHelper() == "" {
		a.showError(fmt.Errorf("%s", a.tr("No privilege helper (pkexec or sudo)")))
		return
	}
	a.startConnection(device.MAC, device.Name, "bluetooth")
}

func (a *App) connectManualPort() {
	a.connMu.Lock()
	active := a.connection != nil
	a.connMu.Unlock()
	if active {
		a.disconnect()
		return
	}
	port := a.portSelect.Selected
	if port == "" {
		dialog.ShowInformation(a.tr("Connect"), a.tr("Select a port first"), a.window)
		return
	}
	a.startConnection(port, port, "port")
}

func (a *App) startConnection(target, name, mode string) {
	a.stopConnection()
	a.connMu.Lock()
	if a.closed {
		a.connMu.Unlock()
		return
	}
	session := &connectionSession{stop: make(chan struct{}), failed: make(chan struct{}, 1)}
	a.connection = session
	a.connMu.Unlock()
	a.prefs.LastPrinter, a.prefs.LastPrinterName, a.prefs.ConnectionMode = target, name, mode
	a.saveSettings()
	a.btDeviceSelect.Disable()
	a.refreshBTBtn.Disable()
	a.connectBtn.SetText(a.tr("Disconnect"))
	a.statusLabel.SetText(fmt.Sprintf(a.tr("Connecting to %s..."), name))
	go a.runConnection(session, target, name, mode)
}

func (a *App) sessionCurrent(s *connectionSession) bool {
	a.connMu.Lock()
	defer a.connMu.Unlock()
	return a.connection == s && !a.closed
}

func (a *App) connectionFailed(p printerConnection) {
	a.connMu.Lock()
	defer a.connMu.Unlock()
	if s := a.connection; s != nil && s.p == p {
		select {
		case s.failed <- struct{}{}:
		default:
		}
	}
}

func (a *App) runConnection(s *connectionSession, target, name, mode string) {
	var p printerConnection
	var conn *printer.RFCOMMConnection
	closePort := func() {
		a.connMu.Lock()
		s.p = nil
		a.connMu.Unlock()
		if p != nil {
			p.Close()
			p = nil
		}
		if conn != nil {
			conn.Close()
			conn = nil
		}
	}
	defer closePort()
	for {
		if !a.sessionCurrent(s) {
			return
		}
		var err error
		a.commLog.Record(target, "CONNECT", nil, "mode="+mode+" name="+name)
		path := target
		if mode == "bluetooth" {
			// Keep the existing platform Bluetooth implementation unchanged.
			conn, err = printer.EstablishRFCOMM(target, 1, nil)
			if err == nil {
				path = conn.DevicePath
			}
		}
		if err == nil {
			if a.dialPrinter != nil {
				p, err = a.dialPrinter(path)
			} else {
				p, err = printer.ConnectWithLog(path, a.commLog)
			}
		}
		if !a.sessionCurrent(s) {
			return
		}
		if err != nil {
			a.commLog.Record(target, "ERROR", nil, err.Error())
			closePort()
			a.withSessionUI(s, func() { a.statusLabel.SetText(fmt.Sprintf(a.tr("Connection failed: %v"), err)) })
			if !a.retryConnection(s) {
				return
			}
			continue
		}
		a.connMu.Lock()
		s.p = p
		a.connMu.Unlock()
		a.withSessionUI(s, func() { a.statusLabel.SetText(fmt.Sprintf(a.tr("Connected to %s"), name)); a.printBtn.Enable() })
		batterySupported := false
		if a.fyneApp.Preferences().Bool("ask.pollBattery") {
			batterySupported = a.updateBattery(p)
		}
		missed := 0
		ticker := time.NewTicker(5 * time.Second)
		lost := false
		for !lost {
			select {
			case <-s.stop:
				ticker.Stop()
				return
			case <-s.failed:
				lost = true
			case <-ticker.C:
				if !p.Alive() {
					lost = true
				} else if a.fyneApp.Preferences().Bool("ask.pollBattery") {
					ok := a.updateBattery(p)
					if ok {
						batterySupported = true
						missed = 0
					} else if batterySupported {
						missed++
						if missed >= 3 {
							lost = true
						}
					}
				}
			}
		}
		ticker.Stop()
		if !a.sessionCurrent(s) {
			return
		}
		a.withSessionUI(s, func() {
			a.printBtn.Disable()
			a.batteryLabel.SetText(a.tr("Battery: unknown"))
			a.statusLabel.SetText(a.tr("Printer disconnected"))
		})
		closePort()
		if !a.retryConnection(s) {
			return
		}
	}
}

func (a *App) retryConnection(s *connectionSession) bool {
	if !a.sessionCurrent(s) {
		return false
	}
	// Preferences' Bool is safe for background reads; avoid racing UI state.
	if !a.fyneApp.Preferences().BoolWithFallback("ask.autoReconnect", true) {
		a.connMu.Lock()
		if a.connection == s {
			a.connection = nil
			s.once.Do(func() { close(s.stop) })
		}
		a.connMu.Unlock()
		a.resetConnectionUI()
		return false
	}
	a.withSessionUI(s, func() { a.statusLabel.SetText(a.tr("Waiting to reconnect...")) })
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case <-s.stop:
		return false
	case <-timer.C:
		return a.sessionCurrent(s)
	}
}

func (a *App) updateBattery(p printerConnection) bool {
	level, err := p.GetBattery()
	a.uiMu.Lock()
	defer a.uiMu.Unlock()
	if a.currentPrinter() != p {
		return false
	}
	if err != nil || level < 0 || level > 100 {
		a.batteryLabel.SetText(a.tr("Battery: unknown"))
		return false
	}
	a.batteryLabel.SetText(fmt.Sprintf(a.tr("Battery: %d%%"), level))
	return true
}

// Widget references can change on language/library reloads. Serialize those
// replacements with worker updates and suppress updates from cancelled sessions.
func (a *App) withSessionUI(s *connectionSession, fn func()) {
	a.uiMu.Lock()
	defer a.uiMu.Unlock()
	if a.sessionCurrent(s) {
		fn()
	}
}
