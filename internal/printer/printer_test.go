package printer

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"go.bug.st/serial"
	"nelko-print/internal/tspl"
)

type testPort struct {
	serial.Port
	writes    [][]byte
	reads     [][]byte
	maxWrite  int
	zeroWrite bool
	failAt    int
	readError error
}

func (p *testPort) Write(data []byte) (int, error) {
	n := len(data)
	if p.maxWrite > 0 && n > p.maxWrite {
		n = p.maxWrite
	}
	if p.zeroWrite {
		n = 0
	}
	p.writes = append(p.writes, append([]byte(nil), data[:n]...))
	if p.failAt == len(p.writes) {
		return n, io.ErrClosedPipe
	}
	return n, nil
}
func (p *testPort) Read(data []byte) (int, error) {
	if len(p.reads) == 0 {
		return 0, p.readError
	}
	chunk := p.reads[0]
	n := copy(data, chunk)
	if n == len(chunk) {
		p.reads = p.reads[1:]
	} else {
		p.reads[0] = chunk[n:]
	}
	return n, nil
}
func (p *testPort) Close() error { return nil }

func TestReferenceSequenceWritesAndTiming(t *testing.T) {
	port := &testPort{}
	log := &CommunicationLog{}
	var events []string
	p := &Printer{port: &loggedPort{Port: port, name: "COM5", log: log}, portName: "COM5", log: log}
	p.sleep = func(d time.Duration) {
		events = append(events, d.String())
		if d == CancelPauseDelay && len(port.writes) != 1 {
			t.Fatal("job sent before cancel-pause delay")
		}
		if d == PrintSettleDelay && len(port.writes) != 10 {
			t.Fatal("settle delay before complete job")
		}
	}
	parts := tspl.ReferenceTestParts()
	if err := p.PrintChunks(parts); err != nil {
		t.Fatal(err)
	}
	expected := append([][]byte{{0x1b, 0x21, 0x6f}}, parts...)
	if !reflect.DeepEqual(port.writes, expected) {
		t.Fatal("reference writes differ from hardware sequence")
	}
	if !reflect.DeepEqual(events, []string{"500ms", "3s"}) {
		t.Fatalf("wrong delays: %v", events)
	}
	text, _ := log.Snapshot()
	if !strings.Contains(text, "[COM5] TX 3 bytes") || !strings.Contains(text, "WAIT 0 bytes 500ms") || !strings.Contains(text, "WAIT 0 bytes 3s") {
		t.Fatalf("missing timing or TX log: %.500s", text)
	}
}

func TestPrintHandlesPartialWritesWithoutLosingBytes(t *testing.T) {
	port := &testPort{maxWrite: 7}
	p := &Printer{port: port, sleep: func(time.Duration) {}}
	job := tspl.ReferenceTestJob()
	if err := p.Print(job); err != nil {
		t.Fatal(err)
	}
	got := bytes.Join(port.writes, nil)
	want := append([]byte{0x1b, 0x21, 0x6f}, job...)
	if !bytes.Equal(got, want) {
		t.Fatal("bytes lost or duplicated on short write")
	}
}

func TestCancelPauseFailureAbortsPrint(t *testing.T) {
	port := &testPort{failAt: 1}
	p := &Printer{port: port, sleep: func(time.Duration) { t.Fatal("wait after failed cancel pause") }}
	if err := p.Print(tspl.ReferenceTestJob()); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("missing cancel failure: %v", err)
	}
	if len(port.writes) != 1 {
		t.Fatal("job sent after cancel failure")
	}
}

