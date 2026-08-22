package kparse

import (
	"testing"
	"time"
)

func TestParseTopContainers_RealisticOutput(t *testing.T) {
	// Realistic `kubectl top pods --all-namespaces --containers --no-headers` output.
	output := `default      web-6f8d9b-abcde     web           250m    180Mi
default      web-6f8d9b-abcde     sidecar       5m      20Mi
kube-system  coredns-5d78c-xyz    coredns       3m      15Mi
`
	at := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	samples, err := ParseTopContainers(output, at)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(samples) != 3 {
		t.Fatalf("expected 3 samples, got %d", len(samples))
	}

	first := samples[0]
	if first.Namespace != "default" || first.Pod != "web-6f8d9b-abcde" || first.Container != "web" {
		t.Errorf("unexpected identity fields: %+v", first)
	}
	if first.CPUMilli != 250 {
		t.Errorf("expected CPU 250m, got %d", first.CPUMilli)
	}
	if first.MemBytes != 180*1024*1024 {
		t.Errorf("expected mem 180Mi in bytes, got %d", first.MemBytes)
	}
}

func TestParseTopContainers_IgnoresBlankLines(t *testing.T) {
	output := "default pod1 c1 10m 10Mi\n\n\ndefault pod2 c2 20m 20Mi\n"
	samples, err := ParseTopContainers(output, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(samples) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(samples))
	}
}

func TestParseTopContainers_RejectsMalformedLine(t *testing.T) {
	output := "default pod1 c1 10m\n" // missing memory column
	if _, err := ParseTopContainers(output, time.Now()); err == nil {
		t.Fatal("expected an error for a malformed line, got nil")
	}
}
