
package main

import (
	"os"
	"fmt"
	"log"
	"flag"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
)

func main() {
	flag.Parse()
	fnames := flag.Args()

	for _, name := range fnames {
		f, err := os.Open(name)
		if err != nil {
			log.Printf("err on %v: %v", name, err)
			continue
		}

		x, err := exif.Decode(f)
		if err != nil {
			log.Printf("err on %v: %v", name, err)
			continue
		}

		fmt.Printf("\n---- Image '%v' ----\n", name)
		x.Walk(Walker{})
	}
}

type Walker struct {}

func (_ Walker) Walk(name exif.FieldName, tag *tiff.Tag) error {
	data, _ := tag.MarshalJSON()
	fmt.Printf("    %v: %v\n", name, string(data))
	return nil
}
