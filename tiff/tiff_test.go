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
  //////////// string type //////////////
  tagTest{
    //   {"TgId", "TYPE", "N-VALUES", "OFFSET--", "VAL..."},
    input{"0003", "0002", "00000002", "11000000", ""},
    output{0x0003, 0x0002, 0x0002, []byte{0x11, 0x00}},
  },
  tagTest{
    input{"0001", "0002", "00000006", "00000012", "111213141516"},
    output{0x0001, 0x0002, 0x0006, []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16}},
  },
  //////////// int (1-byte) type ////////////////
  tagTest{
    input{"0001", "0001", "00000001", "11000000", ""},
    output{0x0001, 0x0001, 0x0001, []byte{0x11}},
  },
  tagTest{
    input{"0001", "0001", "00000005", "00000010", "1112131415"},
    output{0x0001, 0x0001, 0x0005, []byte{0x11, 0x12, 0x13, 0x14, 0x15}},
  },
  tagTest{
    input{"0001", "0006", "00000001", "11000000", ""},
    output{0x0001, 0x0006, 0x0001, []byte{0x11}},
  },
  tagTest{
    input{"0001", "0006", "00000005", "00000010", "1112131415"},
    output{0x0001, 0x0006, 0x0005, []byte{0x11, 0x12, 0x13, 0x14, 0x15}},
  },
  //////////// int (2-byte) types ////////////////
  tagTest{
    input{"0001", "0003", "00000002", "11111212", ""},
    output{0x0001, 0x0003, 0x0002, []byte{0x11, 0x11, 0x12, 0x12}},
  },
  tagTest{
    input{"0001", "0003", "00000003", "00000010", "111213141516"},
    output{0x0001, 0x0003, 0x0003, []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16}},
  },
  tagTest{
    input{"0001", "0008", "00000001", "11120000", ""},
    output{0x0001, 0x0008, 0x0001, []byte{0x11, 0x12}},
  },
  tagTest{
    input{"0001", "0008", "00000003", "00000100", "111213141516"},
    output{0x0001, 0x0008, 0x0003, []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16}},
  },
  //////////// int (4-byte) types ////////////////
  tagTest{
    input{"0001", "0004", "00000001", "11121314", ""},
    output{0x0001, 0x0004, 0x0001, []byte{0x11, 0x12, 0x13, 0x14}},
  },
  tagTest{
    input{"0001", "0004", "00000002", "00000010", "1112131415161718"},
    output{0x0001, 0x0004, 0x0002, []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}},
  },
  tagTest{
    input{"0001", "0009", "00000001", "11121314", ""},
    output{0x0001, 0x0009, 0x0001, []byte{0x11, 0x12, 0x13, 0x14}},
  },
  tagTest{
    input{"0001", "0009", "00000002", "00000011", "1112131415161819"},
    output{0x0001, 0x0009, 0x0002, []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x18, 0x19}},
  },
  //////////// rational types ////////////////////
  tagTest{
    input{"0001", "0005", "00000001", "00000010", "1112131415161718"},
    output{0x0001, 0x0005, 0x0001, []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}},
  },
  tagTest{
    input{"0001", "000A", "00000001", "00000011", "1112131415161819"},
    output{0x0001, 0x000A, 0x0001, []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x18, 0x19}},
  },
  //////////// float types ///////////////////////
  tagTest{
    input{"0001", "0005", "00000001", "00000010", "1112131415161718"},
    output{0x0001, 0x0005, 0x0001, []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}},
  },
  tagTest{
    input{"0101", "000A", "00000001", "00000011", "1112131415161819"},
    output{0x0101, 0x000A, 0x0001, []byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x18, 0x19}},
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
    }
    if tg.Fmt != tst.out.format {
      t.Errorf("tag %v format decode: expected %v, got %v", i, tst.out.format, tg.Fmt)
    }
    if tg.Ncomp != tst.out.count {
      t.Errorf("tag %v N-components decode: expected %v, got %v", i, tst.out.count, tg.Ncomp)
    }
    if ! bytes.Equal(tg.Val, tst.out.val) {
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

