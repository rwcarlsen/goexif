package tiff

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"os"
	"testing"
)

//   {"TgId", "TYPE", "N-VALUES", "VALUE---"},
type input [4]string

type output struct {
  id uint16
  format uint16
  count uint32
  offset uint32
  val []byte
}

type tagTest struct {
  in input
  out output
}

///////////////////////////////////////////////
//// Big endian Tests /////////////////////////
///////////////////////////////////////////////

var set1 = []tagTest{
  tagTest{
    //   {"TgId", "TYPE", "N-VALUES", "VALUE---"},
    input{"0001", "0001", "00000001", "11000000"},
    output{0x0001, 0x0001, 0x0001, 0x11},
  }
  tagTest{
    //   {"TgId", "TYPE", "N-VALUES", "VALUE---"},
    input{"0001", "0001", "00000002", "11120000"},
    output{0x0001, 0x0001, 0x0001, 0000},
  }

  input{"0002", "0002", "00000002", "61000000"},
  input{"0002", "0002", "00000003", "61000000"},
  input{"0002", "0002", "00000004", "61000000"},
  input{"0002", "0002", "00000005", "61000000"},

  input{"0003", "0003", "00000001", "00000000"},
  input{"0004", "0004", "00000001", "00000000"},
  input{"0005", "0005", "00000001", "00000000"},
  input{"0006", "0006", "00000001", "00000000"},
  input{"0007", "0007", "00000001", "00000000"},
  input{"0008", "0008", "00000001", "00000000"},
  input{"0009", "0009", "00000001", "00000000"},
  input{"000A", "000A", "00000001", "00000000"},
  input{"000B", "000B", "00000001", "00000000"},
  input{"000C", "000C", "00000001", "00000000"},
}

var littleEndianEnts = []entry{
  //   {"TgId", "TYPE", "N-VALUES", "VALUE---"},
  entry{"0101", "0100", "00001111", "00001111"},
  entry{"0201", "0200", "00001111", "00001111"},
  entry{"0301", "0300", "00001111", "00001111"},
  entry{"0401", "0400", "00001111", "00001111"},
  entry{"0501", "0500", "00001111", "00001111"},
  entry{"0601", "0600", "00001111", "00001111"},
  entry{"0701", "0700", "00001111", "00001111"},
  entry{"0801", "0800", "00001111", "00001111"},
  entry{"0901", "0900", "00001111", "00001111"},
  entry{"0A01", "0A00", "00001111", "00001111"},
  entry{"0B01", "0B00", "00001111", "00001111"},
  entry{"0C01", "0C00", "00001111", "00001111"},
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

func TestDecodeTag(t *testing.T) {
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
