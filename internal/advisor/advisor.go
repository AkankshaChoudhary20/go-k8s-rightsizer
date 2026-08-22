// Package advisor turns observed container resource usage plus configured
// requests/limits into right-sizing recommendations.
package advisor

import (
	"sort"

	"go-k8s-rightsizer/internal/kparse"
)

type Verdict string

const (
	VerdictNoData           Verdict = "no_data"
	VerdictMissingRequests  Verdict = "missing_requests"
	VerdictOverProvisioned  Verdict = "over_provisioned"
	VerdictUnderProvisioned Verdict = "under_provisioned"
	VerdictApproachingLimit Verdict = "approaching_limit"
	VerdictRightSized       Verdict = "right_sized"
)

// overProvisionedFactor: flag as over-provisioned if requested exceeds
// observed p95 usage by more than this multiple.
const overProvisionedFactor = 2.0

// headroomFactor: suggested request = observed p95 usage * this factor.
const headroomFactor = 1.2

// limitApproachFactor: flag as approaching its limit if p95 usage exceeds
// this fraction of the configured limit.
const limitApproachFactor = 0.9

type Recommendation struct {
	Namespace           string
	Pod                 string
	Container           string
	Verdict             Verdict
	Reason              string
	RequestedCPUMilli   int64
	RequestedMemBytes   int64
	ObservedP95CPUMilli int64
	ObservedP95MemBytes int64
	SuggestedCPUMilli   int64
	SuggestedMemBytes   int64
}

// Percentile returns the p-th percentile (0-100) of values using
// nearest-rank interpolation. values need not be pre-sorted; it is not
// mutated.
func Percentile(values []int64, p float64) int64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	rank := int(float64(len(sorted)-1) * p / 100.0)
	return sorted[rank]
}

// Analyze compares configured spec against observed samples for one
// container and returns a right-sizing recommendation.
func Analyze(spec kparse.Spec, samples []kparse.Sample) Recommendation {
	rec := Recommendation{
		Namespace:         spec.Namespace,
		Pod:               spec.Pod,
		Container:         spec.Container,
		RequestedCPUMilli: spec.RequestedCPUMilli,
		RequestedMemBytes: spec.RequestedMemBytes,
	}

	if len(samples) == 0 {
		rec.Verdict = VerdictNoData
		rec.Reason = "no usage samples collected yet"
		return rec
	}

	cpuValues := make([]int64, len(samples))
	memValues := make([]int64, len(samples))
	for i, s := range samples {
		cpuValues[i] = s.CPUMilli
		memValues[i] = s.MemBytes
	}
	p95CPU := Percentile(cpuValues, 95)
	p95Mem := Percentile(memValues, 95)
	rec.ObservedP95CPUMilli = p95CPU
	rec.ObservedP95MemBytes = p95Mem
	rec.SuggestedCPUMilli = int64(float64(p95CPU) * headroomFactor)
	rec.SuggestedMemBytes = int64(float64(p95Mem) * headroomFactor)

	if spec.RequestedCPUMilli == 0 || spec.RequestedMemBytes == 0 {
		rec.Verdict = VerdictMissingRequests
		rec.Reason = "no CPU and/or memory request configured -- the scheduler can't bin-pack this reliably"
		return rec
	}

	if spec.LimitCPUMilli > 0 && float64(p95CPU) > float64(spec.LimitCPUMilli)*limitApproachFactor {
		rec.Verdict = VerdictApproachingLimit
		rec.Reason = "p95 CPU usage is within 10% of the configured limit -- at risk of throttling"
		return rec
	}
	if spec.LimitMemBytes > 0 && float64(p95Mem) > float64(spec.LimitMemBytes)*limitApproachFactor {
		rec.Verdict = VerdictApproachingLimit
		rec.Reason = "p95 memory usage is within 10% of the configured limit -- at risk of OOMKill"
		return rec
	}

	if p95CPU > spec.RequestedCPUMilli || p95Mem > spec.RequestedMemBytes {
		rec.Verdict = VerdictUnderProvisioned
		rec.Reason = "p95 usage regularly exceeds the configured request"
		return rec
	}

	cpuOver := spec.RequestedCPUMilli > 0 && float64(spec.RequestedCPUMilli) > float64(p95CPU)*overProvisionedFactor
	memOver := spec.RequestedMemBytes > 0 && float64(spec.RequestedMemBytes) > float64(p95Mem)*overProvisionedFactor
	if cpuOver || memOver {
		rec.Verdict = VerdictOverProvisioned
		rec.Reason = "requested resources are more than 2x observed p95 usage"
		return rec
	}

	rec.Verdict = VerdictRightSized
	rec.Reason = "requested resources are within a reasonable margin of observed usage"
	return rec
}
