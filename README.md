goexif
======

Provides decoding of basic exif and tiff encoded data. Still in alpha - no garuntees.
Suggestions/pull requests welcome.  Funcionality is split into two packages - "exif" and "tiff"
The exif package depends on the tiff package. 
Documentation can be found at http://go.pkgdoc.org/github.com/rwcarlsen/goexif

To install, in a terminal type:

```
go get github.com/rwcarlsen/goexif/exif
```

Or if you just want the tiff package:

```
go get github.com/rwcarlsen/goexif/tiff
```

Example usage:

```go
package main

import (
  "github.com/rwcarlsen/goexif/exif"
  "os"
  "log"
  "fmt"
)

func main() {
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
```

<!--golang-->