// Package exif implements decoding of EXIF data as defined in the EXIF 2.2
// specification.
package exif

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"

	"github.com/rwcarlsen/goexif/tiff"
)

const (
	exifPointer    = 0x8769
	gpsPointer     = 0x8825
	interopPointer = 0xA005
)


type Exif struct {
	tif *tiff.Tiff

	main   map[uint16]*tiff.Tag
	gps       map[uint16]*tiff.Tag
	interOp       map[uint16]*tiff.Tag
}

func (x Exif) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{}

	for name, id := range fields {
		if tag, ok := x.main[id]; ok {
			m[name] = tag
		}
	}

	for name, id := range gpsFields {
		if tag, ok := x.gps[id]; ok {
			m[name] = tag
		}
	}
	for name, id := range interOpFields {
		if tag, ok := x.interOp[id]; ok {
			m[name] = tag
		}
	}

	return json.Marshal(m)
}

// Decode parses exif encoded data from r and returns a queryable Exif object.
func Decode(r io.Reader) (*Exif, error) {
	sec, err := newAppSec(0xE1, r)
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
		main:          map[uint16]*tiff.Tag{},
		gps:           map[uint16]*tiff.Tag{},
		interOp:       map[uint16]*tiff.Tag{},
		tif:           tif,
	}

	ifd0 := tif.Dirs[0]
	for _, tag := range ifd0.Tags {
		x.main[tag.Id] = tag
	}

	// recurse into exif, gps, and interop sub-IFDs
	if err = x.loadSubDir(er, exifPointer, x.main); err != nil {
		return x, err
	}
	if err = x.loadSubDir(er, gpsPointer, x.gps); err != nil {
		return x, err
	}
	if err = x.loadSubDir(er, interopPointer, x.interOp); err != nil {
		return x, err
	}

	return x, nil
}

func (x *Exif) loadSubDir(r *bytes.Reader, tagId uint16, tags map[uint16]*tiff.Tag) error {
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
		tags[tag.Id] = tag
	}
	return nil
}

// Get retrieves the exif tag for the given field name. It returns nil if the
// tag name is not found.
func (x *Exif) Get(name string) *tiff.Tag {
	if tg, ok := x.main[fields[name]]; ok {
		return tg
	} else if tg, ok := x.gps[gpsFields[name]]; ok {
		return tg
	} else if tg, ok := x.interOp[interOpFields[name]]; ok {
		return tg
	}
	return nil
}

func (x *Exif) Iter() func() (string, *tiff.Tag) {
  i := 0
  return func() (string, *tiff.Tag) {
    if i == len(fieldList) {
      return "", nil
    }
    next := fieldList[i]
    i++
    return next, x.Get(next)
  }
}

// String returns a pretty text representation of the decoded exif data.
func (x *Exif) String() string {
	msg := "Main:\n"
	for name, id := range fields {
		if tag, ok := x.main[id]; ok {
			msg += name + ":" + tag.String() + "\n"
		}
	}
	msg += "\n\nGPS:\n"
	for name, id := range gpsFields {
		if tag, ok := x.gps[id]; ok {
			msg += name + ":" + tag.String() + "\n"
		}
	}
	msg += "\n\nInteroperability:\n"
	for name, id := range interOpFields {
		if tag, ok := x.interOp[id]; ok {
			msg += name + ":" + tag.String() + "\n"
		}
	}
	return msg
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
