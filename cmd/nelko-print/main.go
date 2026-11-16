package main

import (
	"fmt"
	"image"
	"net/url"
	"os"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nelko-print/internal/fonts"
	"nelko-print/internal/imaging"
	"nelko-print/internal/label"
	"nelko-print/internal/printer"
	"nelko-print/internal/settings"
	"nelko-print/internal/tspl"
)

const (
	AppVersion = "1.0.1"
	AppName    = "NELKO P21 – ASK Edition"
)

type App struct {
	commLog      *printer.CommunicationLog
	diagLogEntry *widget.Entry
	diagStatus   *widget.Label
	diagButtons  []*widget.Button
	diagBusy     bool
	diagRevision uint64
	diagStop     chan struct{}
	diagDone     chan struct{}
	prefs        settings.Settings
	library      settings.Library
	libraryErr   error
	loading      bool
	fontName     string
	lastText     string
	mode         string
	tabs         *container.AppTabs
	imageSource  image.Image
	dialPrinter  func(string) (printerConnection, error)
	connMu       sync.Mutex
	uiMu         sync.Mutex
	connection   *connectionSession
	closed       bool
	batteryLabel *widget.Label

	fyneApp    fyne.App
	window     fyne.Window
	sourceImg  image.Image
	previewImg *canvas.Image

	// Settings
	labelSize tspl.LabelSize
	density   int
	threshold uint8
	copies    int
	invert    bool

	// Widgets that need updating
	statusLabel    *widget.Label
	connectBtn     *widget.Button
	printBtn       *widget.Button
	btDeviceSelect *widget.Select
	portSelect     *widget.Select
	refreshBTBtn   *widget.Button

	// Bluetooth devices cache
	btDevices []printer.BluetoothDevice

	// Text mode
	textEntry     *widget.Entry
	orientation   imaging.Orientation
	fontSize      float64
	textInvert    bool
	wordBreakOnly bool

	// Designer mode
	designDoc    *label.Document
	designCanvas *labelCanvas
	designProps  *fyne.Container
}

func main() {
	a := app.NewWithID("dk.ask.nelko-p21")
	w := a.NewWindow(fmt.Sprintf("%s v%s", AppName, AppVersion))
	w.Resize(fyne.NewSize(1060, 820))

	nelkoApp := &App{fyneApp: a, window: w}

	nelkoApp.loadSettings()
	// Set up menu
	w.SetMainMenu(nelkoApp.buildMenu())
	w.SetContent(nelkoApp.buildUI())
	nelkoApp.renderActivePreview()
	nelkoApp.startDiagnosticsMonitor()
	nelkoApp.restoreConnection()
	w.SetOnClosed(func() {
		nelkoApp.cleanup()
	})
	w.ShowAndRun()
}

func (a *App) buildMenu() *fyne.MainMenu {
	// Help menu with About
	aboutItem := fyne.NewMenuItem(a.tr("About"), func() {
		a.showAboutDialog()
	})

	languageMenu := fyne.NewMenu("Language / Sprog",
		fyne.NewMenuItem("Dansk", func() { a.changeLanguage("da") }),
		fyne.NewMenuItem("English", func() { a.changeLanguage("en") }),
	)
	helpMenu := fyne.NewMenu(a.tr("Help"), aboutItem)

	return fyne.NewMainMenu(languageMenu, helpMenu)
}

