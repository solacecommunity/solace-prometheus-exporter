# AGENTS.md

Contributor and AI-agent guide for the **solace-prometheus-exporter** codebase. It captures the conventions,
structure and invariants a change should respect. Read it alongside [`CONTRIBUTING.MD`](CONTRIBUTING.MD) and
[`docs/CONFIG.md`](docs/CONFIG.md).

## 1. Technology Stack

* **Language:** Go (see `go` directive in [`go.mod`](go.mod); module name `solace_exporter`).
* **Metrics:** `github.com/prometheus/client_golang` (the exporter implements `prometheus.Collector`).
* **CLI / logging:** `github.com/alecthomas/kingpin/v2` for flags; `github.com/prometheus/common/promslog` +
  `log/slog` for structured logging; `github.com/prometheus/common/version` for build info.
* **Config:** `gopkg.in/ini.v1` for the INI file; `os.Getenv` for environment overrides.
* **Auth / TLS:** `golang.org/x/oauth2/clientcredentials` for broker OAuth;
  `software.sslmate.com/src/go-pkcs12` for PKCS#12 keystores; standard `crypto/tls`.
* **Concurrency:** `golang.org/x/sync/semaphore` bounds concurrent SEMP connections.
* **SEMP transport:** standard `net/http`, `encoding/xml` (SEMP v1), `encoding/json` (SEMP v2).
* **Build & CI:** [`Makefile`](Makefile), multi-stage [`Dockerfile`](Dockerfile) producing a `scratch` image,
  GitHub Actions running `golangci-lint`, `go mod tidy` checks and `go test`.

## 2. Project Structure

```
cmd/solace-prometheus-exporter/   Entry point: flag parsing, HTTP handler wiring, /solace param parsing
internal/exporter/                Config, the Exporter collector, per-request config, HTTP client, auth, TLS, async prefetch
  config.struct.go                Config struct + ParseConfig (INI + env), endpoint alias parsing
  exporter.struct.go              Exporter type; wires a *semp.Semp per request
  exporter.collect.go             CollectPrometheusMetric: the scrape-target dispatch switch + Collect()
  exporter.describe.go            Describe() over the metric registry
  dataSource.struct.go            DataSource{Name, VpnFilter, ItemFilter, MetricFilter}
  auth.go / http.go / tlsServer.go  OAuth/basic visitor, HTTP client, HTTPS listener
  asyncFetcher.go                 Prefetch collector used when prefetchInterval > 0
internal/semp/                    SEMP access layer
  getXxxSemp1.go                  One file per SEMP v1 target (POST XML to /SEMP)
  getQueueStatsSemp2.go           SEMP v2 target (GET JSON from /SEMP/v2/monitor/...)
  metricDesc.go                   MetricDesc registry: label sets + all metric Descriptions
  semp.desc.struct.go             Desc, Descriptions, V2Result, field-selection helpers
  prometheusMetric.struct.go      PrometheusMetric wrapper (dedupe key, cardinality checks)
  http.go / helper.go / types/    HTTP verbs, filter escaping, shared XML types
internal/web/                     Index page handler/template + exporter Basic Auth wrapper
configs/                          Sample INI config (also baked into the Docker image)
docs/CONFIG.md                    Full settings + scrape-target reference
examples/                         Grafana dashboards and an NGINX reverse-proxy Helm chart
test/                             Captured SEMP fixtures (test/data) and an OAuth e2e stack (test/oauth)
```

## 3. Coding Standards & Conventions

* **Formatting/linting:** code must be `gofmt`-clean and pass `golangci-lint run` (`make lint`). Keep imports
  grouped (stdlib, then third-party) and use tabs for indentation.
* **Logging:** use the injected `*slog.Logger` with key/value pairs (`logger.Error("msg", "err", err, "broker",
  uri)`), not `fmt`/`log`. Do not `panic` in a scrape path (see §7).
* **Errors:** wrap with `%w` (`fmt.Errorf("...: %w", err)`); return early. Config parsing returns
  `(..., *Config, error)` and fails fast on invalid/incomplete input.
* **Metrics namespace:** every metric name is prefixed with `solace_` — this is applied once in
  `NewSemDesc` (do not repeat the prefix in call sites).
* **Value types:** pick `prometheus.CounterValue` for monotonic counters and `prometheus.GaugeValue` for gauges
  when emitting via `semp.NewMetric`.
* **No magic values:** durations, page sizes and channel capacities are named constants (`longQuery`,
  `capMetricChan`, `metricCacheChunkSize`, ...).
