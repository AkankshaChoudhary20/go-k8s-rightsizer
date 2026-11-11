package kparse

import "testing"

func TestCPUFractionalUnits(t *testing.T) {
	for input, want := range map[string]int64{"1n": 1, "1000000n": 1, "1500000n": 2, "1u": 1, "1500u": 2} {
		got, err := ParseCPU(input)
		if err != nil || got != want {
			t.Errorf("ParseCPU(%q) = %d, %v; want %d", input, got, err, want)
		}
	}
	for _, input := range []string{"-1n", "NaNu", "Infn", "1e400u"} {
		if _, err := ParseCPU(input); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
}
