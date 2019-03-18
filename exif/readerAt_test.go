package exif

import (
	"bytes"
	"testing"
)

func TestReadToReaderAt(t *testing.T) {
	b := []byte{0, 1, 2, 3, 4, 5}
	readerAt := &readerToReaderAt{
		reader: bytes.NewReader(b),
	}

	t.Run("reading from begining", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			test := make([]byte, 2)
			// check the idempotence reading from reader or from cache
			count, err := readerAt.ReadAt(test, 0)
			if err != nil {
				t.Errorf("Unexpected read error for while reading at offset 0: %v", err)
			}
			if count != 2 {
				t.Errorf("Unexpected count returned for while reading at offset 0: %v", count)
			}
			if bytes.Compare(test, []byte{0, 1}) != 0 {
				t.Errorf("unexpected read at offset 0, expecting to read [0,1], got %v", test)
			}
		}
	})
	t.Run("reading skipping bytes", func(t *testing.T) {
		test := make([]byte, 2)
		count, err := readerAt.ReadAt(test, 2)
		if err != nil {
			t.Errorf("Unexpected read error for while reading at offset 2: %v", err)
		}
		if count != 2 {
			t.Errorf("Unexpected count returned for while reading at offset 2: %v", count)
		}
		if bytes.Compare(test, []byte{2, 3}) != 0 {
			t.Errorf("unexpected read at offset 2, expecting to read [0,1], got %v", test)
		}
	})
	t.Run("reading larger than available", func(t *testing.T) {
		test := make([]byte, 7)
		count, err := readerAt.ReadAt(test, 0)
		if err != nil {
			t.Errorf("Unexpected read error for while reading at offset 2: %v", err)
		}
		if count != 6 {
			t.Errorf("Unexpected count returned for while reading at offset 2: %v", count)
		}
		if bytes.Compare(test, append(b, 0)) != 0 {
			t.Errorf("unexpected read at offset 2, expecting to read [0,1], got %v", test)
		}
	})
}