func (a *App) showAboutDialog() {
	content := container.NewVBox(
		widget.NewLabelWithStyle(AppName, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabel(fmt.Sprintf(a.tr("Version %s"), AppVersion)),
		widget.NewSeparator(),
		widget.NewLabel(a.tr("A label printing app for the Nelko P21 thermal printer.")),
		widget.NewLabel(""),
		widget.NewLabel(a.tr("Based on reverse engineering work from:")),
		widget.NewHyperlink("merlinschumacher/nelko-p21-print", parseURL("https://github.com/merlinschumacher/nelko-p21-print")),
		widget.NewLabel(""),
		widget.NewLabel("Fyne-P21-Print — Copyright (c) 2026 Tyler C — MIT"),
		widget.NewLabel(a.tr("Built with Fyne and Go")),
	)

	dialog.ShowCustom(a.tr("About"), a.tr("Close"), content, a.window)
}

func parseURL(urlStr string) *url.URL {
	u, _ := url.Parse(urlStr)
	return u
}

func (a *App) cleanup() {
	if a.diagStop != nil {
		close(a.diagStop)
		<-a.diagDone
		a.diagStop = nil
	}
	a.saveSettings()
	a.connMu.Lock()
	a.closed = true
	a.connMu.Unlock()
	a.stopConnection()
}

func (a *App) buildUI() fyne.CanvasObject {
	a.uiMu.Lock()
	defer a.uiMu.Unlock()
	a.loading = true
	defer func() { a.loading = false }()
	// Create preview before selectors can trigger callbacks.
	a.previewImg = canvas.NewImageFromImage(nil)
	a.previewImg.SetMinSize(fyne.NewSize(150, 160))
	a.previewImg.FillMode = canvas.ImageFillContain

	// Status bar
	a.statusLabel = widget.NewLabel(a.tr("Not connected"))
	a.statusLabel.Wrapping = fyne.TextWrapWord

	// === BLUETOOTH CONNECTION SECTION ===
	btLabel := widget.NewLabel(a.tr("Bluetooth Printer:"))
	a.btDeviceSelect = widget.NewSelect([]string{}, func(s string) {})
	a.btDeviceSelect.PlaceHolder = a.tr("Select a printer")
	a.refreshBTBtn = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		a.refreshBluetoothDevices()
	})

	a.connectBtn = widget.NewButton("Connect", func() {
		a.connectBluetooth()
	})

	// Refresh BT devices on startup
	a.refreshBluetoothDevices()

	btRow := container.NewBorder(
		nil, nil, nil,
		container.NewHBox(a.refreshBTBtn, a.connectBtn),
		a.btDeviceSelect,
	)

	// === MANUAL PORT SECTION (collapsible/advanced) ===
	a.portSelect = widget.NewSelect([]string{}, func(s string) {})
	a.portSelect.PlaceHolder = a.tr("Select a port")
	a.refreshPorts()

	manualConnectBtn := widget.NewButton("Connect Port", func() {
		a.connectManualPort()
	})

	manualRefreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		a.refreshPorts()
	})

	manualRow := container.NewBorder(
		nil, nil, nil,
		container.NewHBox(manualRefreshBtn, manualConnectBtn),
		a.portSelect,
	)

	// Advanced section (manual port)
	advancedContent := container.NewVBox(
		widget.NewLabel(a.tr("Manual Port (if already connected):")),
		manualRow,
	)

	// Print settings
	sizeOptions := make([]string, len(tspl.AllSizes))
	for i, s := range tspl.AllSizes {
		sizeOptions[i] = s.Name
	}
	sizeSelect := widget.NewSelect(sizeOptions, func(s string) {
		for _, size := range tspl.AllSizes {
			if size.Name == s {
				a.labelSize = size
				a.saveSettings()
				if a.designCanvas != nil {
					a.designCanvas.SetSize(size)
				}
				a.renderActivePreview()
				break
			}
		}
	})
	sizeSelect.SetSelected(a.labelSize.Name)

	densitySlider := widget.NewSlider(0, 15)
	densitySlider.Value = float64(a.density)
	densitySlider.OnChanged = func(f float64) {
		a.density = int(f)
		a.saveSettings()
	}

	copiesEntry := widget.NewEntry()
	copiesEntry.SetText(fmt.Sprint(a.copies))
	copiesEntry.OnChanged = func(s string) {
		var n int
		fmt.Sscanf(s, "%d", &n)
		if n > 0 && n <= 999 {
			a.copies = n
			a.saveSettings()
		}
	}

	// Print button
	a.printBtn = widget.NewButton("Print", func() {
		a.print()
	})
	a.printBtn.Importance = widget.HighImportance
	a.printBtn.Disable()

	// === IMAGE TAB ===
	thresholdSlider := widget.NewSlider(0, 255)
	thresholdSlider.Value = float64(a.threshold)
	thresholdSlider.OnChanged = func(f float64) {
		a.threshold = uint8(f)
		a.saveSettings()
		a.updatePreview()
	}

	invertCheck := widget.NewCheck("Invert", func(b bool) {
		a.invert = b
		a.saveSettings()
		a.updatePreview()
	})

	invertCheck.SetChecked(a.invert)

	loadBtn := widget.NewButton("Load Image", func() {
		a.loadImage()
	})

	imageSettings := widget.NewForm(
		widget.NewFormItem("Threshold", thresholdSlider),
		widget.NewFormItem("", invertCheck),
	)

	imageTab := container.NewVBox(
		loadBtn,
		imageSettings,
	)

	// === TEXT TAB ===
	a.textEntry = widget.NewMultiLineEntry()
	a.textEntry.SetPlaceHolder("Enter label text...")
	a.textEntry.SetMinRowsVisible(3)
	a.textEntry.SetText(a.lastText)
	a.textEntry.OnChanged = func(s string) {
		a.lastText = s
		a.saveSettings()
		a.updateTextPreview()
	}

	orientationSelect := widget.NewSelect([]string{"Horizontal", "Vertical"}, func(s string) {
		if s == "Vertical" {
			a.orientation = imaging.Vertical
		} else {
			a.orientation = imaging.Horizontal
		}
		a.saveSettings()
		a.updateTextPreview()
	})
	if a.orientation == imaging.Vertical {
		orientationSelect.SetSelected("Vertical")
	} else {
		orientationSelect.SetSelected("Horizontal")
	}

	fontSizeSlider := widget.NewSlider(4, 72) // Reduced min from 8 to 4
	fontSizeSlider.Value = a.fontSize
	fontSizeSlider.OnChanged = func(f float64) {
		a.fontSize = f
		a.saveSettings()
		a.updateTextPreview()
	}

	textInvertCheck := widget.NewCheck("Invert", func(b bool) {
		a.textInvert = b
		a.saveSettings()
		a.updateTextPreview()
	})

	wordBreakCheck := widget.NewCheck("Break on space only", func(b bool) {
		a.wordBreakOnly = b
		a.saveSettings()
		a.updateTextPreview()
	})

	textInvertCheck.SetChecked(a.textInvert)
	wordBreakCheck.SetChecked(a.wordBreakOnly)
	fontSelect := widget.NewSelect(fonts.Names, func(s string) { a.fontName = s; a.saveSettings(); a.updateTextPreview() })
	fontSelect.SetSelected(a.fontName)
	textSettings := widget.NewForm(
		widget.NewFormItem("Font", fontSelect),
		widget.NewFormItem("Orientation", orientationSelect),
		widget.NewFormItem("Font Size", fontSizeSlider),
		widget.NewFormItem("", textInvertCheck),
		widget.NewFormItem("", wordBreakCheck),
	)

	textTab := container.NewVBox(
		a.textEntry,
		textSettings,
	)

	// === TABS ===
	tabs := container.NewAppTabs(
		container.NewTabItem("Image", imageTab),
		container.NewTabItem("Text", textTab),
		container.NewTabItem("Designer", a.buildDesignerTab()),
		container.NewTabItem("Library", a.buildLibraryTab()),
		container.NewTabItem("Diagnostics", a.buildDiagnosticsTab()),
	)
	a.tabs = tabs
	switch a.mode {
	case "image":
		tabs.SelectIndex(0)
	case "designer":
		tabs.SelectIndex(2)
	default:
		tabs.SelectIndex(1)
	}
	tabs.OnSelected = func(ti *container.TabItem) {
		switch tabs.SelectedIndex() {
		case 0:
			a.mode = "image"
		case 1:
			a.mode = "text"
		case 2:
			a.mode = "designer"
		default:
			return
		}
		a.saveSettings()
		a.renderActivePreview()
	}

	// Left panel - Connection and Settings
	autoReconnect := widget.NewCheck("Auto reconnect", func(b bool) {
		if a.loading {
			return
		}
		a.prefs.AutoReconnect = b
		a.saveSettings()
		if !b && !a.isConnected() {
			a.disconnect()
		}
	})
	autoReconnect.SetChecked(a.prefs.AutoReconnect)
	leftPanel := container.NewVBox(
		autoReconnect,
		btLabel,
		btRow,
		widget.NewSeparator(),
		widget.NewAccordion(
			widget.NewAccordionItem("Advanced", advancedContent),
		),
		widget.NewSeparator(),
		widget.NewLabel(a.tr("Label Size")),
		sizeSelect,
		widget.NewLabel(a.tr("Density")),
		densitySlider,
		widget.NewLabel(a.tr("Copies")),
		copiesEntry,
		widget.NewSeparator(),
		a.printBtn,
	)

	// Right panel: draggable split between the editing tabs (top) and the
	// shared mono preview (bottom). Default gives the tabs/canvas most of the
	// height; drag the divider to resize either side.
	rightPanel := container.NewVSplit(tabs, container.NewCenter(a.previewImg))
	rightPanel.SetOffset(0.66)

	content := container.NewHSplit(leftPanel, rightPanel)
	content.SetOffset(0.38)

	a.batteryLabel = widget.NewLabel(a.tr("Battery: unknown"))
	a.connMu.Lock()
	active := a.connection != nil
	a.connMu.Unlock()
	if active {
		a.connectBtn.SetText(a.tr("Disconnect"))
		a.btDeviceSelect.Disable()
		a.refreshBTBtn.Disable()
		if a.isConnected() {
			a.printBtn.Enable()
			a.statusLabel.SetText(a.tr("Connected"))
		} else {
			a.statusLabel.SetText(a.tr("Waiting to reconnect..."))
		}
	}
	a.translateUI(content)
	return container.NewBorder(
		nil,
		container.NewVBox(a.statusLabel, container.NewHBox(widget.NewLabel("v"+AppVersion), a.batteryLabel)),
		nil, nil,
		content,
	)
}

