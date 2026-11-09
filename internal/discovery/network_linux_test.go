//go:build linux

package discovery

import "testing"

func TestParseProcHexAddr(t *testing.T) {
	addr, ok := parseProcHexAddr("010200C0")
	if !ok || addr.String() != "192.0.2.1" {
		t.Fatalf("got %s ok=%v, want 192.0.2.1", addr, ok)
	}
	if _, ok := parseProcHexAddr("nothex"); ok {
		t.Fatal("expected invalid hex address to be rejected")
	}
}
