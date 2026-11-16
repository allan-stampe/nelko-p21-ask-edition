package tspl

import "bytes"

// ReferenceTestJob matches the hardware-tested PowerShell command spelling,
// order, mode and dimensions. Only its deterministic 96x284 test pattern is new.
// Its bits define a border and three bars; physical polarity is not inferred.
// No GUI threshold, invert, font, label size or copies setting affects this job.
func ReferenceTestParts() [][]byte {
	bitmap := bytes.Repeat([]byte{0xff}, 12*284)
	for y := 0; y < 284; y++ {
		for x := 0; x < 96; x++ {
			border := (x >= 4 && x < 92 && ((y >= 4 && y < 7) || (y >= 277 && y < 280))) || (y >= 4 && y < 280 && ((x >= 4 && x < 7) || (x >= 89 && x < 92)))
			bar := x >= 20 && x < 76 && ((y >= 60 && y < 66) || (y >= 120 && y < 126) || (y >= 180 && y < 186))
			if border || bar {
				bitmap[y*12+x/8] &= ^byte(1 << uint(7-x%8))
			}
		}
	}
	return [][]byte{
		[]byte("SIZE 14.0 mm,40.0 mm\r\n"),
		[]byte("GAP 5.0 mm,0 mm\r\n"),
		[]byte("DIRECTION 0,0\r\n"),
		[]byte("DENSITY 15\r\n"),
		[]byte("CLS\r\n"),
		[]byte("BITMAP 0,0,12,284,1,"),
		bitmap,
		[]byte("\r\n"),
		[]byte("PRINT 1\r\n"),
	}
}

func ReferenceTestJob() []byte { return bytes.Join(ReferenceTestParts(), nil) }
