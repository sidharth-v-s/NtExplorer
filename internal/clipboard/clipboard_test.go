package clipboard

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestWriteEncodesOSC52(t *testing.T) {
	var out strings.Builder
	if err := Write(&out, "hello clipboard\n"); err != nil {
		t.Fatalf("Write: %v", err)
	}
	sequence := out.String()
	if !strings.HasPrefix(sequence, "\x1b]52;c;") {
		t.Fatalf("missing OSC 52 prefix: %q", sequence)
	}
	if !strings.HasSuffix(sequence, "\x07") {
		t.Fatalf("missing BEL terminator: %q", sequence)
	}
	payload := strings.TrimSuffix(strings.TrimPrefix(sequence, "\x1b]52;c;"), "\x07")
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("payload is not base64: %v", err)
	}
	if string(decoded) != "hello clipboard" {
		t.Fatalf("decoded payload = %q", decoded)
	}
}

func TestWriteRejectsEmptyAndOversizedText(t *testing.T) {
	var out strings.Builder
	if err := Write(&out, " \n\n"); err == nil {
		t.Fatal("expected an error for empty text")
	}
	if out.Len() != 0 {
		t.Fatal("empty text should not write a sequence")
	}
	if err := Write(&out, strings.Repeat("x", MaxBytes+1)); err == nil {
		t.Fatal("expected an error for oversized text")
	}
	if out.Len() != 0 {
		t.Fatal("oversized text should not write a sequence")
	}
}
