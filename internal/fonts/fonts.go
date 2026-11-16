// Package fonts provides bundled fonts so saved labels render identically on every PC.
package fonts

import (
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

var Names = []string{"Go Regular", "Go Bold", "Go Mono"}

func Valid(name string) bool {
	for _, n := range Names {
		if n == name {
			return true
		}
	}
	return false
}

func Parse(name string) (*truetype.Font, error) {
	data := goregular.TTF
	switch name {
	case "Go Bold":
		data = gobold.TTF
	case "Go Mono":
		data = gomono.TTF
	}
	return truetype.Parse(data)
}