func (a *App) refreshBluetoothDevices() {
	a.statusLabel.SetText(a.tr("Scanning for paired devices..."))

	devices, err := printer.ListPairedBluetoothDevices()
	if err != nil {
		a.statusLabel.SetText(fmt.Sprintf(a.tr("BT scan failed: %v"), err))
		return
	}

	a.btDevices = devices

	// Build display list
	options := make([]string, len(devices))
	for i, d := range devices {
		options[i] = fmt.Sprintf("%s (%s)", d.Name, d.MAC)
	}

	a.btDeviceSelect.Options = options
	if len(options) > 0 {
		// Try to auto-select a Nelko device if present
		selectedIdx := 0
		for i, d := range devices {
			if d.MAC == a.prefs.LastPrinter {
				selectedIdx = i
				break
			}
		}
		for i, d := range devices {
			if a.prefs.LastPrinter == "" && (strings.Contains(strings.ToLower(d.Name), "nelko") ||
				strings.Contains(strings.ToLower(d.Name), "p21")) {
				selectedIdx = i
				break
			}
		}
		a.btDeviceSelect.SetSelected(options[selectedIdx])
	}

	a.statusLabel.SetText(fmt.Sprintf(a.tr("Found %d paired device(s)"), len(devices)))
}

func (a *App) refreshPorts() {
	// Look for /dev/rfcomm* devices (Linux)
	ports, _ := printer.FindRFCOMMDevices()

	// Add platform-detected serial ports (COM* on Windows, /dev/cu.* on macOS, etc.)
	if platPorts, err := printer.ListSerialPorts(); err == nil {
		for _, p := range platPorts {
			found := false
			for _, existing := range ports {
				if existing == p {
					found = true
					break
				}
			}
			if !found {
				ports = append(ports, p)
			}
		}
	}

	// Also add common serial ports
	commonPorts := []string{"/dev/rfcomm0", "/dev/rfcomm1", "/dev/ttyUSB0", "/dev/ttyACM0"}
	for _, p := range commonPorts {
		if _, err := os.Stat(p); err == nil {
			found := false
			for _, existing := range ports {
				if existing == p {
					found = true
					break
				}
			}
			if !found {
				ports = append(ports, p)
			}
		}
	}

	a.portSelect.Options = ports
	if len(ports) > 0 {
		a.portSelect.SetSelected(ports[0])
		for _, port := range ports {
			if port == a.prefs.LastPrinter {
				a.portSelect.SetSelected(port)
				break
			}
		}
	}
}

