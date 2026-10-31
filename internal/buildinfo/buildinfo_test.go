package buildinfo

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestPrint(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	Print("1.2.3", "2026-10-28", "deadbeef")
	_ = w.Close()
	os.Stdout = stdout

	got, err := io.ReadAll(r)
	_ = r.Close()
	if err != nil {
		t.Fatal(err)
	}
	want := "Build version: 1.2.3\nBuild date: 2026-10-28\nBuild commit: deadbeef\n"
	if !bytes.Equal(got, []byte(want)) {
		t.Fatalf("got %q, want %q", got, want)
	}
}
