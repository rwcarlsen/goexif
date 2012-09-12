package tiff

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"os"
	"testing"
)

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