* **Security exceptions:** the SEMP HTTP client intentionally uses `InsecureSkipVerify` (broker certs are often
  self-signed); it is annotated `//nolint:gosec`. Do not remove the annotation without changing the behaviour.

## 4. Scrape Handlers & SEMP Access

* **Handler wiring (`main.go`):** `/metrics` serves process metrics; `/solace` parses `m.<Target>` GET params into
  `[]exporter.DataSource`; each `[endpoint.<alias>]` config section becomes its own handler. Every handler is
  wrapped with `web.WrapWithAuth` (exporter Basic Auth). When `prefetchInterval > 0`, alias handlers are served by
  an `AsyncFetcher` from cache instead of scraping inline.
* **Parameter grammar:** a `/solace` value is `vpnFilter|itemFilter[|metricFilter,...]`; fewer than two
  `|`-separated parts is skipped/rejected with a log. The metric filter (third part) is SEMP v2 only.
* **Dispatch (`exporter.collect.go`):** `CollectPrometheusMetric` switches on `DataSource.Name`. Each target has a
  bare name and a `...V1` alias mapped to the same getter (e.g. `"VpnStats", "VpnStatsV1"`). Hardware-only and
  software-only targets are gated on `e.config.IsHWBroker` and emit an explanatory error otherwise.
* **SEMP v1 getters:** build an `<rpc><show>...</rpc>` command, `POST` it to `<brokerURI>/SEMP` via
  `semp.postHTTP`, decode XML into a local `Data` struct, check `ExecuteResult.Result == "ok"`, then emit metrics.
  Paged calls loop on `types.MoreCookie` using `<num-elements>` = `sempPageSize`.
* **SEMP v2 getters:** `GET` `<brokerURI>/SEMP/v2/monitor/...` via `semp.getHTTPbytes`, decode JSON, check
  `Meta.ResponseCode == 200`, follow `Meta.Paging.NextPageURI`. Field selection uses `select=`, filtering uses
  `where=`. Unlike v1, v2 does **not** support `*` wildcards.
* **Auth:** requests are decorated by `httpRequestVisitor` from `Config.setAuthHeader` — Basic Auth or an OAuth
  bearer (optionally issuer-prefixed). The visitor is always non-nil.
* **Return contract:** getters return `(up float64, err error)`: `up >= 1` success, `up == 0` recoverable failure
  for this target, `up == -1` unrecoverable (e.g. broker unreachable) which aborts the remaining targets and is
  reported under the `global` endpoint.

## 5. Adding a Metric or Scrape Target

To add or extend a target:

1. **Describe the metrics** in `internal/semp/metricDesc.go`: add entries to an existing `Descriptions` block or a
   new one, using `NewSemDesc(fqName, sempV2field, help, variableLabels)`, and register the block under a key in the
   `MetricDesc` map. Reuse an existing `variableLabels...` slice where the label set matches.
2. **Implement the getter** as `internal/semp/getXxxSemp1.go` (or a `...Semp2.go`), following the pattern in §4:
   inner struct with `xml:`/`json:` tags, request, decode, result-code check, `ch <- semp.NewMetric(...)`, return
   `(up, err)`.
3. **Wire the dispatch** by adding a `case "Xxx", "XxxV1":` to the switch in `exporter.collect.go`, passing the
   relevant `dataSource` filters and `e.config.SempPageSize`. Gate on `IsHWBroker` if the target is
   hardware/software specific.
4. **Document it** by adding a row to the "Supported Scrape Targets" table in `docs/CONFIG.md`, the table in
   `internal/web/templates/index.html`, and the metric groups table in `README.md`.
5. **Test it** with a captured payload fixture under `test/data` and a table-driven test (see §6).

To add a new named endpoint, no code change is needed — add an `[endpoint.<name>]` section to the config file.

## 6. Testing

* **Run:** `make test` (`go test -short ./...`). Use `-short` to skip long/integration paths; `make test-coverage`
  writes an HTML report to `reports/`.
* **Style:** table-driven tests decode fixtures under `test/data` (e.g. `semp1-vpn.txt`, `semp2-queue.txt`) and
  assert emitted metrics; shared helpers live in `helper_test.go`. Add a fixture for every new getter.
* **Static checks:** `make vet` and `make lint` (golangci-lint) must pass. CI also runs `go mod tidy` and fails if
  `go.mod`/`go.sum` change, so run `go mod tidy` after touching dependencies.
