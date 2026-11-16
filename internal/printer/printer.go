package printer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
)

var (
	ErrNotConnected = errors.New("printer not connected")
	ErrTimeout      = errors.New("operation timed out")
)

// Printer represents a connection to the Nelko P21
type Printer struct {
	port     serial.Port
	portName string
	mac      string
	mu       sync.Mutex // serializes all port I/O, including print delays
	log      *CommunicationLog
	sleep    func(time.Duration) // test seam; nil uses time.Sleep
}

// FindRFCOMMDevices lists available /dev/rfcomm* devices
func FindRFCOMMDevices() ([]string, error) {
	// Check for existing rfcomm devices
	out, err := exec.Command("ls", "/dev/rfcomm*").Output()
	if err != nil {
		return nil, nil // No devices found, not an error
	}

	devices := strings.Fields(string(out))
	return devices, nil
}

// ListPairedDevices returns paired Bluetooth devices (name, MAC)
func ListPairedDevices() (map[string]string, error) {
	// Use bluetoothctl to list paired devices
	out, err := exec.Command("bluetoothctl", "devices", "Paired").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}

	devices := make(map[string]string)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		// Format: "Device XX:XX:XX:XX:XX:XX DeviceName"
		parts := strings.SplitN(strings.TrimPrefix(line, "Device "), " ", 2)
		if len(parts) == 2 {
			mac := parts[0]
			name := parts[1]
			devices[name] = mac
		}
	}

	return devices, nil
}

// ConnectRFCOMM establishes an RFCOMM connection to a Bluetooth MAC address
// Returns the device path (e.g., /dev/rfcomm0)
func ConnectRFCOMM(mac string, channel int) (string, error) {
	// Find an available rfcomm device number
	devNum := 0
	for i := 0; i < 10; i++ {
		devPath := fmt.Sprintf("/dev/rfcomm%d", i)
		// Try to use this device number
		cmd := exec.Command("rfcomm", "connect", devPath, mac, fmt.Sprintf("%d", channel))
		if err := cmd.Start(); err != nil {
			continue
		}

		// Give it a moment to connect
		time.Sleep(2 * time.Second)

		// Check if device exists now
		if _, err := exec.Command("test", "-e", devPath).Output(); err == nil {
			return devPath, nil
		}

		cmd.Process.Kill()
		devNum++
	}

	return "", errors.New("failed to establish RFCOMM connection")
}

// Connect opens a connection to the printer on the given serial port
func Connect(portName string) (*Printer, error) {
	return ConnectWithLog(portName, nil)
}

func ConnectWithLog(portName string, log *CommunicationLog) (*Printer, error) {
	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(portName, mode)
	if err != nil {
		log.Record(portName, "OPEN", nil, err.Error())
		return nil, fmt.Errorf("failed to open port %s: %w", portName, err)
	}

	if err = port.SetReadTimeout(3 * time.Second); err != nil {
		port.Close()
		log.Record(portName, "OPEN", nil, err.Error())
		return nil, err
	}
	log.Record(portName, "OPEN", nil, "115200 baud, 8N1; read timeout 3s")
	port = &loggedPort{Port: port, log: log, name: portName}

	p := &Printer{
		port:     port,
		portName: portName,
		log:      log,
	}

	return p, nil
}

// Close closes the printer connection
func (p *Printer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.port != nil {
		err := p.port.Close()
		p.port = nil
		return err
	}
	return nil
}

// queryLocked reads binary bytes until LF or the existing serial read timeout.
// Partial replies and replies without LF are retained and logged unchanged.
func (p *Printer) queryLocked(cmd string) ([]byte, error) {
	if p.port == nil {
		return nil, ErrNotConnected
	}
	if err := p.writeAllLocked([]byte(cmd + "\r\n")); err != nil {
		return nil, err
	}
	var response []byte
	buf := make([]byte, 256)
	for len(response) < 4096 {
		n, err := p.port.Read(buf)
		if n < 0 || n > len(buf) {
			return response, fmt.Errorf("invalid read count %d", n)
		}
		response = append(response, buf[:n]...)
		if err != nil {
			return response, fmt.Errorf("read failed: %w", err)
		}
		if n == 0 {
			if len(response) == 0 {
				return nil, ErrTimeout
			}
			return response, nil
		}
		start := 0
		if cmd == "BATTERY?" {
			start = 8
		}
		if len(response) > start && bytes.Contains(response[start:], []byte{'\n'}) {
			return response, nil
		}
	}
	return response, errors.New("response exceeds 4096 bytes")
}

func (p *Printer) sendCommandLocked(cmd string) (string, error) {
	response, err := p.queryLocked(cmd)
	return string(bytes.TrimSuffix(response, []byte("\r\n"))), err
}

