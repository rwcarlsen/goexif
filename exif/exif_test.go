package exif

import (
	"fmt"
	"os"
	"testing"
	"path/filepath"
	"io"
	"strings"

	"github.com/rwcarlsen/goexif/tiff"
)

// switch to true to regenerate regression expected values
var regenRegress = false

// TestRegenRegress regenerates the expected image exif fields/values for
// sample images.
func TestRegenRegress(t *testing.T) {
	if !regenRegress {
		return
	}

	dst, err := os.Create("regress_expected_test.go")
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()

	dir, err := os.Open("samples")
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()

	names, err := dir.Readdirnames(0)
	if err != nil {
		t.Fatal(err)
	}
	for i, name := range names {
		names[i] = filepath.Join("samples", name)
	}
	makeExpected(names, dst)
}

func makeExpected(files []string, w io.Writer) {
	fmt.Fprintf(w, "package exif\n\n")
	fmt.Fprintf(w, "var regressExpected = map[string]map[FieldName]string{\n")

	for _, name := range files {
		f, err := os.Open(name)
		if err != nil {
			continue
		}

		x, err := Decode(f)
		if err != nil {
			f.Close()
			continue
		}

		fmt.Fprintf(w, "\t\"%v\": map[FieldName]string{\n", filepath.Base(name))
		x.Walk(&regresswalk{w})
		fmt.Fprintf(w, "\t},\n")
		f.Close()
	}
	fmt.Fprintf(w, "}\n")
}

type regresswalk struct {
	wr io.Writer
}

func (w *regresswalk) Walk(name FieldName, tag *tiff.Tag) error {
	if strings.HasPrefix(string(name), unknownPrefix) {
		fmt.Fprintf(w.wr, "\t\t\"%v\": `%v`,\n", name, tag.String())
	} else {
		fmt.Fprintf(w.wr, "\t\t%v: `%v`,\n", name, tag.String())
	}
	return nil
}

func TestDecode(t *testing.T) {
	fpath := "samples"
	f, err := os.Open(fpath)
	if err != nil {
		t.Fatalf("Could not open sample directory: %v", err)
	}

	names, err := f.Readdirnames(0)
	if err != nil {
		t.Fatalf("Could not read sample directory: %v", err)
	}

	for _, name := range names {
		t.Logf("testing file %v", name)
		f, err := os.Open(filepath.Join(fpath, name))
		if err != nil {
			t.Fatal(err)
		}

		x, err := Decode(f)
		if err != nil {
			t.Fatal(err)
		} else if x == nil {
			t.Fatalf("No error and yet %v was not decoded", name)
		}

		x.Walk(&walker{name, t})
	}
}

type walker struct {
	picName string
	t *testing.T
}

func (w *walker) Walk(field FieldName, tag *tiff.Tag) error {
	// this needs to be commented out when regenerating regress expected vals
	if v := regressExpected[w.picName][field]; v != tag.String() {
		w.t.Errorf("pic %v:  expected '%v' got '%v'", w.picName, v, tag.String())
	}
	return nil
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
