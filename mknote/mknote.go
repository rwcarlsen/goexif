// Package mknote implements decoding of EXIF makernote data from media
// files.
package mknote

import (
	"bytes"
	"errors"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
)

// Decode decodes all makernote data found in x and adds it to x.  n is the
// number of fields found/decoded from the makernote.
func Decode(x *exif.Exif) (n int, err error) {
	m, err := x.Get(exif.MakerNote)
	if err != nil {
		return 0, errors.New("makernote: no makernote data found")
	}

	mk, err := x.Get(exif.Make)
	if err != nil {
		return 0, errors.New("makernote: no make data found")
	}

	if mk.StringVal() == "Canon" {
		return loadCanon(x, m)
	} else if bytes.Compare(m.Val[:6], []byte("Nikon\000")) == 0 {
		return loadNikonV3(x, m)
	} else {
		return 0, errors.New("makernote: unsupported make")
	}
}

func loadCanon(x *exif.Exif, m *tiff.Tag) (n int, err error) {
	// Canon notes are a single IFD directory with no header.
	// Reader offsets need to be w.r.t. the original tiff structure.
	buf := bytes.NewReader(append(make([]byte, m.ValOffset), m.Val...))
	buf.Seek(int64(m.ValOffset), 0)

	mkNotesDir, _, err := tiff.DecodeDir(buf, x.Tiff.Order)
	if err != nil {
		return 0, err
	}
	x.LoadDirTags(mkNotesDir, makerNoteCanonFields)
	return len(mkNotesDir.Tags), nil
}

func loadNikonV3(x *exif.Exif, m *tiff.Tag) (n int, err error) {
	// Nikon v3 maker note is a self-contained IFD (offsets are relative
	// to the start of the maker note)
	mkNotes, err := tiff.Decode(bytes.NewReader(m.Val[10:]))
	if err != nil {
		return 0, err
	}
	x.LoadDirTags(mkNotes.Dirs[0], makerNoteNikon3Fields)
	return len(mkNotes.Dirs[0].Tags), nil
}
