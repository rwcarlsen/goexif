package exif

import (
	"os"
	"testing"
)

func TestDecode(t *testing.T) {
	name := "sample1.jpg"
	f, err := os.Open(name)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	x, err := Decode(f)
	if err != nil {
		t.Error(err)
	}
	if x == nil {
		t.Fatal("bad err")
	}

	t.Logf("Model: %v", x.Get("Model").StringVal())
	t.Log(x)
}

func TestIter(t *testing.T) {
	name := "sample1.jpg"
	f, err := os.Open(name)
	if err != nil {
		t.Fatalf("%v\n", err)
	}

	x, err := Decode(f)
	if err != nil {
		t.Error(err)
	}
	if x == nil {
		t.Fatal("bad err")
	}

  it := x.Iter()
  for name, tag := it(); tag != nil; name, tag = it() {
    t.Logf("%v: %v", name, tag)
  }
}

func TestMarshal(t *testing.T) {
	name := "sample1.jpg"
	f, err := os.Open(name)
	if err != nil {
		t.Fatalf("%v\n", err)
	}
	defer f.Close()

	x, err := Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if x == nil {
		t.Fatal("bad err")
	}

	b, err := x.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s", b)
}
