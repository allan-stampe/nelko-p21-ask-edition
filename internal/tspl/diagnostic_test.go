package tspl

import (
	"bytes"
	"testing"
)

func TestReferenceTestJobIsBinaryAndMatchesHardwareSequence(t *testing.T) {
	parts := ReferenceTestParts()
	if len(parts) != 9 {
		t.Fatal("wrong write boundaries")
	}
	expected := []string{"SIZE 14.0 mm,40.0 mm\r\n", "GAP 5.0 mm,0 mm\r\n", "DIRECTION 0,0\r\n", "DENSITY 15\r\n", "CLS\r\n", "BITMAP 0,0,12,284,1,"}
	for i, want := range expected {
		if string(parts[i]) != want {
			t.Fatalf("part %d: %q", i, parts[i])
		}
	}
	if len(parts[6]) != 3408 {
		t.Fatalf("bitmap: %d bytes", len(parts[6]))
	}
	if !bytes.Contains(parts[6], []byte{0xff}) || !bytes.Contains(parts[6], []byte{0}) {
		t.Fatal("test pattern contains no contrast")
	}
	if string(parts[7]) != "\r\n" || string(parts[8]) != "PRINT 1\r\n" {
		t.Fatal("missing bitmap terminator or print command")
	}
	job := ReferenceTestJob()
	header := bytes.Join(parts[:6], nil)
	if !bytes.Equal(job[len(header):len(header)+3408], parts[6]) {
		t.Fatal("extra newline or text encoding before bitmap")
	}
	if !bytes.Equal(bytes.Join(parts, nil), job) {
		t.Fatal("combined job differs from segmented writes")
	}
	// Normal prints use the same mode, exact line spelling and binary framing.
	normal := BuildPrintJob(Label14x40, 15, parts[6], 1)
	if !bytes.Equal(normal, job) {
		t.Fatal("normal print command bytes differ from reference for identical settings")
	}
}
