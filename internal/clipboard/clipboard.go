// Package clipboard writes text to the terminal clipboard using OSC 52.
//
// OSC 52 is a terminal control sequence, not a system API call. The package
// never reads the clipboard, starts helper processes, or accesses the network.
package clipboard

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	osc52 "github.com/aymanbagabas/go-osc52/v2"
)

// MaxBytes bounds a single clipboard write. Terminals commonly cap OSC 52
// payloads; keeping the raw text under 64 KiB leaves room for base64 encoding.
const MaxBytes = 64 * 1024

// ErrUnavailable means no terminal was available to receive the sequence.
var ErrUnavailable = errors.New("terminal clipboard is unavailable")

// Copy sends text to the controlling terminal's clipboard. Most modern
// terminals support OSC 52, including over SSH; unsupported terminals simply
// ignore the sequence.
func Copy(text string) error {
	text, err := prepare(text)
	if err != nil {
		return err
	}
	writer, closeFn, err := terminalWriter()
	if err != nil {
		return err
	}
	defer closeFn()
	return Write(writer, text)
}

// Write emits the OSC 52 sequence for text to writer. It is exported for
// tests and for callers that already own a terminal writer.
func Write(writer io.Writer, text string) error {
	text, err := prepare(text)
	if err != nil {
		return err
	}
	_, err = osc52.New(text).WriteTo(writer)
	return err
}

func prepare(text string) (string, error) {
	text = strings.TrimRight(text, "\r\n")
	if strings.TrimSpace(text) == "" {
		return "", errors.New("nothing to copy")
	}
	if len(text) > MaxBytes {
		return "", fmt.Errorf("selection is %d bytes; clipboard limit is %d bytes", len(text), MaxBytes)
	}
	return text, nil
}

// terminalWriter opens the controlling terminal so the copy sequence is not
// interleaved with Bubble Tea's stdout renderer.
func terminalWriter() (io.Writer, func(), error) {
	var (
		file *os.File
		err  error
	)
	if runtime.GOOS == "windows" {
		file, err = os.OpenFile("CONOUT$", os.O_WRONLY, 0)
	} else {
		file, err = os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	}
	if err == nil {
		return file, func() { _ = file.Close() }, nil
	}
	// A redirected process may have no controlling terminal. Stderr still
	// reaches the emulator in most interactive setups and never panics.
	if os.Stderr != nil {
		return os.Stderr, func() {}, nil
	}
	return nil, func() {}, fmt.Errorf("%w: %v", ErrUnavailable, err)
}
