package main

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

// The window/taskbar icon is compiled into the executable, not loaded from disk.
//
//go:embed assets/app-icon.png
var appIconPNG []byte

var appIcon = fyne.NewStaticResource("nelko-p21-ask-edition.png", appIconPNG)