* **End-to-end:** `test/oauth/docker-compose.yaml` spins up Keycloak + a Solace broker + a scrape check to exercise
  the OAuth path; it runs in CI and can be run locally with Docker Compose.

## 7. Project-Specific Patterns & Invariants

* **Config isolation:** the shared base `*Config` is never mutated after startup. The `/solace` handler derives a
  per-request `Config.Clone()` and overrides `Username`/`Password`/`ScrapeURI`/`Timeout` on the copy so concurrent
  scrapes cannot clobber each other's credentials.
* **Shared OAuth cache:** `Config.oAuthToken` is intentionally shared by pointer across clones so the token stays
  warm; the token is read fresh from the cache on every request (it refreshes shortly before expiry).
* **Never crash the process:** `Collect` and the async fetcher wrap `CollectPrometheusMetric` in `recover()` — a
  malformed broker reply must degrade to fewer metrics, not take down the exporter (which may front many brokers).
  The auth visitor must therefore always be non-nil.
* **Deduplication:** `Collect` keeps a `map[string]PrometheusMetric` keyed by `PrometheusMetric.Name()` (fqName +
  label values) and emits the last value per key. Two enabled targets that produce the same metric name with
  different label sets collide — avoid combining them (see the metric-collision note in `docs/CONFIG.md`).
* **Label cardinality:** `semp.NewMetric` panics if the number of label values differs from the `Desc`'s
  `variableLabels`, or if a value is not valid UTF-8. Pass label values in the exact order of the label slice.
* **`solace_up`:** emitted once per target with `error` and `endpoint` labels; `up == -1` is reported under
  `endpoint="global"` and stops processing further targets.
* **Prefetch semantics:** `AsyncFetcher` deprecates all cached metrics before each poll and deletes the ones not
  refreshed, so stale series disappear. All prefetch endpoints honour the same exporter Basic Auth.
* **Filters:** SEMP v1 supports `*` wildcards in VPN and item filters; SEMP v2 needs a concrete VPN (falling back to
  `defaultVpn` when the filter is `*`) and uses `where=`/`select=` query filters instead.

## 8. Example

A minimal SEMP v1 target — metric description, getter and dispatch wiring:

```go
// internal/semp/metricDesc.go
var Memory = Descriptions{
    "physical_memory_usage_percent": NewSemDesc("system_memory_physical_usage_percent", NoSempV2Ready,
        "Physical memory usage percentage.", nil),
}
// ... registered in the MetricDesc map:
//   "Memory": Memory,
```

```go
// internal/semp/getMemorySemp1.go
func (semp *Semp) GetMemorySemp1(ch chan<- PrometheusMetric) (float64, error) {
    type Data struct {
        RPC struct {
            Show struct {
                Memory struct {
                    PhysicalUsagePercent float64 `xml:"physical-memory-usage-percent"`
                } `xml:"memory"`
            } `xml:"show"`
        } `xml:"rpc"`
        ExecuteResult types.ExecuteResult `xml:"execute-result"`
    }

    body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml",
        "<rpc><show><memory/></show></rpc>", "MemorySemp1", 1)
    if err != nil {
        semp.logger.Error("Can't scrape MemorySemp1", "err", err, "broker", semp.brokerURI)
        return -1, err // unrecoverable: broker unreachable
    }
    defer func() { _ = body.Close() }()

    var target Data
    if err := xml.NewDecoder(body).Decode(&target); err != nil {
        return 0, err
    }
    if target.ExecuteResult.Result != "ok" {
        return 0, errors.New("unexpected result: see log for details")
    }

    ch <- semp.NewMetric(MetricDesc["Memory"]["physical_memory_usage_percent"],
        prometheus.GaugeValue, target.RPC.Show.Memory.PhysicalUsagePercent)
    return 1, nil
}
```

```go
// internal/exporter/exporter.collect.go — inside the dispatch switch
case "Memory", "MemoryV1":
    up, err = e.semp.GetMemorySemp1(ch)
```

Scrape it via `http://<host>:9628/solace?m.Memory=*|*`, or add `Memory = *|*` to an `[endpoint.<alias>]` section.

## Related Projects

Public ecosystem projects this exporter integrates with:

* [Prometheus](https://prometheus.io/) — scrapes and stores the exposed metrics.
* [Grafana](https://grafana.com/) — dashboards for the metrics (see [`examples/grafana`](examples/grafana)).
* [Solace PubSub+ & the SEMP API](https://docs.solace.com/Admin/SEMP/Using-SEMP.htm) — the monitored broker and its
  management protocol.