func TestZeroLengthWriteFailsWithoutLooping(t *testing.T) {
	port := &testPort{zeroWrite: true}
	p := &Printer{port: port, sleep: func(time.Duration) { t.Fatal("wait after failed write") }}
	if err := p.Print([]byte("test")); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCancelPauseButtonUsesExactBytesAndWait(t *testing.T) {
	port := &testPort{}
	var delays []time.Duration
	p := &Printer{port: port, sleep: func(d time.Duration) { delays = append(delays, d) }}
	if err := p.CancelPause(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(port.writes, [][]byte{{0x1b, 0x21, 0x6f}}) || !reflect.DeepEqual(delays, []time.Duration{500 * time.Millisecond}) {
		t.Fatal("cancel pause altered bytes or delay")
	}
}

func TestBatteryBytesAreNotTrimmedOrLostOnPartialReads(t *testing.T) {
	for _, level := range []byte{0, 9, 10, 13, 32, 73, 100} {
		port := &testPort{reads: [][]byte{[]byte("BATTERY"), {level}, {'\r'}, {'\n'}}}
		log := &CommunicationLog{}
		p := &Printer{port: &loggedPort{Port: port, log: log, name: "COM5"}}
		got, err := p.GetBattery()
		if err != nil || got != int(level) {
			t.Fatalf("battery %d: got %d, %v", level, got, err)
		}
		if len(port.reads) != 0 {
			t.Fatalf("premature end of battery reply for level %d", level)
		}
		if string(port.writes[0]) != "BATTERY?\r\n" {
			t.Fatal("battery command changed")
		}
		text, _ := log.Snapshot()
		if !strings.Contains(text, " RX ") {
			t.Fatal("RX not logged")
		}
	}
}

func TestBatteryWithoutNewlineAndTimeout(t *testing.T) {
	port := &testPort{reads: [][]byte{append([]byte("BATTERY"), byte(42))}}
	p := &Printer{port: port}
	got, err := p.GetBattery()
	if err != nil || got != 42 {
		t.Fatalf("partial timeout reply lost: %d %v", got, err)
	}
	if _, err = p.GetBattery(); !errors.Is(err, ErrTimeout) {
		t.Fatalf("empty timeout accepted: %v", err)
	}
	for _, reply := range [][]byte{[]byte("garbage\r\n"), append([]byte("BATTERY"), byte(255))} {
		port.reads = [][]byte{reply}
		if _, err = p.GetBattery(); err == nil {
			t.Fatal("invalid response accepted")
		}
	}
}

func TestPrintQuietPeriodsExcludeBatteryRequests(t *testing.T) {
	port := &testPort{reads: [][]byte{append([]byte("BATTERY"), 73, '\r', '\n')}}
	inSettle := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	p := &Printer{port: port, sleep: func(d time.Duration) {
		if d == PrintSettleDelay {
			close(inSettle)
			<-release
		}
	}}
	go func() { done <- p.Print(tspl.ReferenceTestJob()) }()
	<-inSettle
	batteryDone := make(chan error, 1)
	started := make(chan struct{})
	go func() { close(started); _, err := p.GetBattery(); batteryDone <- err }()
	<-started
	// The job and cancel-pause are the only writes before the lock is released.
	if len(port.writes) != 2 {
		t.Fatal("battery interleaved with print sequence")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := <-batteryDone; err != nil {
		t.Fatal(err)
	}
	if string(port.writes[len(port.writes)-1]) != "BATTERY?\r\n" {
		t.Fatal("battery query not serialized after settle delay")
	}
}

func TestCommunicationLogRecordsActualBytesAndErrors(t *testing.T) {
	raw := &testPort{maxWrite: 2, failAt: 1, reads: [][]byte{{0, 13, 10, 255, 27}}}
	log := &CommunicationLog{}
	port := &loggedPort{Port: raw, name: "COM5", log: log}
	if n, err := port.Write([]byte("ABCD")); n != 2 || err == nil {
		t.Fatal("bad test write")
	}
	buf := make([]byte, 32)
	port.Read(buf)
	text, revision := log.Snapshot()
	if revision != 2 || !strings.Contains(text, "TX 2 bytes") || strings.Contains(text, "ABCD") || !strings.Contains(text, `ASCII: "AB"`) || !strings.Contains(text, "00 0d 0a ff 1b") {
		t.Fatalf("inaccurate log: %s", text)
	}
	first := strings.Fields(text)[0]
	if _, err := time.Parse(time.RFC3339Nano, first); err != nil {
		t.Fatal("missing timestamp", err)
	}
}

func TestCommunicationLogIsBoundedAndConcurrent(t *testing.T) {
	log := &CommunicationLog{}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				log.Record("COM5", "TX", bytes.Repeat([]byte{0xff}, 1000), "")
				log.Snapshot()
			}
		}()
	}
	wg.Wait()
	text, revision := log.Snapshot()
	if revision != 400 || len(text) > logCapacity+100 || !strings.Contains(text, "older log entries discarded") {
		t.Fatalf("unbounded or incomplete log: %d bytes, revision %d", len(text), revision)
	}
}
