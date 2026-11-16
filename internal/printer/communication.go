package printer

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
)

// CommunicationLog keeps complete I/O chunks within a bounded, session-only log.
// It records what the serial API actually accepted/returned, not intended writes.
type CommunicationLog struct {
	mu       sync.Mutex
	entries  []string
	bytes    int
	dropped  int
	revision uint64
}

const logCapacity = 256 * 1024

func (l *CommunicationLog) Record(port, direction string, data []byte, note string) {
	if l == nil {
		return
	}
	// A single large print is logged in full; old entries are dropped first.
	line := fmt.Sprintf("%s [%s] %s %d bytes %s\n", time.Now().Format(time.RFC3339Nano), port, direction, len(data), note)
	if len(data) > 0 {
		line += "ASCII: " + strconv.QuoteToASCII(string(data)) + "\nHEX:\n" + hex.Dump(data) + "\n"
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for len(l.entries) > 0 && l.bytes+len(line) > logCapacity {
		l.bytes -= len(l.entries[0])
		l.entries = l.entries[1:]
		l.dropped++
	}
	// Slice log bytes into normal serial-sized chunks in loggedPort; only INFO
	// messages can exceed the limit, so bound those rather than growing forever.
	if len(line) > logCapacity {
		line = line[:logCapacity] + "\n[entry truncated]\n"
	}
	l.entries = append(l.entries, line)
	l.bytes += len(line)
	l.revision++
}

func (l *CommunicationLog) Snapshot() (string, uint64) {
	if l == nil {
		return "", 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	var b strings.Builder
	if l.dropped > 0 {
		fmt.Fprintf(&b, "[older log entries discarded: %d]\n", l.dropped)
	}
	for _, entry := range l.entries {
		b.WriteString(entry)
		b.WriteByte('\n')
	}
	return b.String(), l.revision
}

// loggedPort delegates all serial configuration unchanged. Read logging uses
// the same reader as the request, so Diagnostics never competes for RX bytes.
type loggedPort struct {
	serial.Port
	log  *CommunicationLog
	name string
}

func (p *loggedPort) record(direction string, data []byte, err error) {
	note := ""
	if err != nil {
		note = err.Error()
	}
	// Preserve every byte of a large write without a single unbounded entry.
	const chunk = 4096
	if len(data) == 0 {
		p.log.Record(p.name, direction, nil, note)
		return
	}
	for len(data) > 0 {
		n := len(data)
		if n > chunk {
			n = chunk
		}
		p.log.Record(p.name, direction, data[:n], note)
		data = data[n:]
	}
}
func (p *loggedPort) Write(data []byte) (int, error) {
	n, err := p.Port.Write(data)
	if n >= 0 && n <= len(data) {
		p.record("TX", data[:n], err)
	} else {
		p.record("TX", nil, fmt.Errorf("invalid write count %d: %v", n, err))
	}
	return n, err
}
func (p *loggedPort) Read(data []byte) (int, error) {
	n, err := p.Port.Read(data)
	if n >= 0 && n <= len(data) {
		p.record("RX", data[:n], err)
	} else {
		p.record("RX", nil, fmt.Errorf("invalid read count %d: %v", n, err))
	}
	return n, err
}
func (p *loggedPort) Close() error {
	err := p.Port.Close()
	p.record("CLOSE", nil, err)
	return err
}

func (l *CommunicationLog) Revision() uint64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.revision
}
