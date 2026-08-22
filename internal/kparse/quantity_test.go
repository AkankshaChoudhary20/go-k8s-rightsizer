package kparse

import "testing"

func TestParseCPU(t *testing.T) {
	cases := map[string]int64{
		"100m": 100,
		"1":    1000,
		"0.5":  500,
		"2":    2000,
		"5m":   5,
	}
	for input, want := range cases {
		got, err := ParseCPU(input)
		if err != nil {
			t.Errorf("ParseCPU(%q) unexpected error: %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("ParseCPU(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestParseCPU_Invalid(t *testing.T) {
	for _, input := range []string{"", "abc", "1x2"} {
		if _, err := ParseCPU(input); err == nil {
			t.Errorf("ParseCPU(%q) expected error, got nil", input)
		}
	}
}

func TestParseMemory(t *testing.T) {
	cases := map[string]int64{
		"128Mi": 128 * 1024 * 1024,
		"1Gi":   1024 * 1024 * 1024,
		"500Ki": 500 * 1024,
		"1000":  1000,
		"2G":    2 * 1000 * 1000 * 1000,
	}
	for input, want := range cases {
		got, err := ParseMemory(input)
		if err != nil {
			t.Errorf("ParseMemory(%q) unexpected error: %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("ParseMemory(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestParseMemory_Invalid(t *testing.T) {
	for _, input := range []string{"", "abcMi"} {
		if _, err := ParseMemory(input); err == nil {
			t.Errorf("ParseMemory(%q) expected error, got nil", input)
		}
	}
}
