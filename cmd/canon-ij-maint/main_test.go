package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestOutputFlagMatchesGolden(t *testing.T) {
	dir := t.TempDir()
	cases := map[string][]string{
		"clean_deep.bin":  {"--output", "", "clean", "--deep"},
		"nozzlecheck.bin": {"nozzle-check", "--output", ""},
	}
	for golden, args := range cases {
		out := filepath.Join(dir, golden)
		for i, a := range args {
			if a == "" {
				args[i] = out
			}
		}
		var so, se bytes.Buffer
		if code := run(args, &so, &se); code != 0 {
			t.Fatalf("%v: exit %d: %s", args, code, se.String())
		}
		got, _ := os.ReadFile(out)
		want, _ := os.ReadFile(filepath.Join("..", "..", "testdata", golden))
		if !bytes.Equal(got, want) {
			t.Errorf("%s: bytes differ", golden)
		}
	}
}

func TestBadGroup(t *testing.T) {
	var so, se bytes.Buffer
	if code := run([]string{"--output", "-", "clean", "--group", "cyan"}, &so, &se); code != 2 {
		t.Errorf("want exit 2, got %d", code)
	}
}
