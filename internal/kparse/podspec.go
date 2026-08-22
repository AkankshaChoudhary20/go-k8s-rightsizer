package kparse

import "encoding/json"

// Spec is a container's configured resource requests/limits.
// Zero means "not set" (a real, common misconfiguration this tool flags).
type Spec struct {
	Namespace         string
	Pod               string
	Container         string
	RequestedCPUMilli int64
	RequestedMemBytes int64
	LimitCPUMilli     int64
	LimitMemBytes     int64
}

// Minimal shape of `kubectl get pods -o json` -- only the fields this tool needs.
type podList struct {
	Items []struct {
		Metadata struct {
			Namespace string `json:"namespace"`
			Name      string `json:"name"`
		} `json:"metadata"`
		Spec struct {
			Containers []struct {
				Name      string `json:"name"`
				Resources struct {
					Requests map[string]string `json:"requests"`
					Limits   map[string]string `json:"limits"`
				} `json:"resources"`
			} `json:"containers"`
		} `json:"spec"`
	} `json:"items"`
}

// ParsePodSpecs parses `kubectl get pods -o json` output into one Spec per container.
func ParsePodSpecs(data []byte) ([]Spec, error) {
	var pl podList
	if err := json.Unmarshal(data, &pl); err != nil {
		return nil, err
	}

	var specs []Spec
	for _, item := range pl.Items {
		for _, c := range item.Spec.Containers {
			s := Spec{
				Namespace: item.Metadata.Namespace,
				Pod:       item.Metadata.Name,
				Container: c.Name,
			}
			if v, ok := c.Resources.Requests["cpu"]; ok {
				if cpu, err := ParseCPU(v); err == nil {
					s.RequestedCPUMilli = cpu
				}
			}
			if v, ok := c.Resources.Requests["memory"]; ok {
				if mem, err := ParseMemory(v); err == nil {
					s.RequestedMemBytes = mem
				}
			}
			if v, ok := c.Resources.Limits["cpu"]; ok {
				if cpu, err := ParseCPU(v); err == nil {
					s.LimitCPUMilli = cpu
				}
			}
			if v, ok := c.Resources.Limits["memory"]; ok {
				if mem, err := ParseMemory(v); err == nil {
					s.LimitMemBytes = mem
				}
			}
			specs = append(specs, s)
		}
	}
	return specs, nil
}
