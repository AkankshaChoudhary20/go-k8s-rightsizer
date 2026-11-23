package kparse

import (
	"fmt"
	"strings"
	"time"
)

// Sample is one observed usage reading for a single container.
type Sample struct {
	Timestamp time.Time
	Namespace string
	Pod       string
	Container string
	CPUMilli  int64
	MemBytes  int64
}

// ParseTopContainers parses the output of
// `kubectl top pods --all-namespaces --containers --no-headers`, whose
// columns are: NAMESPACE POD NAME CPU(cores) MEMORY(bytes).
func ParseTopContainers(output string, at time.Time) ([]Sample, error) {
	var samples []Sample
	for index, rawLine := range strings.Split(output, "\n") {
		lineNum := index + 1
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 5 {
			return nil, fmt.Errorf("line %d: expected 5 fields (namespace pod container cpu mem), got %d: %q", lineNum, len(fields), line)
		}

		cpu, err := ParseCPU(fields[3])
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		mem, err := ParseMemory(fields[4])
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		samples = append(samples, Sample{
			Timestamp: at,
			Namespace: fields[0],
			Pod:       fields[1],
			Container: fields[2],
			CPUMilli:  cpu,
			MemBytes:  mem,
		})
	}
	return samples, nil
}
