// Package exif implements decoding of EXIF data as defined in the EXIF 2.2
// specification.
package exif

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/rwcarlsen/goexif/tiff"
)

var validField map[FieldName]bool

func init() {
	validField = make(map[FieldName]bool)
	addValidFields(exifFields)
	addValidFields(gpsFields)
	addValidFields(gpsFields)
	addValidFields(makerNoteCanonFields)
	addValidFields(makerNoteNikon3Fields)
}

func addValidFields(fields map[uint16]FieldName) {
	for _, name := range fields {
		validField[name] = true
	}
}

const (
	jpeg_APP1 = 0xE1

	exifPointer    = 0x8769
	gpsPointer     = 0x8825
	interopPointer = 0xA005
)

// A TagNotPresentError is returned when the requested field is not
// present in the EXIF.
type TagNotPresentError FieldName

func (tag TagNotPresentError) Error() string {
	return fmt.Sprintf("exif: tag %q is not present", string(tag))
}

func isTagNotPresentErr(err error) bool {
	_, ok := err.(TagNotPresentError)
	return ok
}

// Exif provides access to decoded EXIF metadata fields and values.
type Exif struct {
	tif  *tiff.Tiff
	main map[FieldName]*tiff.Tag
}

// Decode parses EXIF-encoded data from r and returns a queryable Exif object.
func Decode(r io.Reader) (*Exif, error) {
	// Locate the EXIF application section.
	sec, err := newAppSec(jpeg_APP1, r)
	if err != nil {
		return nil, err
	}
	er, err := sec.exifReader()
	if err != nil {
		return nil, err
	}
	tif, err := tiff.Decode(er)
	if err != nil {
		return nil, errors.New("exif: decode failed: " + err.Error())
	}

	// build an exif structure from the tiff
	x := &Exif{
		main: map[FieldName]*tiff.Tag{},
		tif:  tif,
	}

	x.loadDirTags(tif.Dirs[0], exifFields)

	// recurse into exif, gps, and interop sub-IFDs
	if err = x.loadSubDir(er, ExifIFDPointer, exifFields); err != nil {
		return x, err
	}
	if err = x.loadSubDir(er, GPSInfoIFDPointer, gpsFields); err != nil {
		return x, err
	}
	if err = x.loadSubDir(er, InteroperabilityIFDPointer, interopFields); err != nil {
		return x, err
	}

	// load MakerNote fields, but don't abort if none are readable
	x.loadMakerNotes(er)

	return x, nil
}

func (x *Exif) loadMakerNotes(r *bytes.Reader) {
	if m, err := x.Get(MakerNote); err == nil {
		if mk, err := x.Get(Make); err == nil && mk.StringVal() == "Canon" {
			x.loadCanonMakerNotes(r)
		} else if bytes.Compare(m.Val[:6], []byte("Nikon\000")) == 0 {
			x.loadNikonV3MakerNotes(m)
		}
	}
}

func (x *Exif) loadCanonMakerNotes(r *bytes.Reader) {
	// Canon maker note is an IFD with absolute offsets from the start
	// of the TIFF header
	offset := int64(826) // XXX - need offset to Maker Note data!
	if _, err := r.Seek(offset, 0); err != nil {
		fmt.Printf("Failed to seek to start of TIFF header: %v\n", err)
		return
	}
	if subDir, _, err := tiff.DecodeDir(r, x.tif.Order); err == nil {
		x.loadDirTags(subDir, makerNoteCanonFields)
	} else {
		fmt.Printf("Error loading Canon Maker note: %v\n", err)
	}
}

func (x *Exif) loadNikonV3MakerNotes(m *tiff.Tag) {
	// Nikon v3 maker note is a self-contained IFD (offsets are relative
	// to the start of the maker note)
	mkNotes, err := tiff.Decode(bytes.NewReader(m.Val[10:]))
	// fmt.Printf("Maker notes (err %v): %v", err, mkNotes)
	if err == nil {
		x.loadDirTags(mkNotes.Dirs[0], makerNoteNikon3Fields)
	}
}

