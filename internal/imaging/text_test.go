package imaging

import (
	"bytes"
	"image/png"
	"testing"

	"nelko-print/internal/fonts"
)

func TestBundledFontsAffectPrintImage(t *testing.T) {
	rendered := map[string][]byte{}
	for _, name := range fonts.Names {
		img, err := RenderTextWithOptions("ASK ÆØÅ iMW", 400, 120, TextOptions{FontName: name, FontSize: 16})
		if err != nil {
			t.Fatal(err)
		}
		var b bytes.Buffer
		if err = png.Encode(&b, img); err != nil {
			t.Fatal(err)
		}
		for other, prior := range rendered {
			if bytes.Equal(prior, b.Bytes()) {
				t.Fatalf("%s rendered identically to %s", name, other)
			}
		}
		rendered[name] = b.Bytes()
	}
}
