
package exif_test

import (
  "os"
  "log"
  "fmt"
  "github.com/rwcarlsen/goexif/exif"
)

func ExampleDecode() {
  fname := "sample1.jpg"

  f, err := os.Open(fname)
  if err != nil {
    log.Fatal(err)
  }

  x, err := exif.Decode(f)
  if err != nil {
    log.Fatal(err)
  }

  camMake := x.Get("Make").StringVal()
  camModel := x.Get("Model").StringVal()
  date := x.Get("DateTimeOriginal").StringVal()
  numer, denom := x.Get("FocalLength").Rat2(0) // retrieve first (only) rat. value

  fmt.Println(camMake, camModel, date, numer, denom)
}