func (a *App) getSelectedBluetoothDevice() *printer.BluetoothDevice {
	selectedIdx := a.btDeviceSelect.SelectedIndex()
	if selectedIdx < 0 || selectedIdx >= len(a.btDevices) {
		return nil
	}
	return &a.btDevices[selectedIdx]
}

func (a *App) loadImage() {
	fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			a.showError(err)
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()

		img, _, err := image.Decode(reader)
		if err != nil {
			a.showError(err)
			return
		}

		a.imageSource = img
		a.sourceImg = img
		a.updatePreview()

		if a.isConnected() {
			a.printBtn.Enable()
		}
	}, a.window)

	a.localizeFileDialog(fd, false)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp"}))
	fd.Show()
}

func (a *App) updatePreview() {
	if a.loading || a.previewImg == nil || a.sourceImg == nil {
		return
	}

	// Convert to monochrome for preview
	mono := imaging.ToMonochrome(a.sourceImg, a.labelSize.PixelW, a.labelSize.PixelH, a.threshold, a.invert)
	preview := imaging.PreviewMonochrome(mono, a.labelSize.PixelW, a.labelSize.PixelH)

	// For vertical orientation, rotate the preview so text is readable on screen
	if a.orientation == imaging.Vertical {
		preview = imaging.RotatePreviewForDisplay(preview)
	}

	a.previewImg.Image = preview
	a.previewImg.Refresh()
}

