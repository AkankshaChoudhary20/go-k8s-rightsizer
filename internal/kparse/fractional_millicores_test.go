package kparse

import "testing"

func TestFractionalMillicoresRoundUp(t *testing.T) {
	for input, want := range map[string]int64{"0.1m": 1, "1.5m": 2, "2m": 2, "0m": 0, "9223372036854775807m": 9223372036854775807} {
		got, err := ParseCPU(input)
		if err != nil || got != want {
			t.Errorf("%s: got %d, %v; want %d", input, got, err, want)
		}
	}
	for _, input := range []string{"NaNm", "Infm", "-0.5m"} {
		if _, err := ParseCPU(input); err == nil {
			t.Errorf("accepted %s", input)
		}
	}
}
