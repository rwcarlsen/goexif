
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

