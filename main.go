package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"time"

	"go-k8s-rightsizer/internal/advisor"
	"go-k8s-rightsizer/internal/kparse"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "collect":
		collectCmd(os.Args[2:])
	case "analyze":
		analyzeCmd(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  go-k8s-rightsizer collect -history FILE           append a kubectl snapshot to FILE (needs a working kubectl context)
  go-k8s-rightsizer analyze -history FILE -specs FILE   print recommendations from accumulated samples`)
}

// -- collect: shells out to kubectl and appends one NDJSON sample per line to -history --

func collectCmd(args []string) {
	fs := flag.NewFlagSet("collect", flag.ExitOnError)
	historyPath := fs.String("history", "history.ndjson", "path to the NDJSON sample history file (appended to)")
	fs.Parse(args)

	out, err := exec.Command("kubectl", "top", "pods", "--all-namespaces", "--containers", "--no-headers").Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "running kubectl top: %v\n", err)
		os.Exit(1)
	}

	samples, err := kparse.ParseTopContainers(string(out), time.Now())
	if err != nil {
		fmt.Fprintf(os.Stderr, "parsing kubectl top output: %v\n", err)
		os.Exit(1)
	}

	f, err := os.OpenFile(*historyPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "opening %s: %v\n", *historyPath, err)
		os.Exit(1)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, s := range samples {
		b, _ := json.Marshal(s)
		w.Write(b)
		w.WriteString("\n")
	}
	w.Flush()

	fmt.Printf("appended %d samples to %s\n", len(samples), *historyPath)
}

// -- analyze: reads accumulated samples + latest specs, prints a recommendation table --

func analyzeCmd(args []string) {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	historyPath := fs.String("history", "history.ndjson", "path to the NDJSON sample history file")
	specsPath := fs.String("specs", "", "path to `kubectl get pods -o json` output (required)")
	fs.Parse(args)

	if *specsPath == "" {
		fmt.Fprintln(os.Stderr, "error: -specs is required (path to `kubectl get pods -o json` output)")
		os.Exit(2)
	}

	specsData, err := os.ReadFile(*specsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading %s: %v\n", *specsPath, err)
		os.Exit(1)
	}
	specs, err := kparse.ParsePodSpecs(specsData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parsing %s: %v\n", *specsPath, err)
		os.Exit(1)
	}

	samples, err := readHistory(*historyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading %s: %v\n", *historyPath, err)
		os.Exit(1)
	}

	byContainer := groupSamples(samples)

	var recs []advisor.Recommendation
	for _, spec := range specs {
		key := spec.Namespace + "/" + spec.Pod + "/" + spec.Container
		recs = append(recs, advisor.Analyze(spec, byContainer[key]))
	}

	sort.Slice(recs, func(i, j int) bool {
		if recs[i].Verdict != recs[j].Verdict {
			return recs[i].Verdict < recs[j].Verdict
		}
		return recs[i].Pod < recs[j].Pod
	})

	printTable(recs)
}

func readHistory(path string) ([]kparse.Sample, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var samples []kparse.Sample
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var s kparse.Sample
		if err := json.Unmarshal(line, &s); err != nil {
			return nil, err
		}
		samples = append(samples, s)
	}
	return samples, scanner.Err()
}

func groupSamples(samples []kparse.Sample) map[string][]kparse.Sample {
	out := make(map[string][]kparse.Sample)
	for _, s := range samples {
		key := s.Namespace + "/" + s.Pod + "/" + s.Container
		out[key] = append(out[key], s)
	}
	return out
}

func printTable(recs []advisor.Recommendation) {
	fmt.Printf("%-20s %-28s %-16s %-16s %-16s %s\n",
		"VERDICT", "POD/CONTAINER", "REQUESTED", "P95 OBSERVED", "SUGGESTED", "REASON")
	for _, r := range recs {
		podContainer := r.Pod + "/" + r.Container
		requested := fmt.Sprintf("%dm/%dMi", r.RequestedCPUMilli, r.RequestedMemBytes/(1024*1024))
		observed := fmt.Sprintf("%dm/%dMi", r.ObservedP95CPUMilli, r.ObservedP95MemBytes/(1024*1024))
		suggested := fmt.Sprintf("%dm/%dMi", r.SuggestedCPUMilli, r.SuggestedMemBytes/(1024*1024))
		fmt.Printf("%-20s %-28s %-16s %-16s %-16s %s\n",
			r.Verdict, podContainer, requested, observed, suggested, r.Reason)
	}
}
