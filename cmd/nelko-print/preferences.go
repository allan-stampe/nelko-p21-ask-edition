package main

import (
	"fmt"

	"nelko-print/internal/imaging"
	"nelko-print/internal/label"
	"nelko-print/internal/printer"
	"nelko-print/internal/settings"
	"nelko-print/internal/tspl"
)

func (a *App) loadSettings() {
	a.commLog = &printer.CommunicationLog{}
	var err error
	a.prefs, err = settings.Load(a.fyneApp.Preferences())
	if err != nil {
		a.showError(err)
	}
	a.library, a.libraryErr = settings.LoadLibrary(a.fyneApp.Preferences())
	if a.libraryErr == nil && a.fyneApp.Preferences().String(settings.LibraryKey) == "" {
		a.library = starterLibrary()
		a.libraryErr = a.library.Save(a.fyneApp.Preferences())
	}
	a.fyneApp.Preferences().SetString("ask.language", a.prefs.Language)
	a.fyneApp.Preferences().SetBool("ask.autoReconnect", a.prefs.AutoReconnect)
	a.fyneApp.Preferences().SetBool("ask.pollBattery", a.prefs.PollBattery)
	a.applySettings(a.prefs)
}

func (a *App) applySettings(s settings.Settings) {
	s.Normalize()
	a.fontName, a.fontSize = s.Font, s.FontSize
	a.lastText, a.mode = s.Text, s.Mode
	a.density, a.threshold, a.copies = s.Density, s.Threshold, s.Copies
	a.orientation, a.invert = imaging.Orientation(s.Orientation), s.Invert
	a.textInvert, a.wordBreakOnly = s.TextInvert, s.WordBreakOnly
	for _, size := range tspl.AllSizes {
		if size.Name == s.LabelSize {
			a.labelSize = size
		}
	}
	a.designDoc = &label.Document{}
	if len(s.Design) > 0 {
		doc, err := label.UnmarshalDocument(s.Design)
		if err != nil {
			a.showError(fmt.Errorf("%s: %w", a.tr("Cannot restore design"), err))
		} else {
			a.designDoc = doc
		}
	}
}

func (a *App) snapshot() (settings.Settings, error) {
	s := a.prefs
	s.Font, s.FontSize = a.fontName, a.fontSize
	s.Text, s.Mode = a.lastText, a.mode
	s.LabelSize, s.Density, s.Threshold, s.Copies = a.labelSize.Name, a.density, a.threshold, a.copies
	s.Orientation, s.Invert = int(a.orientation), a.invert
	s.TextInvert, s.WordBreakOnly = a.textInvert, a.wordBreakOnly
	data, err := label.MarshalDocument(a.designDoc)
	s.Design = data
	return s, err
}

func (a *App) saveSettings() {
	if a.loading {
		return
	}
	a.fyneApp.Preferences().SetString("ask.language", a.prefs.Language)
	a.fyneApp.Preferences().SetBool("ask.autoReconnect", a.prefs.AutoReconnect)
	a.fyneApp.Preferences().SetBool("ask.pollBattery", a.prefs.PollBattery)
	s, err := a.snapshot()
	if err == nil {
		err = settings.Save(a.fyneApp.Preferences(), s)
	}
	if err != nil {
		a.showError(err)
	}
}

func (a *App) renderActivePreview() {
	if a.loading {
		return
	}
	switch a.mode {
	case "designer":
		a.renderDesignPreview()
	case "text":
		a.updateTextPreview()
	case "image":
		a.sourceImg = a.imageSource
		if a.sourceImg == nil {
			a.previewImg.Image = nil
			a.previewImg.Refresh()
		} else {
			a.updatePreview()
		}
	}
}

func (a *App) changeLanguage(language string) {
	a.prefs.Language = language
	a.saveSettings()
	a.window.SetMainMenu(a.buildMenu())
	a.window.SetContent(a.buildUI())
	a.renderActivePreview()
}
