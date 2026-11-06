package canonij

import (
	"bytes"
	"os"
	"reflect"
	"testing"
)

// The files in testdata are the exact bytes the original tool sent to a G3010 and that fixed it.
func TestBytesMatchKnownGood(t *testing.T) {
	cases := []struct {
		file  string
		kind  Kind
		group Group
	}{
		{"clean.bin", Regular, All},
		{"clean_deep.bin", Deep, All},
		{"clean_group_black.bin", Regular, Black},
		{"clean_deep_group_color.bin", Deep, Color},
	}
	for _, c := range cases {
		want, err := os.ReadFile("testdata/" + c.file)
		if err != nil {
			t.Fatal(err)
		}
		got, err := Cleaning(c.kind, c.group)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: bytes differ (got %d, want %d)", c.file, len(got), len(want))
		}
	}
	want, _ := os.ReadFile("testdata/nozzlecheck.bin")
	if got := NozzleCheck(); !bytes.Equal(got, want) {
		t.Errorf("nozzle check: bytes differ (got %d, want %d)", len(got), len(want))
	}
}

func TestWrapPadsWithNULs(t *testing.T) {
	start := startJob(DefaultJobID)
	b := Wrap("OP", DefaultJobID)
	pad := b[len(start) : 2*len(start)]
	if !bytes.Equal(pad, make([]byte, len(start))) {
		t.Fatal("expected NUL padding as long as StartJob")
	}
}

func TestCleaningRejectsBadInput(t *testing.T) {
	if _, err := Cleaning("hard", All); err == nil {
		t.Error("want error for bad kind")
	}
	if _, err := Cleaning(Deep, "cyan"); err == nil {
		t.Error("want error for bad group")
	}
}

func TestParseQueues(t *testing.T) {
	out := "device for Canon_G3010: usb://Canon/G3010%20series?serial=92B81B&interface=1\n" +
		"device for HP: ipp://hp.local/ipp/print\n" +
		"device for Office: usb://CANON/MX490\n"
	got := ParseQueues(out)
	want := []string{"Canon_G3010", "Office"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
