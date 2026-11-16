package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (a *App) showError(err error) {
	message := widget.NewLabel(err.Error())
	message.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustom(a.tr("Error"), a.tr("Close"), message, a.window)
	d.Resize(fyne.NewSize(520, 200))
	d.Show()
}

func (a *App) showConfirm(title, message string, callback func(bool)) {
	d := dialog.NewConfirm(title, message, callback, a.window)
	d.SetConfirmText(a.tr("Yes"))
	d.SetDismissText(a.tr("No"))
	d.Show()
}

func (a *App) localizeFileDialog(d *dialog.FileDialog, save bool) {
	if save {
		d.SetConfirmText(a.tr("Save"))
	} else {
		d.SetConfirmText(a.tr("Open"))
	}
	d.SetDismissText(a.tr("Cancel"))
}
