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

const (
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

type Exif struct {
	tif *tiff.Tiff

	main map[uint16]*tiff.Tag
}

// Decode parses EXIF-encoded data from r and returns a queryable Exif object.
func Decode(r io.Reader) (*Exif, error) {
	// EXIF data in JPEG is stored in the APP1 marker. EXIF data uses the TIFF
	// format to store data.
	// If we're parsing a TIFF image, we don't need to strip away any data.
	// If we're parsing a JPEG image, we need to strip away the JPEG APP1
	// marker and also the EXIF header.
	header := make([]byte, 4)
	n, err := r.Read(header)
	if n < len(header) {
		return nil, errors.New("exif: short read on header")
	}
	if err != nil {
		return nil, err
	}

	var isTiff bool
	switch string(header) {
	case "II*\x00":
		// TIFF - Little endian (Intel)
		isTiff = true
	case "MM\x00*":
		// TIFF - Big endian (Motorola)
		isTiff = true
	default:
		// Not TIFF, assume JPEG
	}

	// Put the header bytes back into the reader.
	r = io.MultiReader(bytes.NewReader(header), r)
	var (
		er  *bytes.Reader
		tif *tiff.Tiff
	)

	if isTiff {
		// Functions below need the IFDs from the TIFF data to be stored in a
		// *bytes.Reader.  We use TeeReader to get a copy of the bytes as a
		// side-effect of tiff.Decode() doing its work.
		b := &bytes.Buffer{}
		tr := io.TeeReader(r, b)
		tif, err = tiff.Decode(tr)
		er = bytes.NewReader(b.Bytes())
	} else {
		// Strip away JPEG APP1 header.
		sec, err := newAppSec(0xE1, r)
		if err != nil {
			return nil, err
		}
		// Strip away EXIF header.
		er, err = sec.exifReader()
		if err != nil {
			return nil, err
		}
		tif, err = tiff.Decode(er)
	}

	if err != nil {
		return nil, errors.New("exif: decode failed: " + err.Error())
	}

	// build an exif structure from the tiff
	x := &Exif{
		main: map[uint16]*tiff.Tag{},
		tif:  tif,
	}

	ifd0 := tif.Dirs[0]
	for _, tag := range ifd0.Tags {
		x.main[tag.Id] = tag
	}

	// recurse into exif, gps, and interop sub-IFDs
	if err = x.loadSubDir(er, exifPointer); err != nil {
		return x, err
	}
	if err = x.loadSubDir(er, gpsPointer); err != nil {
		return x, err
	}
	if err = x.loadSubDir(er, interopPointer); err != nil {
		return x, err
	}

	return x, nil
}

func (x *Exif) loadSubDir(r *bytes.Reader, tagId uint16) error {
	tag, ok := x.main[tagId]
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
	for _, tag := range subDir.Tags {
		x.main[tag.Id] = tag
	}
	return nil
}

// Get retrieves the EXIF tag for the given field name.
//
// If the tag is not known or not present, an error is returned. If the
// tag name is known, the error will be a TagNotPresentError.
func (x *Exif) Get(name FieldName) (*tiff.Tag, error) {
	id, ok := fields[name]
	if !ok {
		return nil, fmt.Errorf("exif: invalid tag name %q", name)
	}
	if tg, ok := x.main[id]; ok {
		return tg, nil
	}
	return nil, TagNotPresentError(name)
}

// Walker is the interface used to traverse all exif fields of an Exif object.
// Returning a non-nil error aborts the walk/traversal.
type Walker interface {
	Walk(name FieldName, tag *tiff.Tag) error
}

// Walk calls the Walk method of w with the name and tag for every non-nil exif
// field.
func (x *Exif) Walk(w Walker) error {
	for name, _ := range fields {
		tag, err := x.Get(name)
		if isTagNotPresentErr(err) {
			continue
		} else if err != nil {
			panic("field list access/construction is broken - this should never happen")
		}

		err = w.Walk(name, tag)
		if err != nil {
			return err
		}
	}
	return nil
}

// String returns a pretty text representation of the decoded exif data.
func (x *Exif) String() string {
	var buf bytes.Buffer
	for name, id := range fields {
		if tag, ok := x.main[id]; ok {
			fmt.Fprintf(&buf, "%s: %s\n", name, tag)
		}
	}
	return buf.String()
}

func (x Exif) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{}

	for name, id := range fields {
		if tag, ok := x.main[id]; ok {
			m[string(name)] = tag
		}
	}

	return json.Marshal(m)
}

type appSec struct {
	marker byte
	data   []byte
}

// newAppSec finds marker in r and returns the corresponding application data
// section.
func newAppSec(marker byte, r io.Reader) (*appSec, error) {
	app := &appSec{marker: marker}

	buf := bufio.NewReader(r)

	// seek to marker
	for {
		b, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}
		n, err := buf.Peek(1)
		if b == 0xFF && n[0] == marker {
			buf.ReadByte()
			break
		}
	}

	// read section size
	var dataLen uint16
	err := binary.Read(buf, binary.BigEndian, &dataLen)
	if err != nil {
		return nil, err
	}
	dataLen -= 2 // subtract length of the 2 byte size marker itself

	// read section data
	nread := 0
	for nread < int(dataLen) {
		s := make([]byte, int(dataLen)-nread)
		n, err := buf.Read(s)
		if err != nil {
			return nil, err
		}
		nread += n
		app.data = append(app.data, s...)
	}

	return app, nil
}

// reader returns a reader on this appSec.
func (app *appSec) reader() *bytes.Reader {
	return bytes.NewReader(app.data)
}

// exifReader returns a reader on this appSec with the read cursor advanced to
// the start of the exif's tiff encoded portion.
func (app *appSec) exifReader() (*bytes.Reader, error) {
	// read/check for exif special mark
	if len(app.data) < 6 {
		return nil, errors.New("exif: failed to find exif intro marker")
	}

	exif := app.data[:6]
	if string(exif) != "Exif"+string([]byte{0x00, 0x00}) {
		return nil, errors.New("exif: failed to find exif intro marker")
	}
	return bytes.NewReader(app.data[6:]), nil
}
