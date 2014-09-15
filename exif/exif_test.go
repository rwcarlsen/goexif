package exif

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rwcarlsen/goexif/tiff"
)

// switch to true to regenerate regression expected values
var regenRegress = false

var dataDir = flag.String("test_data_dir", ".", "Directory where the data files for testing are located")

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
	if strings.HasPrefix(string(name), UnknownPrefix) {
		fmt.Fprintf(w.wr, "\t\t\"%v\": `%v`,\n", name, tag.String())
	} else {
		fmt.Fprintf(w.wr, "\t\t%v: `%v`,\n", name, tag.String())
	}
	return nil
}

func TestDecode(t *testing.T) {
	fpath := filepath.Join(*dataDir, "samples")
	f, err := os.Open(fpath)
	if err != nil {
		t.Fatalf("Could not open sample directory '%s': %v", fpath, err)
	}

	names, err := f.Readdirnames(0)
	if err != nil {
		t.Fatalf("Could not read sample directory '%s': %v", fpath, err)
	}

	cnt := 0
	for _, name := range names {
		if !strings.HasSuffix(name, ".jpg") {
			t.Logf("skipping non .jpg file %v", name)
			continue
		}
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

		t.Logf("checking pic %v", name)
		x.Walk(&walker{name, t})
		cnt++
	}
	if cnt != len(regressExpected) {
		t.Errorf("Did not process enough samples, got %d, want %d", cnt, len(regressExpected))
	}
}

type walker struct {
	picName string
	t       *testing.T
}

func (w *walker) Walk(field FieldName, tag *tiff.Tag) error {
	// this needs to be commented out when regenerating regress expected vals
	pic := regressExpected[w.picName]
	if pic == nil {
		w.t.Errorf("   regression data not found")
		return nil
	}

	exp, ok := pic[field]
	if !ok {
		w.t.Errorf("   regression data does not have field %v", field)
		return nil
	}

	s := tag.String()
	if tag.Count == 1 && s != "\"\"" {
		s = fmt.Sprintf("[%s]", s)
	}
	got := tag.String()

	if exp != got {
		fmt.Println("s: ", s)
		fmt.Printf("len(s)=%v\n", len(s))
		w.t.Errorf("   field %v bad tag: expected '%s', got '%s'", field, exp, got)
	}
	return nil
}

func TestMarshal(t *testing.T) {
	name := filepath.Join(*dataDir, "sample1.jpg")
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

func testSingleParseDegreesString(t *testing.T, s string, w float64) {
	g, err := parseTagDegreesString(s)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(w-g) > 1e-10 {
		t.Errorf("Wrong parsing result %s: Want %.12f, got %.12f", s, w, g)
	}
}

func TestParseTagDegreesString(t *testing.T) {
	// semicolon as decimal mark
	testSingleParseDegreesString(t, "52,00000,50,00000,34,01180", 52.842781055556) // comma as separator
	testSingleParseDegreesString(t, "52,00000;50,00000;34,01180", 52.842781055556) // semicolon as separator

	// point as decimal mark
	testSingleParseDegreesString(t, "14.00000,44.00000,34.01180", 14.742781055556) // comma as separator
	testSingleParseDegreesString(t, "14.00000;44.00000;34.01180", 14.742781055556) // semicolon as separator
	testSingleParseDegreesString(t, "14.00000;44.00000,34.01180", 14.742781055556) // mixed separators

	testSingleParseDegreesString(t, "-008.0,30.0,03.6", -8.501) // leading zeros

	// no decimal places
	testSingleParseDegreesString(t, "-10,15,54", -10.265)
	testSingleParseDegreesString(t, "-10;15;54", -10.265)

	// incorrect mix of comma and point as decimal mark
	s := "-17,00000,15.00000,04.80000"
	if _, err := parseTagDegreesString(s); err == nil {
		t.Error("parseTagDegreesString: false positive for " + s)
	}
}
