package main

import (
	"bytes"
	"image/png"
	"testing"

	"fyne.io/fyne/v2/app"
)

func TestEmbeddedApplicationAndWindowIcon(t *testing.T) {
	image, err := png.Decode(bytes.NewReader(appIcon.Content()))
	if err != nil {
		t.Fatal(err)
	}
	if image.Bounds().Dx() != 256 || image.Bounds().Dy() != 256 {
		t.Fatalf("runtime icon dimensions: %v", image.Bounds())
	}
	a := app.NewWithID("dk.ask.nelko-p21.icon-test")
	defer a.Quit()
	a.SetIcon(appIcon)
	w := a.NewWindow("Icon test")
	defer w.Close()
	w.SetIcon(appIcon)
	if a.Icon() != appIcon || w.Icon() != appIcon {
		t.Fatal("application/window icon differs from the embedded artwork")
	}
}
