// Package kparse parses the resource-quantity formats Kubernetes tooling
// actually emits (kubectl top, kubectl get -o json), without depending on
// k8s.io/apimachinery.
package kparse

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ParseCPU parses a Kubernetes CPU quantity ("100m", "0.5", "2") into millicores.
func ParseCPU(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty CPU quantity")
	}
	if strings.HasSuffix(s, "m") {
		v, err := strconv.ParseInt(strings.TrimSuffix(s, "m"), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid millicore CPU quantity %q: %w", s, err)
		}
		if v < 0 {
			return 0, fmt.Errorf("negative CPU quantity %q", s)
		}
		return v, nil
	}
	// Whole/fractional cores, e.g. "2" or "0.5" -> multiply by 1000.
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid CPU quantity %q: %w", s, err)
	}
	return checkedQuantity(f*1000, s)
}

var memSuffixes = map[string]int64{
	"Ki": 1024,
	"Mi": 1024 * 1024,
	"Gi": 1024 * 1024 * 1024,
	"Ti": 1024 * 1024 * 1024 * 1024,
	"K":  1000,
	"M":  1000 * 1000,
	"G":  1000 * 1000 * 1000,
	"T":  1000 * 1000 * 1000 * 1000,
}

// ParseMemory parses a Kubernetes memory quantity ("128Mi", "1Gi", "500000")
// into bytes.
func ParseMemory(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty memory quantity")
	}

	for _, suffix := range []string{"Ki", "Mi", "Gi", "Ti", "K", "M", "G", "T"} {
		if strings.HasSuffix(s, suffix) {
			numStr := strings.TrimSuffix(s, suffix)
			v, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid memory quantity %q: %w", s, err)
			}
			return checkedQuantity(v*float64(memSuffixes[suffix]), s)
		}
	}

	// No suffix -- raw byte count.
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory quantity %q: %w", s, err)
	}
	if v < 0 {
		return 0, fmt.Errorf("negative memory quantity %q", s)
	}
	return v, nil
}

// Float-to-int conversion must not turn malformed or overflowing input into
// negative resource recommendations. float64(MaxInt64) rounds up to 2^63.
func checkedQuantity(value float64, original string) (int64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value >= math.Exp2(63) {
		return 0, fmt.Errorf("resource quantity %q is outside the nonnegative int64 range", original)
	}
	return int64(math.Ceil(value)), nil
}
