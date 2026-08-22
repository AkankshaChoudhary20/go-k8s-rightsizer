package kparse

import "testing"

// A realistic (trimmed) `kubectl get pods -o json` PodList, including a
// container with no resources set at all -- the common misconfiguration
// this whole tool exists to catch.
const samplePodListJSON = `{
  "apiVersion": "v1",
  "kind": "PodList",
  "items": [
    {
      "metadata": { "namespace": "default", "name": "web-6f8d9b-abcde" },
      "spec": {
        "containers": [
          {
            "name": "web",
            "resources": {
              "requests": { "cpu": "500m", "memory": "256Mi" },
              "limits": { "cpu": "1", "memory": "512Mi" }
            }
          },
          {
            "name": "sidecar",
            "resources": {}
          }
        ]
      }
    },
    {
      "metadata": { "namespace": "kube-system", "name": "coredns-5d78c-xyz" },
      "spec": {
        "containers": [
          {
            "name": "coredns",
            "resources": {
              "requests": { "cpu": "100m", "memory": "70Mi" }
            }
          }
        ]
      }
    }
  ]
}`

func TestParsePodSpecs_RealisticPodList(t *testing.T) {
	specs, err := ParsePodSpecs([]byte(samplePodListJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 3 {
		t.Fatalf("expected 3 container specs, got %d", len(specs))
	}

	web := specs[0]
	if web.Namespace != "default" || web.Pod != "web-6f8d9b-abcde" || web.Container != "web" {
		t.Errorf("unexpected identity: %+v", web)
	}
	if web.RequestedCPUMilli != 500 || web.RequestedMemBytes != 256*1024*1024 {
		t.Errorf("unexpected requests: %+v", web)
	}
	if web.LimitCPUMilli != 1000 || web.LimitMemBytes != 512*1024*1024 {
		t.Errorf("unexpected limits: %+v", web)
	}

	sidecar := specs[1]
	if sidecar.RequestedCPUMilli != 0 || sidecar.RequestedMemBytes != 0 {
		t.Errorf("sidecar has no resources block, expected all-zero spec, got %+v", sidecar)
	}

	coredns := specs[2]
	if coredns.RequestedCPUMilli != 100 || coredns.RequestedMemBytes != 70*1024*1024 {
		t.Errorf("unexpected coredns requests: %+v", coredns)
	}
	if coredns.LimitCPUMilli != 0 {
		t.Errorf("coredns has no limits set, expected 0, got %d", coredns.LimitCPUMilli)
	}
}

func TestParsePodSpecs_InvalidJSON(t *testing.T) {
	if _, err := ParsePodSpecs([]byte("not json")); err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
}