func (a *App) updateTextPreview() {
	if a.loading || a.mode != "text" {
		return
	}
	text := a.textEntry.Text
	if text == "" {
		a.sourceImg = nil
		a.previewImg.Image = nil
		a.previewImg.Refresh()
		return
	}

	opts := imaging.TextOptions{
		FontName:      a.fontName,
		FontSize:      a.fontSize,
		Orientation:   a.orientation,
		Invert:        a.textInvert,
		WordBreakOnly: a.wordBreakOnly,
	}

	img, err := imaging.RenderTextWithOptions(text, a.labelSize.PixelW, a.labelSize.PixelH, opts)
	if err != nil {
		return
	}

	a.sourceImg = img
	a.updatePreview()

	if a.isConnected() {
		a.printBtn.Enable()
	}
}

func (a *App) print() {
	p := a.currentPrinter()
	if p == nil {
		a.showError(fmt.Errorf("%s", a.tr("not connected to printer")))
		return
	}

	if a.sourceImg == nil {
		a.showError(fmt.Errorf("%s", a.tr("no image loaded")))
		return
	}

	// Convert image to bitmap
	bitmap := imaging.ToMonochrome(a.sourceImg, a.labelSize.PixelW, a.labelSize.PixelH, a.threshold, !a.invert)

	// Build print job
	job := tspl.BuildPrintJob(a.labelSize, a.density, bitmap, a.copies)

	// Send to printer
	a.statusLabel.SetText(a.tr("Printing..."))
	a.printBtn.Disable()

	a.commLog.Record("app", "ACTION", nil, "Print started")
	go func() {
		err := p.Print(job)
		note := "Print data sent; settle delay complete"
		if err != nil {
			note = err.Error()
		}
		a.commLog.Record("app", "ACTION", nil, note)

		a.uiMu.Lock()
		defer a.uiMu.Unlock()
		if err != nil {
			a.statusLabel.SetText(fmt.Sprintf(a.tr("Print error: %v"), err))
			a.connectionFailed(p)
			a.showError(err)
		} else {
			a.statusLabel.SetText(a.tr("Print data sent; check printer"))
		}
		if a.isConnected() {
			a.printBtn.Enable()
		}
	}()
}
