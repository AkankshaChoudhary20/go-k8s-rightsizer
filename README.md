# go-k8s-rightsizer

A CLI that reads real Kubernetes container resource usage vs. configured
requests/limits and tells you what's misconfigured: over-provisioned
(wasting cluster capacity), under-provisioned (at risk of throttling or
OOMKill), or missing requests entirely (the scheduler can't bin-pack it
properly).

## The problem this solves

Most clusters accumulate resource requests that nobody ever revisits --
someone guessed "512Mi ought to be enough" eighteen months ago, the
workload grew or shrank, and now capacity planning is guesswork. This
tool turns `kubectl top` history + `kubectl get pods -o json` into a
straight verdict per container.

## How it works

Two subcommands, no `client-go` dependency -- just parses the exact text/
JSON formats `kubectl` already prints:

- **`collect`** -- shells out to
  `kubectl top pods --all-namespaces --containers --no-headers`, parses
  it, and appends timestamped samples to an NDJSON history file. Run this
  on a cron/timer to build up a usage history over time.
- **`analyze`** -- reads the accumulated history plus the latest
  `kubectl get pods -o json` output, computes p95 CPU/memory usage per
  container, and prints a verdict:
  - `missing_requests` -- no request configured at all
  - `over_provisioned` -- requested more than 2x observed p95 usage
  - `under_provisioned` -- p95 usage regularly exceeds the request
  - `approaching_limit` -- p95 usage is within 10% of the configured limit
  - `right_sized` -- requested resources are within a reasonable margin

## Try it without a cluster

```bash
go run . analyze -history testdata/sample_history.ndjson -specs testdata/sample_specs.json
```

```
VERDICT              POD/CONTAINER                REQUESTED        P95 OBSERVED     SUGGESTED        REASON
missing_requests     batch-worker-1/worker        0m/0Mi           374m/483Mi       448m/579Mi       no CPU and/or memory request configured...
over_provisioned     web-6f8d9b-abcde/web         1000m/1000Mi     99m/98Mi         118m/117Mi       requested resources are more than 2x observed p95 usage
right_sized          checkout-7d4f-xyz/checkout   100m/128Mi       94m/114Mi        112m/136Mi       requested resources are within a reasonable margin of observed usage
```

## Against a real cluster

```bash
# run periodically (cron, a CronJob, whatever) to build up history:
go run . collect -history history.ndjson

# then, whenever you want a report:
kubectl get pods --all-namespaces -o json > specs.json
go run . analyze -history history.ndjson -specs specs.json
```

## Tests

```bash
go test ./...
```

17 tests, all against realistic fixture data (real `kubectl top` column
format, a real `PodList` JSON shape including a container with no
`resources` block at all):

- Quantity parsing: CPU (`100m`, `1`, `0.5`) and memory (`128Mi`, `1Gi`,
  `500Ki`, raw bytes) formats, plus rejection of malformed input.
- `kubectl top` output parsing against a realistic multi-line fixture,
  blank-line tolerance, and malformed-line rejection.
- `kubectl get pods -o json` parsing, including a container with an
  empty `resources: {}` block.
- The recommendation engine itself: every verdict (`no_data`,
  `missing_requests`, `over_provisioned`, `under_provisioned`,
  `approaching_limit`, `right_sized`) triggered by realistic usage
  patterns, plus percentile-calculation correctness (including that it
  doesn't mutate its input).

**Honest caveat:** `collect` (the part that shells out to a real
`kubectl`) was not tested against a live cluster -- there wasn't one
available in this environment. Everything downstream of parsing (the
`analyze` pipeline, the recommendation engine) was run end-to-end against
the bundled `testdata/` fixtures, shown above.

## What's intentionally not here

No Prometheus integration (this uses point-in-time `kubectl top` samples,
not a real metrics time series), no automatic patching of pod specs --
this recommends, it doesn't act. Wiring `analyze`'s suggestions into a
`kubectl patch` or a VPA object is a reasonable next step, deliberately
left out here.
