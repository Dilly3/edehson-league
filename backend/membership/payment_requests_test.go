package membership

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReceiptKeyFilesUnderPlayerName(t *testing.T) {
	got := receiptKey("Chukwuemeka Okafor", "abc", "2026-11", "image/jpeg")
	if got != "chukwuemeka-okafor/2026-11/abc.jpg" {
		t.Fatalf("got %q", got)
	}
	if k := receiptKey("../../etc", "x", "2026-11", "application/pdf"); strings.Contains(k, "..") {
		t.Fatalf("key must not contain path traversal: %q", k)
	}
}

func TestParseKobo(t *testing.T) {
	for in, want := range map[string]int64{"5000": 500000, "1,500.5": 150050, "0.01": 1} {
		if got, ok := parseKobo(in); !ok || got != want {
			t.Errorf("parseKobo(%q) = %d, %v; want %d", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "0", "-5", "1.234", "abc"} {
		if _, ok := parseKobo(bad); ok {
			t.Errorf("parseKobo(%q) should fail", bad)
		}
	}
}

func TestLocalReceiptStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := localStore{dir: dir}
	ctx := context.Background()
	if err := s.Put(ctx, "ada/2026-11/r1.pdf", "application/pdf", []byte("%PDF-1.4")); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ctx, "ada/2026-11/r1.pdf")
	if err != nil || string(got) != "%PDF-1.4" {
		t.Fatalf("round trip: %q %v", got, err)
	}
	if err := s.Put(ctx, "../escape.pdf", "application/pdf", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "escape.pdf")); err == nil {
		t.Fatal("local store wrote outside its directory")
	}
	if err := s.Delete(ctx, "ada/2026-11/r1.pdf"); err != nil {
		t.Fatal(err)
	}
}