func (x *Exif) loadSubDir(r *bytes.Reader, ptrName FieldName, fieldMap map[uint16]FieldName) error {
	tag, ok := x.main[ptrName]
	if !ok {
		return nil
	}
	offset := tag.Int(0)

	_, err := r.Seek(offset, 0)
	if err != nil {
		return errors.New("exif: seek to sub-IFD failed: " + err.Error())
	}
	subDir, _, err := tiff.DecodeDir(r, x.tif.Order)
	if err != nil {
		return errors.New("exif: sub-IFD decode failed: " + err.Error())
	}
	x.loadDirTags(subDir, fieldMap)
	return nil
}

func (x *Exif) loadDirTags(d *tiff.Dir, fieldMap map[uint16]FieldName) {
	for _, tag := range d.Tags {
		name := fieldMap[tag.Id]
		if name == "" {
			name = FieldName(fmt.Sprintf("%v%x", unknownPrefix, tag.Id))
		}
		x.main[name] = tag
	}
}

// Get retrieves the EXIF tag for the given field name.
//
// If the tag is not known or not present, an error is returned. If the
// tag name is known, the error will be a TagNotPresentError.
func (x *Exif) Get(name FieldName) (*tiff.Tag, error) {
	if !validField[name] {
		return nil, fmt.Errorf("exif: invalid tag name %q", name)
	} else if tg, ok := x.main[name]; ok {
		return tg, nil
	}
	return nil, TagNotPresentError(name)
}

// Walker is the interface used to traverse all fields of an Exif object.
type Walker interface {
	// Walk is called for each non-nil EXIF field. Returning a non-nil
	// error aborts the walk/traversal.
	Walk(name FieldName, tag *tiff.Tag) error
}

// Walk calls the Walk method of w with the name and tag for every non-nil
// EXIF field.  If w aborts the walk with an error, that error is returned.
func (x *Exif) Walk(w Walker) error {
	for name, tag := range x.main {
		if err := w.Walk(name, tag); err != nil {
			return err
		}
	}
	return nil
}

// String returns a pretty text representation of the decoded exif data.
func (x *Exif) String() string {
	var buf bytes.Buffer
	for name, tag := range x.main {
		fmt.Fprintf(&buf, "%s: %s\n", name, tag)
	}
	return buf.String()
}

// MarshalJson implements the encoding/json.Marshaler interface providing output of
// all EXIF fields present (names and values).
func (x Exif) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.main)
}

type appSec struct {
	marker byte
	data   []byte
}

// newAppSec finds marker in r and returns the corresponding application data
// section.
func newAppSec(marker byte, r io.Reader) (*appSec, error) {
	br := bufio.NewReader(r)
	app := &appSec{marker: marker}
	var dataLen int

	// seek to marker
	for dataLen == 0 {
		if _, err := br.ReadBytes(0xFF); err != nil {
			return nil, err
		}
		c, err := br.ReadByte()
		if err != nil {
			return nil, err
		} else if c != marker {
			continue
		}

		dataLenBytes, err := br.Peek(2)
		if err != nil {
			return nil, err
		}
		dataLen = int(binary.BigEndian.Uint16(dataLenBytes))
	}

	// read section data
	nread := 0
	for nread < dataLen {
		s := make([]byte, dataLen-nread)
		n, err := br.Read(s)
		nread += n
		if err != nil && nread < dataLen {
			return nil, err
		}
		app.data = append(app.data, s[:n]...)
	}
	app.data = app.data[2:] // exclude dataLenBytes
	return app, nil
}

// reader returns a reader on this appSec.
func (app *appSec) reader() *bytes.Reader {
	return bytes.NewReader(app.data)
}

// exifReader returns a reader on this appSec with the read cursor advanced to
// the start of the exif's tiff encoded portion.
func (app *appSec) exifReader() (*bytes.Reader, error) {
	if len(app.data) < 6 {
		return nil, errors.New("exif: failed to find exif intro marker")
	}

	// read/check for exif special mark
	exif := app.data[:6]
	if !bytes.Equal(exif, append([]byte("Exif"), 0x00, 0x00)) {
		return nil, errors.New("exif: failed to find exif intro marker")
	}
	return bytes.NewReader(app.data[6:]), nil
}
