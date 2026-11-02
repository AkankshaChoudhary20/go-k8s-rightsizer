package kparse

import "testing"

func TestMemoryLowercaseK(t *testing.T) {
	for input, want := range map[string]int64{"1k": 1000, "1.5k": 1500, "0.0001k": 1} {
		got, err := ParseMemory(input)
		if err != nil || got != want {
			t.Errorf("ParseMemory(%q) = %d, %v; want %d", input, got, err, want)
		}
	}
	if _, err := ParseMemory("-1k"); err == nil {
		t.Fatal("negative quantity accepted")
	}
}
