// Package arduino handles automatic USB serial communication with the game controller.
package arduino

import (
	"context"
	"errors"
	"io"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.bug.st/serial"
)

const baudRate = 115200

// Frame contains one decoded controller message and its lowercase field names.
type Frame struct {
	Type   string
	Fields map[string]string
}

// Run()
// Discovers the USB Arduino and reconnects until cancelled, delivering parsed serial frames.
func Run(ctx context.Context, handle func(Frame)) {
	for ctx.Err() == nil {
		port, device, err := openSerialPort()
		if err == nil {
			_ = port.SetReadTimeout(time.Second)
			log.Printf("Arduino verbunden: %s mit %d Baud", device, baudRate)

			// UNO per USB-Serial
			if isUSBSerial(device) {
				select {
				case <-ctx.Done():
					_ = port.Close()
					return
				case <-time.After(2 * time.Second):
				}
			}
			err = readSerial(ctx, port, handle)
			_ = port.Close()
		}
		if err != nil && ctx.Err() == nil {
			log.Printf("Arduino nicht verfuegbar: %v; neuer Versuch in 2 s", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

// openSerialPort()
// Opens the first available USB Arduino at the configured baud rate.
func openSerialPort() (serial.Port, string, error) {
	devices, err := serialDevices()
	if err != nil {
		return nil, "", err
	}

	var openErrors []string
	for _, device := range devices {
		port, openErr := serial.Open(device, &serial.Mode{BaudRate: baudRate})
		if openErr == nil {
			return port, device, nil
		}
		openErrors = append(openErrors, device+": "+openErr.Error())
	}
	return nil, "", errors.New(strings.Join(openErrors, "; "))
}

// serialDevices()
// Finds and deduplicates supported USB serial device paths on Linux and macOS.
func serialDevices() ([]string, error) {
	patterns := []string{
		"/dev/serial/by-id/*",
		"/dev/ttyACM*",
		"/dev/ttyUSB*",
		"/dev/cu.usbmodem*",
		"/dev/cu.usbserial*",
		"/dev/cu.wchusbserial*",
		"/dev/cu.SLAB_USBtoUART*",
		"/dev/tty.usbmodem*",
		"/dev/tty.usbserial*",
	}
	seen := make(map[string]bool)
	var devices []string
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		sort.Strings(matches)
		for _, device := range matches {
			resolved, err := filepath.EvalSymlinks(device)
			if err != nil {
				continue
			}
			if !seen[resolved] {
				seen[resolved] = true
				devices = append(devices, device)
			}
		}
	}
	if len(devices) == 0 {
		return nil, errors.New("kein USB-Arduino unter Linux oder macOS gefunden")
	}
	return devices, nil
}

// isUSBSerial()
// Identifies USB device names that need time to reboot after opening the port.
func isUSBSerial(device string) bool {
	return strings.Contains(device, "/dev/serial/by-id/") ||
		strings.Contains(device, "/dev/ttyACM") ||
		strings.Contains(device, "/dev/ttyUSB") ||
		strings.Contains(device, "usbmodem") ||
		strings.Contains(device, "usbserial") ||
		strings.Contains(device, "USBtoUART")
}

// readSerial()
// Reads bounded newline-delimited frames and forwards valid messages to the handler.
func readSerial(ctx context.Context, port io.Reader, handle func(Frame)) error {
	buf, pending := make([]byte, 256), make([]byte, 0, 256)
	discardLine := false
	for ctx.Err() == nil {
		n, err := port.Read(buf)
		for _, value := range buf[:n] {
			if value != '\n' {
				if discardLine {
					continue
				}
				pending = append(pending, value)
				if len(pending) > 4096 {
					pending = pending[:0]
					discardLine = true
				}
				continue
			}
			if discardLine {
				discardLine = false
				continue
			}
			if f, ok := ParseFrame(strings.TrimSpace(string(pending))); ok {
				handle(f)
			}
			pending = pending[:0]
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// ParseFrame()
// Decodes an Arduino frame and normalizes field names without applying game logic.
func ParseFrame(line string) (Frame, bool) {
	if len(line) < 3 || line[0] != '<' || line[len(line)-1] != '>' {
		return Frame{}, false
	}
	parts := strings.Split(line[1:len(line)-1], "|")
	f := Frame{Type: parts[0], Fields: map[string]string{}}
	for _, part := range parts[1:] {
		if key, value, ok := strings.Cut(part, "="); ok {
			f.Fields[strings.ToLower(key)] = value
		}
	}
	return f, f.Type != ""
}
