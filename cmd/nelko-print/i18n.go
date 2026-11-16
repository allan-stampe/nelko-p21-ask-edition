package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var danish = map[string]string{
	"Error": "Fejl", "Yes": "Ja", "No": "Nej", "Open": "Åbn",
	"About": "Om programmet", "Help": "Hjælp", "Close": "Luk",
	"A label printing app for the Nelko P21 thermal printer.": "Etiketprogram til NELKO P21-termoprinteren.",
	"Based on reverse engineering work from:":                 "Baseret på protokolarbejdet fra:",
	"Built with Fyne and Go":                                  "Udviklet med Fyne og Go",
	"Select a printer":                                        "Vælg en printer", "Select a port": "Vælg en port",
	"Not connected": "Ikke tilsluttet", "Connected": "Tilsluttet", "Bluetooth Printer:": "Bluetooth-printer:",
	"Connect": "Tilslut", "Disconnect": "Afbryd", "Disconnected": "Afbrudt", "Connect Port": "Tilslut port",
	"Manual Port (if already connected):": "Manuel port (allerede parret):", "Advanced": "Avanceret",
	"Label Size": "Etiketstørrelse", "Density": "Sværtningsgrad", "Copies": "Antal kopier", "Print": "Udskriv",
	"Threshold": "Tærskelværdi", "Invert": "Invertér", "Load Image": "Åbn billede",
	"Enter label text...": "Skriv etikettens tekst...", "Horizontal": "Vandret", "Vertical": "Lodret",
	"Orientation": "Retning", "Font": "Skrifttype", "Font Size": "Skriftstørrelse", "Break on space only": "Ombryd ved mellemrum",
	"Image": "Billede", "Text": "Tekst", "Designer": "Designer", "Library": "Bibliotek", "Auto reconnect": "Tilslut automatisk igen",
	"Scanning for paired devices...": "Søger efter parrede enheder...", "BT scan failed: %v": "Bluetooth-søgning mislykkedes: %v",
	"Found %d paired device(s)": "Fandt %d parrede enheder", "Connecting to %s...": "Tilslutter %s...",
	"Connection failed: %v": "Tilslutning mislykkedes: %v", "Connected to %s": "Tilsluttet %s",
	"Printer disconnected": "Printerforbindelsen blev afbrudt", "Waiting to reconnect...": "Venter på gentilslutning...",
	"Battery: unknown": "Batteri: ukendt", "Battery: %d%%": "Batteri: %d%%",
	"Printing...": "Udskriver...", "Print error: %v": "Udskrivningsfejl: %v", "Print complete!": "Udskrivning færdig!",
	"not connected to printer": "printeren er ikke tilsluttet", "no image loaded": "ingen etiket at udskrive",
	"Select a printer first": "Vælg først en printer", "Select a port first": "Vælg først en port",
	"No privilege helper (pkexec or sudo)": "Ingen rettighedshjælper (pkexec eller sudo)",
	"Cannot restore design":                "Kan ikke gendanne design", "Select an element to edit": "Vælg et objekt for at redigere det",
	"+ Text": "+ Tekst", "+ Barcode": "+ Stregkode", "+ Shape": "+ Figur", "+ Image": "+ Billede",
	"Delete": "Slet", "Save": "Gem", "Load": "Åbn fil", "Export PNG": "Eksportér PNG", "Size": "Størrelse", "Type": "Type", "Data": "Indhold",
	"Shape": "Figur", "Line": "Linje", "Rectangle": "Rektangel", "FilledRect": "Udfyldt rektangel", "No properties": "Ingen egenskaber",
	"Template": "Skabelon", "Save current label": "Gem aktuel etiket", "Open for editing": "Åbn til redigering",
	"Replace with current label": "Erstat med aktuel etiket", "Rename": "Omdøb", "Cancel": "Annuller", "Name": "Navn",
	"Delete saved label?": "Slet gemt etiket?", "Enter a name": "Indtast et navn", "Select a saved label first": "Vælg først en gemt etiket",
	"Use Text or Designer to save a label":                "Brug Tekst eller Designer for at gemme en etiket",
	"Open, edit, then replace the saved label":            "Åbn, redigér og erstat derefter den gemte etiket",
	"Cannot read library; stored data has been preserved": "Kan ikke læse biblioteket; de gemte data er bevaret",
}

func (a *App) tr(text string) string {
	// Fyne preferences synchronize reads with writes, including worker reads.
	if a.fyneApp.Preferences().StringWithFallback("ask.language", "da") == "da" {
		if translated, ok := danish[text]; ok {
			return translated
		}
	}
	return text
}

// Translate presentation labels; values handed to existing callbacks retain
// their original identifiers, including orientation and shape selections.
func (a *App) translateUI(obj fyne.CanvasObject) {
	switch o := obj.(type) {
	case *fyne.Container:
		for _, child := range o.Objects {
			a.translateUI(child)
		}
	case *container.Split:
		a.translateUI(o.Leading)
		a.translateUI(o.Trailing)
	case *container.Scroll:
		a.translateUI(o.Content)
	case *container.AppTabs:
		for _, item := range o.Items {
			item.Text = a.tr(item.Text)
			a.translateUI(item.Content)
		}
		o.Refresh()
	case *widget.Label:
		o.SetText(a.tr(o.Text))
	case *widget.Button:
		o.SetText(a.tr(o.Text))
	case *widget.Check:
		o.Text = a.tr(o.Text)
		o.Refresh()
	case *widget.Entry:
		o.SetPlaceHolder(a.tr(o.PlaceHolder))
	case *widget.Form:
		for _, item := range o.Items {
			item.Text = a.tr(item.Text)
			a.translateUI(item.Widget)
		}
		o.Refresh()
	case *widget.Accordion:
		for _, item := range o.Items {
			item.Title = a.tr(item.Title)
			a.translateUI(item.Detail)
		}
		o.Refresh()
	case *widget.Select:
		reverse := map[string]string{}
		for i, v := range o.Options {
			translated := a.tr(v)
			o.Options[i] = translated
			reverse[translated] = v
		}
		o.Selected = a.tr(o.Selected)
		callback := o.OnChanged
		if callback != nil {
			o.OnChanged = func(value string) {
				if original, ok := reverse[value]; ok {
					value = original
				}
				callback(value)
			}
		}
		o.Refresh()
	}
}