// GetBattery keeps the existing BATTERY? query. Binary payloads are never
// TrimSpace'd: valid percentages include whitespace bytes such as 10 and 32.
func (p *Printer) GetBattery() (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	response, err := p.queryLocked("BATTERY?")
	if err != nil {
		return 0, err
	}
	if !bytes.HasPrefix(response, []byte("BATTERY")) || len(response) < 8 {
		return 0, errors.New("invalid battery response")
	}
	level := int(response[7])
	if level > 100 {
		return 0, errors.New("battery percentage outside 0..100")
	}
	return level, nil
}

// GetConfig queries printer configuration
func (p *Printer) GetConfig() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.sendCommandLocked("CONFIG?")
}

// CancelPause sends escape sequence to cancel pause status
func (p *Printer) CancelPause() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.port == nil {
		return ErrNotConnected
	}
	if err := p.cancelPauseLocked(); err != nil {
		return err
	}
	p.waitLocked(CancelPauseDelay)
	return nil
}

// cancelPauseLocked sends the cancel-pause escape sequence. Caller must hold mu.
func (p *Printer) cancelPauseLocked() error {
	return p.writeAllLocked([]byte{0x1b, 0x21, 0x6f})
}

// CheckReady checks if printer is ready
func (p *Printer) CheckReady() (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.port == nil {
		return false, ErrNotConnected
	}
	err := p.writeAllLocked([]byte("\x1b!?"))
	if err != nil {
		return false, err
	}
	// Read response
	buf := make([]byte, 32)
	_, err = p.port.Read(buf)
	return err == nil, err
}

// Ping checks the printer is alive by sending a status query and waiting for
// a response. A bare Write always "succeeds" over an open serial port even if
// the device is off, so reading back is the only reliable liveness signal.
func (p *Printer) Ping() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.port == nil {
		return ErrNotConnected
	}
	_ = p.port.ResetInputBuffer()
	if err := p.writeAllLocked([]byte("BATTERY?\r\n")); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	buf := make([]byte, 32)
	n, err := p.port.Read(buf) // read timeout was set in Connect
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}
	if n == 0 {
		return errors.New("no response from device (powered off or out of range)")
	}
	return nil
}

const CancelPauseDelay = 500 * time.Millisecond
const PrintSettleDelay = 3 * time.Second

func (p *Printer) waitLocked(delay time.Duration) {
	p.log.Record(p.portName, "WAIT", nil, delay.String())
	if p.sleep != nil {
		p.sleep(delay)
	} else {
		time.Sleep(delay)
	}
}

// writeAllLocked handles short serial writes and never skips a byte.
func (p *Printer) writeAllLocked(data []byte) error {
	for len(data) > 0 {
		n, err := p.port.Write(data)
		if n < 0 || n > len(data) {
			return io.ErrShortWrite
		}
		data = data[n:]
		if err != nil {
			return fmt.Errorf("write failed: %w", err)
		}
		if n == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

// Print holds the same serial mutex over the complete reference sequence,
// preventing battery/status commands during the two required quiet periods.
// Successful return means the bytes were written and the settle delay elapsed;
// it is not a printer acknowledgement or confirmation of physical output.
func (p *Printer) Print(data []byte) error { return p.PrintChunks([][]byte{data}) }

// PrintChunks preserves the caller's write boundaries while holding the same
// transaction lock across all parts and both quiet periods.
func (p *Printer) PrintChunks(parts [][]byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.port == nil {
		return ErrNotConnected
	}
	if err := p.cancelPauseLocked(); err != nil {
		return fmt.Errorf("cancel pause: %w", err)
	}
	p.waitLocked(CancelPauseDelay)
	for _, data := range parts {
		if err := p.writeAllLocked(data); err != nil {
			return fmt.Errorf("print failed: %w", err)
		}
	}
	p.waitLocked(PrintSettleDelay)
	return nil
}

// PortName returns the current port name
func (p *Printer) PortName() string {
	return p.portName
}

// Alive reports whether the serial device still exists. On macOS/Linux a
// Bluetooth serial node disappears when the device truly disconnects, so this
// is a reliable liveness signal that (unlike a status read) does not require
// the printer to answer queries it may not support.
func (p *Printer) Alive() bool {
	if p.portName == "" {
		return false
	}
	// COM ports on Windows are not filesystem paths; treat as always present.
	if strings.HasPrefix(strings.ToUpper(p.portName), "COM") || strings.HasPrefix(p.portName, `\\.\`) {
		return true
	}
	_, err := os.Stat(p.portName)
	return err == nil
}
