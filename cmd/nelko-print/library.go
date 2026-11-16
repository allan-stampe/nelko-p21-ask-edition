package main

import (
	"crypto/rand"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"nelko-print/internal/label"
	"nelko-print/internal/settings"
)

func (a *App) buildLibraryTab() fyne.CanvasObject {
	if a.libraryErr != nil {
		return container.NewVBox(widget.NewLabel(a.tr("Cannot read library; stored data has been preserved")), widget.NewLabel(a.libraryErr.Error()))
	}
	selected := -1
	var list *widget.List
	list = widget.NewList(func() int { return len(a.library.Items) }, func() fyne.CanvasObject { return widget.NewLabel("") }, func(id widget.ListItemID, obj fyne.CanvasObject) {
		item := a.library.Items[id]
		name := item.Name
		if item.Template {
			name += " (" + a.tr("Template") + ")"
		}
		obj.(*widget.Label).SetText(name)
	})
	list.OnSelected = func(id widget.ListItemID) { selected = id }
	choose := func() *settings.SavedLabel {
		if selected < 0 || selected >= len(a.library.Items) {
			dialog.ShowInformation(a.tr("Library"), a.tr("Select a saved label first"), a.window)
			return nil
		}
		return &a.library.Items[selected]
	}
	persist := func() {
		if err := a.library.Save(a.fyneApp.Preferences()); err != nil {
			a.showError(err)
		}
		list.Refresh()
	}
	save := widget.NewButton(a.tr("Save current label"), func() {
		if a.mode == "image" {
			dialog.ShowInformation(a.tr("Library"), a.tr("Use Text or Designer to save a label"), a.window)
			return
		}
		name := widget.NewEntry()
		template := widget.NewCheck(a.tr("Template"), nil)
		dialog.ShowForm(a.tr("Save current label"), a.tr("Save"), a.tr("Cancel"), []*widget.FormItem{widget.NewFormItem(a.tr("Name"), name), widget.NewFormItem("", template)}, func(ok bool) {
			if !ok {
				return
			}
			snapshot, err := a.snapshot()
			if err != nil {
				a.showError(err)
				return
			}
			// Library entries carry print/layout settings, never connection identities.
			snapshot.LastPrinter = ""
			snapshot.LastPrinterName = ""
			id := make([]byte, 16)
			if _, err = rand.Read(id); err != nil {
				a.showError(err)
				return
			}
			err = a.library.Put(settings.SavedLabel{ID: fmt.Sprintf("%x", id), Name: name.Text, Template: template.Checked, Settings: snapshot})
			if err != nil {
				dialog.ShowInformation(a.tr("Library"), a.tr("Enter a name"), a.window)
				return
			}
			persist()
		}, a.window)
	})
	load := widget.NewButton(a.tr("Open for editing"), func() {
		item := choose()
		if item == nil {
			return
		}
		a.applySettings(item.Settings)
		a.saveSettings()
		a.window.SetContent(a.buildUI())
		a.renderActivePreview()
	})
	update := widget.NewButton(a.tr("Replace with current label"), func() {
		item := choose()
		if item == nil {
			return
		}
		if a.mode == "image" {
			dialog.ShowInformation(a.tr("Library"), a.tr("Use Text or Designer to save a label"), a.window)
			return
		}
		id, name, template := item.ID, item.Name, item.Template
		a.showConfirm(a.tr("Replace with current label"), name, func(ok bool) {
			if !ok {
				return
			}
			s, err := a.snapshot()
			if err != nil {
				a.showError(err)
				return
			}
			s.LastPrinter = ""
			s.LastPrinterName = ""
			if err = a.library.Put(settings.SavedLabel{ID: id, Name: name, Template: template, Settings: s}); err != nil {
				a.showError(err)
				return
			}
			persist()
		})
	})
	rename := widget.NewButton(a.tr("Rename"), func() {
		item := choose()
		if item == nil {
			return
		}
		copyItem := *item
		name := widget.NewEntry()
		name.SetText(item.Name)
		template := widget.NewCheck(a.tr("Template"), nil)
		template.SetChecked(item.Template)
		dialog.ShowForm(a.tr("Rename"), a.tr("Save"), a.tr("Cancel"), []*widget.FormItem{widget.NewFormItem(a.tr("Name"), name), widget.NewFormItem("", template)}, func(ok bool) {
			if !ok {
				return
			}
			copyItem.Name = strings.TrimSpace(name.Text)
			copyItem.Template = template.Checked
			if err := a.library.Put(copyItem); err != nil {
				dialog.ShowInformation(a.tr("Library"), a.tr("Enter a name"), a.window)
				return
			}
			persist()
		}, a.window)
	})
	deleteBtn := widget.NewButton(a.tr("Delete"), func() {
		item := choose()
		if item == nil {
			return
		}
		id, name := item.ID, item.Name
		a.showConfirm(a.tr("Delete saved label?"), name, func(ok bool) {
			if ok {
				a.library.Delete(id)
				selected = -1
				list.UnselectAll()
				persist()
			}
		})
	})
	return container.NewBorder(container.NewVBox(save, widget.NewLabel(a.tr("Open, edit, then replace the saved label"))), container.NewVBox(container.NewGridWithColumns(2, load, update), container.NewGridWithColumns(2, rename, deleteBtn)), nil, nil, list)
}

func starterLibrary() settings.Library {
	var l settings.Library
	for i, text := range []string{"Navn / Name", "KABEL / CABLE"} {
		s := settings.Default()
		s.Mode = "designer"
		el := label.NewText(text)
		el.FontSize = 12
		el.FontName = s.Font
		b := el.Bounds()
		b.X = 20
		b.Y = 12
		b.W = 240
		b.H = 70
		el.SetBounds(b)
		s.Design, _ = label.MarshalDocument(&label.Document{Elements: []label.Element{el}})
		l.Items = append(l.Items, settings.SavedLabel{ID: fmt.Sprintf("starter-%d", i), Name: text, Template: true, Settings: s})
	}
	return l
}
