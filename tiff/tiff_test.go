package tiff

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"os"
	"testing"
)

type input struct {
  tgId string
  tpe string
  nVals string
  offset string
  val string
}

type output struct {
  id uint16
  format uint16
  count uint32
  val []byte
}

type tagTest struct {
  in input
  out output
}

///////////////////////////////////////////////
//// Big endian Tests /////////////////////////
///////////////////////////////////////////////

var bigEndSet = []tagTest{
  tagTest{
    //   {"TgId", "TYPE", "N-VALUES", "OFFSET--", "VAL..."},
    input{"0001", "0001", "00000001", "11000000", ""},
    output{0x0001, 0x0001, 0x0001, []byte{0x11}},
  },
  tagTest{
    //   {"TgId", "TYPE", "N-VALUES", "OFFSET--", "VAL..."},
    input{"0001", "0001", "00000002", "11120000", ""},
    output{0x0001, 0x0001, 0x0002, []byte{0x11, 0x12}},
  },
  tagTest{
    //   {"TgId", "TYPE", "N-VALUES", "OFFSET--", "VAL..."},
    input{"0001", "0001", "00000005", "00000010", "1112131415"},
    output{0x0001, 0x0001, 0x0005, []byte{0x11, 0x12, 0x13, 0x14, 0x15}},
  },
}

func TestDecodeTag_bigendian(t *testing.T) {
  for i, tst := range bigEndSet {
    data := buildInput(tst.in, binary.BigEndian)
    buf := bytes.NewReader(data)

    tg, err := DecodeTag(buf, binary.BigEndian)
    if err != nil {
      t.Errorf("tag %v%+v decode failed: %v", i, tst.in, err)
      continue
    }

    if tg.Id != tst.out.id {
      t.Errorf("tag %v id decode: expected %v, got %v", i, tst.out.id, tg.Id)
    } else if tg.Fmt != tst.out.format {
      t.Errorf("tag %v format decode: expected %v, got %v", i, tst.out.format, tg.Fmt)
    } else if tg.Ncomp != tst.out.count {
      t.Errorf("tag %v N-components decode: expected %v, got %v", i, tst.out.count, tg.Ncomp)
    } else if ! bytes.Equal(tg.Val, tst.out.val) {
      t.Errorf("tag %v value decode: expected %v, got %v", i, tst.out.val, tg.Val)
    }
  }
}

// buildInputBig creates a byte-slice based on big-endian ordered input
func buildInput(in input, order binary.ByteOrder) []byte {
  data := make([]byte, 0)
  d, _ := hex.DecodeString(in.tgId)
  data = append(data, d...)
  d, _ = hex.DecodeString(in.tpe)
  data = append(data, d...)
  d, _ = hex.DecodeString(in.nVals)
  data = append(data, d...)
  d, _ = hex.DecodeString(in.offset)
  data = append(data, d...)

  if in.val != "" {
    off := order.Uint32(d)
    for i := 0; i < int(off) - 12; i++ {
      data = append(data, 0xFF)
    }

    d, _ = hex.DecodeString(in.val)
    data = append(data, d...)
  }

  return data
}

func data() []byte {
	s1 := "49492A000800000002001A0105000100"
	s1 += "00002600000069870400010000001102"
	s1 += "0000000000004800000001000000"

	dat, err := hex.DecodeString(s1)
	if err != nil {
		panic("invalid string fixture")
	}
	return dat
}

func TestDecode(t *testing.T) {
	name := "sample1.tif"
	f, err := os.Open(name)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	tif, err := Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(tif)
}

func TestDecodeTag_blob(t *testing.T) {
	buf := bytes.NewReader(data())
	buf.Seek(10, 1)
	tg, err := DecodeTag(buf, binary.LittleEndian)
	if err != nil {
		t.Fatalf("tag decode failed: %v", err)
	}

	t.Logf("tag: %v+\n", tg)
	n, d := tg.Rat2(0)
	t.Logf("tag rat val: %v\n", n, d)
}
