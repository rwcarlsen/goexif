package mknote

import (
	"bytes"
	"testing"

	"github.com/rwcarlsen/goexif/exif"
)

// nikonV3.Parse sliced m.Val[:6] (and later m.Val[10:]) before checking the
// MakerNote length, so a MakerNote (tag 0x927C) shorter than those bounds
// panicked regardless of the image's make. exif.Decode must not panic.
func TestNikonShortMakerNoteNoPanic(t *testing.T) {
	exif.RegisterParsers(NikonV3)
	// Little-endian TIFF, one IFD0 tag Id=0x927C type=7 (UNDEFINED) count=3:
	// a 3-byte inline MakerNote value, shorter than the 6/10 bytes Parse sliced.
	tiffBytes := []byte{
		'I', 'I', 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00,
		0x01, 0x00,
		0x7C, 0x92, 0x07, 0x00, 0x03, 0x00, 0x00, 0x00,
		0x01, 0x02, 0x03, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("exif.Decode panicked on a short MakerNote: %v", r)
		}
	}()
	_, _ = exif.Decode(bytes.NewReader(tiffBytes))
}
