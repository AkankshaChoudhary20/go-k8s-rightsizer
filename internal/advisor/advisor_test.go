package advisor

import (
	"testing"
	"time"

	"go-k8s-rightsizer/internal/kparse"
)

func sample(cpu, mem int64) kparse.Sample {
	return kparse.Sample{Timestamp: time.Now(), CPUMilli: cpu, MemBytes: mem}
}

func TestPercentile(t *testing.T) {
	values := []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	if p := Percentile(values, 50); p != 50 {
		t.Errorf("p50 = %d, want 50", p)
	}
	if p := Percentile(values, 95); p != 90 {
		t.Errorf("p95 = %d, want 90", p)
	}
	if p := Percentile(values, 100); p != 100 {
		t.Errorf("p100 = %d, want 100", p)
	}
	if p := Percentile(values, 0); p != 10 {
		t.Errorf("p0 = %d, want 10", p)
	}
}

func TestPercentile_EmptyIsZero(t *testing.T) {
	if p := Percentile(nil, 95); p != 0 {
		t.Errorf("Percentile(nil) = %d, want 0", p)
	}
}

func TestPercentile_DoesNotMutateInput(t *testing.T) {
	values := []int64{30, 10, 20}
	_ = Percentile(values, 50)
	if values[0] != 30 || values[1] != 10 || values[2] != 20 {
		t.Errorf("Percentile mutated its input slice: %v", values)
	}
}

func TestAnalyze_NoData(t *testing.T) {
	spec := kparse.Spec{RequestedCPUMilli: 500, RequestedMemBytes: 256 * 1024 * 1024}
	rec := Analyze(spec, nil)
	if rec.Verdict != VerdictNoData {
		t.Errorf("expected VerdictNoData, got %s", rec.Verdict)
	}
}

func TestAnalyze_MissingRequests(t *testing.T) {
	spec := kparse.Spec{} // no requests configured at all
	samples := []kparse.Sample{sample(50, 50*1024*1024)}
	rec := Analyze(spec, samples)
	if rec.Verdict != VerdictMissingRequests {
		t.Errorf("expected VerdictMissingRequests, got %s", rec.Verdict)
	}
}

func TestAnalyze_OverProvisioned(t *testing.T) {
	// Requesting 1000m/1000Mi but only ever using ~50m/50Mi.
	spec := kparse.Spec{RequestedCPUMilli: 1000, RequestedMemBytes: 1000 * 1024 * 1024}
	var samples []kparse.Sample
	for i := 0; i < 20; i++ {
		samples = append(samples, sample(50, 50*1024*1024))
	}
	rec := Analyze(spec, samples)
	if rec.Verdict != VerdictOverProvisioned {
		t.Errorf("expected VerdictOverProvisioned, got %s (%s)", rec.Verdict, rec.Reason)
	}
	if rec.SuggestedCPUMilli >= spec.RequestedCPUMilli {
		t.Errorf("suggested CPU (%d) should be lower than current request (%d)", rec.SuggestedCPUMilli, spec.RequestedCPUMilli)
	}
}

func TestAnalyze_UnderProvisioned(t *testing.T) {
	// Requesting only 100m/100Mi but consistently using ~300m/300Mi.
	spec := kparse.Spec{RequestedCPUMilli: 100, RequestedMemBytes: 100 * 1024 * 1024}
	var samples []kparse.Sample
	for i := 0; i < 20; i++ {
		samples = append(samples, sample(300, 300*1024*1024))
	}
	rec := Analyze(spec, samples)
	if rec.Verdict != VerdictUnderProvisioned {
		t.Errorf("expected VerdictUnderProvisioned, got %s (%s)", rec.Verdict, rec.Reason)
	}
	if rec.SuggestedCPUMilli <= spec.RequestedCPUMilli {
		t.Errorf("suggested CPU (%d) should be higher than current request (%d)", rec.SuggestedCPUMilli, spec.RequestedCPUMilli)
	}
}

func TestAnalyze_ApproachingLimit(t *testing.T) {
	// Requested and used are close, but usage is creeping up on the configured limit.
	spec := kparse.Spec{
		RequestedCPUMilli: 200,
		RequestedMemBytes: 200 * 1024 * 1024,
		LimitCPUMilli:     220,
		LimitMemBytes:     220 * 1024 * 1024,
	}
	var samples []kparse.Sample
	for i := 0; i < 20; i++ {
		samples = append(samples, sample(210, 100*1024*1024)) // 210/220 = 95% of CPU limit
	}
	rec := Analyze(spec, samples)
	if rec.Verdict != VerdictApproachingLimit {
		t.Errorf("expected VerdictApproachingLimit, got %s (%s)", rec.Verdict, rec.Reason)
	}
}

func TestAnalyze_RightSized(t *testing.T) {
	// Requesting 150m/150Mi, using ~120m/120Mi -- comfortably close, not over, not under.
	spec := kparse.Spec{RequestedCPUMilli: 150, RequestedMemBytes: 150 * 1024 * 1024}
	var samples []kparse.Sample
	for i := 0; i < 20; i++ {
		samples = append(samples, sample(120, 120*1024*1024))
	}
	rec := Analyze(spec, samples)
	if rec.Verdict != VerdictRightSized {
		t.Errorf("expected VerdictRightSized, got %s (%s)", rec.Verdict, rec.Reason)
	}
}
