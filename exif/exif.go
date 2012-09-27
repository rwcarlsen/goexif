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
	fields map[string]uint16

	gps       map[uint16]*tiff.Tag
	gpsFields map[string]uint16

	interOp       map[uint16]*tiff.Tag
	interOpFields map[string]uint16
}

func (x Exif) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{}

	for name, id := range x.fields {
		if tag, ok := x.main[id]; ok {
			m[name] = tag
		}
	}

	for name, id := range x.gpsFields {
		if tag, ok := x.gps[id]; ok {
			m[name] = tag
		}
	}
	for name, id := range x.interOpFields {
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
		fields:        map[string]uint16{},
		gps:           map[uint16]*tiff.Tag{},
		gpsFields:     map[string]uint16{},
		interOp:       map[uint16]*tiff.Tag{},
		interOpFields: map[string]uint16{},
		tif:           tif,
	}
	x.loadStdFields()
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

func (x *Exif) loadStdFields() {
	/////////////////////////////////////
	////////// IFD 0 ////////////////////
	/////////////////////////////////////

	// image data structure
	x.fields["ImageWidth"] = 0x0100
	x.fields["ImageLength"] = 0x0101
	x.fields["BitsPerSample"] = 0x0102
	x.fields["Compression"] = 0x0103
	x.fields["PhotometricInterpretation"] = 0x0106
	x.fields["Orientation"] = 0x0112
	x.fields["SamplesPerPixel"] = 0x0115
	x.fields["PlanarConfiguration"] = 0x011C
	x.fields["YCbCrSubSampling"] = 0x0212
	x.fields["YCbCrPositioning"] = 0x0213
	x.fields["XResolution"] = 0x011A
	x.fields["YResolution"] = 0x011B
	x.fields["ResolutionUnit"] = 0x0128

	// Other tags
	x.fields["DateTime"] = 0x0132
	x.fields["ImageDescription"] = 0x010E
	x.fields["Make"] = 0x010F
	x.fields["Model"] = 0x0110
	x.fields["Software"] = 0x0131
	x.fields["Artist"] = 0x010e
	x.fields["Copyright"] = 0x010e

	// private tags
	x.fields["ExifIFDPointer"] = exifPointer

	/////////////////////////////////////
	////////// Exif sub IFD /////////////
	/////////////////////////////////////

	x.fields["GPSInfoIFDPointer"] = gpsPointer
	x.fields["InteroperabilityIFDPointer"] = interopPointer

	x.fields["ExifVersion"] = 0x9000
	x.fields["FlashpixVersion"] = 0xA000

	x.fields["ColorSpace"] = 0xA001

	x.fields["ComponentsConfiguration"] = 0x9101
	x.fields["CompressedBitsPerPixel"] = 0x9102
	x.fields["PixelXDimension"] = 0xA002
	x.fields["PixelYDimension"] = 0xA003

	x.fields["MakerNote"] = 0x927C
	x.fields["UserComment"] = 0x9286

	x.fields["RelatedSoundFile"] = 0xA004
	x.fields["DateTimeOriginal"] = 0x9003
	x.fields["DateTimeDigitized"] = 0x9004
	x.fields["SubSecTime"] = 0x9290
	x.fields["SubSecTimeOriginal"] = 0x9291
	x.fields["SubSecTimeDigitized"] = 0x9292

	x.fields["ImageUniqueID"] = 0xA420

	// picture conditions
	x.fields["ExposureTime"] = 0x829A
	x.fields["FNumber"] = 0x829D
	x.fields["ExposureProgram"] = 0x8822
	x.fields["SpectralSensitivity"] = 0x8824
	x.fields["ISOSpeedRatings"] = 0x8827
	x.fields["OECF"] = 0x8828
	x.fields["ShutterSpeedValue"] = 0x9201
	x.fields["ApertureValue"] = 0x9202
	x.fields["BrightnessValue"] = 0x9203
	x.fields["ExposureBiasValue"] = 0x9204
	x.fields["MaxApertureValue"] = 0x9205
	x.fields["SubjectDistance"] = 0x9206
	x.fields["MeteringMode"] = 0x9207
	x.fields["LightSource"] = 0x9208
	x.fields["Flash"] = 0x9209
	x.fields["FocalLength"] = 0x920A
	x.fields["SubjectArea"] = 0x9214
	x.fields["FlashEnergy"] = 0xA20B
	x.fields["SpatialFrequencyResponse"] = 0xA20C
	x.fields["FocalPlaneXResolution"] = 0xA20E
	x.fields["FocalPlaneYResolution"] = 0xA20F
	x.fields["FocalPlaneResolutionUnit"] = 0xA210
	x.fields["SubjectLocation"] = 0xA214
	x.fields["ExposureIndex"] = 0xA215
	x.fields["SensingMethod"] = 0xA217
	x.fields["FileSource"] = 0xA300
	x.fields["SceneType"] = 0xA301
	x.fields["CFAPattern"] = 0xA302
	x.fields["CustomRendered"] = 0xA401
	x.fields["ExposureMode"] = 0xA402
	x.fields["WhiteBalance"] = 0xA403
	x.fields["DigitalZoomRatio"] = 0xA404
	x.fields["FocalLengthIn35mmFilm"] = 0xA405
	x.fields["SceneCaptureType"] = 0xA406
	x.fields["GainControl"] = 0xA407
	x.fields["Contrast"] = 0xA408
	x.fields["Saturation"] = 0xA409
	x.fields["Sharpness"] = 0xA40A
	x.fields["DeviceSettingDescription"] = 0xA40B
	x.fields["SubjectDistanceRange"] = 0xA40C

	/////////////////////////////////////
	//// GPS sub-IFD ////////////////////
	/////////////////////////////////////

	x.gpsFields["GPSVersionID"] = 0x0
	x.gpsFields["GPSLatitudeRef"] = 0x1
	x.gpsFields["GPSLatitude"] = 0x2
	x.gpsFields["GPSLongitudeRef"] = 0x3
	x.gpsFields["GPSLongitude"] = 0x4
	x.gpsFields["GPSAltitudeRef"] = 0x5
	x.gpsFields["GPSAltitude"] = 0x6
	x.gpsFields["GPSTimeStamp"] = 0x7
	x.gpsFields["GPSSatelites"] = 0x8
	x.gpsFields["GPSStatus"] = 0x9
	x.gpsFields["GPSMeasureMode"] = 0xA
	x.gpsFields["GPSDOP"] = 0xB
	x.gpsFields["GPSSpeedRef"] = 0xC
	x.gpsFields["GPSSpeed"] = 0xD
	x.gpsFields["GPSTrackRef"] = 0xE
	x.gpsFields["GPSTrack"] = 0xF
	x.gpsFields["GPSImgDirectionRef"] = 0x10
	x.gpsFields["GPSImgDirection"] = 0x11
	x.gpsFields["GPSMapDatum"] = 0x12
	x.gpsFields["GPSDestLatitudeRef"] = 0x13
	x.gpsFields["GPSDestLatitude"] = 0x14
	x.gpsFields["GPSDestLongitudeRef"] = 0x15
	x.gpsFields["GPSDestLongitude"] = 0x16
	x.gpsFields["GPSDestBearingRef"] = 0x17
	x.gpsFields["GPSDestBearing"] = 0x18
	x.gpsFields["GPSDestDistanceRef"] = 0x19
	x.gpsFields["GPSDestDistance"] = 0x1A
	x.gpsFields["GPSProcessingMethod"] = 0x1B
	x.gpsFields["GPSAreaInformation"] = 0x1C
	x.gpsFields["GPSDateStamp"] = 0x1D
	x.gpsFields["GPSDifferential"] = 0x1E

	/////////////////////////////////////
	//// Interoperability sub-IFD ///////
	/////////////////////////////////////

	x.interOpFields["InteroperabilityIndex"] = 0x1

}

// Get retrieves the exif tag for the given field name. It returns nil if the
// tag name is not found.
func (x *Exif) Get(name string) *tiff.Tag {
	if tg, ok := x.main[x.fields[name]]; ok {
		return tg
	} else if tg, ok := x.gps[x.gpsFields[name]]; ok {
		return tg
	} else if tg, ok := x.interOp[x.interOpFields[name]]; ok {
		return tg
	}
	return nil
}

// String returns a pretty text representation of the decoded exif data.
func (x *Exif) String() string {
	msg := "Main:\n"
	for name, id := range x.fields {
		if tag, ok := x.main[id]; ok {
			msg += name + ":" + tag.String() + "\n"
		}
	}
	msg += "\n\nGPS:\n"
	for name, id := range x.gpsFields {
		if tag, ok := x.gps[id]; ok {
			msg += name + ":" + tag.String() + "\n"
		}
	}
	msg += "\n\nInteroperability:\n"
	for name, id := range x.interOpFields {
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
