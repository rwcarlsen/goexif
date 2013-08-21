package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/rwcarlsen/goexif/tiff"
)

var mnote = flag.Bool("mknote", false, "try to parse makernote data")

func main() {
	flag.Parse()
	fnames := flag.Args()

	if *mnote {
		exif.RegisterParsers(mknote.All...)
	}

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

type Walker struct{}

func (_ Walker) Walk(name exif.FieldName, tag *tiff.Tag) error {
	data, _ := tag.MarshalJSON()
	fmt.Printf("    %v: %v\n", name, string(data))
	return nil
}
