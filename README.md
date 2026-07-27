# Solace Prometheus Exporter

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v2.0%20adopted-ff69b4.svg)](CODE_OF_CONDUCT.md)
[![Go Report Card](https://goreportcard.com/badge/github.com/solacecommunity/solace-prometheus-exporter)](https://goreportcard.com/report/github.com/solacecommunity/solace-prometheus-exporter)

The community-led standard for real-time observability and monitoring of Solace PubSub+ Event Brokers.

![Architecture overview](https://raw.githubusercontent.com/solacecommunity/solace-prometheus-exporter/master/docs/architecture_001.png)

## Overview

`solace-prometheus-exporter` is a standalone [Prometheus](https://prometheus.io/) exporter, written in Go, that
scrapes the [SEMP](https://docs.solace.com/Admin/SEMP/Using-SEMP.htm) (Solace Element Management Protocol) API of a
Solace PubSub+ broker and exposes the results as Prometheus metrics over HTTP.

The exporter sits between your brokers and your monitoring stack:

* It talks to the broker over **SEMP v1** (XML over `POST /SEMP`) and **SEMP v2 monitor** (JSON over
  `GET /SEMP/v2/monitor/...`), using Basic Auth or OAuth 2.0 client credentials.
* It publishes broker, VPN, queue, client, bridge and hardware metrics on port **9628**, ready to be scraped by
  Prometheus and visualised in [Grafana](https://grafana.com/) (ready-made dashboards ship in
  [`examples/grafana`](examples/grafana)).
* Instead of one fixed metric set, it exposes a **modular `/solace` endpoint** plus named endpoint aliases, so each
  Prometheus scrape job can request exactly the data it needs and keep broker load under control.

## Quick Start

The fastest way to get visibility into your Solace broker is with Docker:

```bash
docker run -d \
  -p 9628:9628 \
  -e SOLACE_SCRAPE_URI=http://<your-broker-ip>:8080 \
  -e SOLACE_USERNAME=admin \
  -e SOLACE_PASSWORD=admin \
  solacecommunity/solace-prometheus-exporter
```

Then scrape a bundled endpoint, for example the system + VPN standard set:

```
http://localhost:9628/solace-std
```

## Features

* **Modular scraping** &mdash; the `/solace` endpoint takes HTTP GET parameters so a scrape can request exactly the
  metrics it needs, avoiding expensive broker queries.
* **Named endpoint aliases** &mdash; group frequently used scrape targets into short, reusable URLs
  (`[endpoint.<name>]` sections in the config file).
* **SEMP v1 and SEMP v2** &mdash; over 40 scrape targets covering system, VPN, queue, client, bridge, RDP and
  hardware statistics.
* **Software and appliance brokers** &mdash; hardware-only targets (disk, RAID, environment, alarms) are enabled
  with a single `isHWBroker` flag.
* **Flexible authentication** &mdash; Basic Auth or OAuth 2.0 client-credentials to the broker, with an optional
  issuer-prefixed bearer token.
* **TLS everywhere** &mdash; serve metrics over HTTPS (PEM or PKCS#12) and optionally protect the exporter's own
  endpoints with Basic Auth.
* **Multi-broker friendly** &mdash; override the target broker, credentials and timeout per request via URL
  parameters or `x-solace-broker-*` headers, so one exporter can front many brokers.
* **Async prefetch** &mdash; optionally poll the broker on a fixed interval and serve cached metrics, smoothing out
  load on slow brokers or very large result sets.
* **Cloud-native** &mdash; small static binary, `scratch`-based container image, and a
  [registered Prometheus default port](https://github.com/prometheus/prometheus/wiki/Default-port-allocations) (9628).

## Endpoints & Scrape Modes

Once running, the exporter serves the following HTTP endpoints:

| Endpoint                | Description                                                                                       |
|-------------------------|---------------------------------------------------------------------------------------------------|
| `/`                     | Landing page listing all configured endpoints.                                                    |
| `/metrics`              | The exporter's own process metrics (Go runtime and standard Prometheus metrics).                  |
| `/solace`               | The modular endpoint. Scrape targets are supplied as `m.<Target>` GET parameters (see below).     |
| `/<alias>`              | One handler per `[endpoint.<alias>]` section defined in the config file.                           |

The bundled sample config (`configs/solace_prometheus_exporter.ini`) predefines these aliases:
`solace-std`, `solace-std-appliance`, `solace-det`, `solace-broker-std`, `solace-broker-std-appliance`,
`solace-vpn-std`, `solace-vpn-stats`, `solace-vpn-det` and `solace-vpn-rdp`.

### The modular `/solace` endpoint

Each scrape target is passed as a GET parameter whose key is `m.<Target>` and whose value has **two or three**
pipe-delimited parts:

```
m.<Target>=<vpnFilter>|<itemFilter>[|<metricFilter>]
```

1. **VPN filter** &mdash; `*` wildcards supported on SEMP v1 targets.
2. **Item filter** &mdash; `*` wildcards on SEMP v1; SEMP v2 targets accept concrete names or `where=` filters.
3. **Metric filter** &mdash; SEMP v2 only; a comma-separated allow-list of fields to limit the returned columns.

Examples:

```bash
# Queue stats for all queues starting with BRAVO in VPN "myVpn"
http://localhost:9628/solace?m.QueueStats=myVpn|BRAVO*

# Reproduce the legacy "det" set for a single VPN
http://localhost:9628/solace?m.ClientStats=myVpn|*&m.VpnStats=myVpn|*&m.BridgeStats=myVpn|*&m.QueueRates=myVpn|*&m.QueueDetails=myVpn|*

# Point a single scrape at a different broker
http://localhost:9628/solace?m.VpnStats=*|*&scrapeURI=http://another-broker:8080&username=monitoring&password=monitoring
```

Targets are dispatched case-sensitively; each also has a `...V1` alias (for example `VpnStats` and `VpnStatsV1`
are equivalent). See the [Configuration Guide](docs/CONFIG.md) for the full list of targets, their filter support
and the SEMP CLI command each one maps to.

### Per-request broker overrides

For the four connection fields below, the value is resolved as **URL parameter → HTTP header → configured value**,
so one exporter instance can be reused as a generic proxy in front of many brokers:

| URL parameter | HTTP header                 | Overrides            |
|---------------|-----------------------------|----------------------|
| `username`    | `x-solace-broker-username`  | Broker Basic Auth user |
| `password`    | `x-solace-broker-password`  | Broker Basic Auth password |
| `scrapeURI`   | `x-solace-broker-scrapeuri` | Broker SEMP base URI |
| `timeout`     | `x-solace-broker-timeout`   | Per-request timeout (e.g. `10s`) |

## Configuration

The exporter is configured through an INI **config file**, **environment variables**, and (for the dynamic scrape
fields) **URL parameters / HTTP headers**. Environment variables take precedence over the config file; the four
connection fields above can additionally be overridden per request. Point the exporter at a config file with:

```
solace_prometheus_exporter --config-file=configs/solace_prometheus_exporter.ini
```

### Command-line flags

```
usage: solace_prometheus_exporter [<flags>]

Flags:
  -h, --help                     Show context-sensitive help.
      --log.level=info           Log level: one of [debug, info, warn, error].
      --log.format=logfmt        Log output format: one of [logfmt, json].
      --config-file=CONFIG-FILE  Path to the INI config file (see configs/solace_prometheus_exporter.ini).
```

### The `[solace]` section

The global broker and listener settings live in the `[solace]` section. Each key can be overridden by the
corresponding environment variable:

| Environment variable                | Config key                | Default        | Description |
|-------------------------------------|---------------------------|----------------|-------------|
| `SOLACE_LISTEN_ADDR`                | `listenAddr`              | `0.0.0.0:9628` | Address the exporter listens on. |
| `SOLACE_SCRAPE_URI`                 | `scrapeURI`               | *(required)*   | Base URI of the broker's SEMP interface, e.g. `http://localhost:8080`. |
| `SOLACE_USERNAME`                   | `username`                | `admin`        | Basic Auth username for SEMP requests. |
| `SOLACE_PASSWORD`                   | `password`                | `admin`        | Basic Auth password for SEMP requests. |
| `SOLACE_DEFAULT_VPN`                | `defaultVpn`              | `default`      | Message VPN used for SEMP v2 targets when the VPN filter is `*`. |
| `SOLACE_TIMEOUT`                    | `timeout`                 | `5s`           | Timeout for SEMP requests to the broker. |
| `SOLACE_SSL_VERIFY`                 | `sslVerify`               | `false`        | Verify the broker's TLS certificate when scraping. |
| `SOLACE_IS_HW_BROKER`              | `isHWBroker`              | `false`        | Enable appliance (hardware) targets and disable software-only ones. |
| `SOLACE_SEMP_PAGE_SIZE`             | `sempPageSize`            | `100`          | Elements per SEMP v1 paging request. |
| `SOLACE_PARALLEL_SEMP_CONNECTIONS`  | `parallelSempConnections` | `1`            | Maximum concurrent SEMP connections to the broker (Solace advises ≤10 per second). |
| `PREFETCH_INTERVAL`                 | `prefetchInterval`        | `0s`           | If > 0, configured endpoints are fetched asynchronously on this interval and served from cache. |
| `SOLACE_LOG_BROKER_IS_SLOW_WARNING` | `logBrokerToSlowWarnings` | `true`         | Log a warning when a SEMP query takes unusually long. |

#### Serving over TLS

| Environment variable       | Config key    | Default | Description |
|----------------------------|---------------|---------|-------------|
| `SOLACE_LISTEN_TLS`        | `enableTLS`   | `false` | Serve the exporter over HTTPS. |
| `SOLACE_LISTEN_CERTTYPE`   | `certType`    | `PEM`   | Certificate type: `PEM` or `PKCS12`. |
| `SOLACE_SERVER_CERT`       | `certificate` | -       | Path to the server certificate (PEM), including intermediates. |
| `SOLACE_PRIVATE_KEY`       | `privateKey`  | -       | Path to the private key (PEM). |
| `SOLACE_PKCS12_FILE`       | `pkcs12File`  | -       | Path to the PKCS#12 keystore. |
| `SOLACE_PKCS12_PASS`       | `pkcs12Pass`  | -       | Password for the PKCS#12 keystore. |

When TLS is enabled the exporter also sets HSTS and standard hardening headers (`X-Content-Type-Options`,
`X-Frame-Options`, `Referrer-Policy`).

#### OAuth 2.0 client credentials (broker auth)

Set all four fields to authenticate to the broker with an OAuth 2.0 client-credentials flow instead of Basic Auth;
tokens are cached and refreshed automatically. An incomplete OAuth configuration is rejected at startup.

| Environment variable          | Config key           | Description |
|-------------------------------|----------------------|-------------|
| `SOLACE_OAUTH_TOKEN_URL`      | `oAuthTokenURL`      | Token endpoint URL. |
| `SOLACE_OAUTH_CLIENT_ID`      | `oAuthClientID`      | OAuth client ID. |
| `SOLACE_OAUTH_CLIENT_SECRET`  | `oAuthClientSecret`  | OAuth client secret. |
| `SOLACE_OAUTH_CLIENT_SCOPE`   | `oAuthClientScope`   | OAuth scope. |
| `SOLACE_OAUTH_ISSUER`         | `oAuthIssuer`        | Optional issuer; when set, the bearer token is issuer-prefixed as `~<base64(issuer)>~<token>`. |

#### Protecting the exporter's own endpoints

| Environment variable            | Config key             | Default | Description |
|---------------------------------|------------------------|---------|-------------|
| `SOLACE_EXPORTER_AUTH_SCHEME`   | `exporterAuthScheme`   | `none`  | `none` or `basic`. Enables Basic Auth on the exporter's HTTP endpoints. |
| `SOLACE_EXPORTER_AUTH_USERNAME` | `exporterAuthUsername` | -       | Basic Auth username for the exporter. |
| `SOLACE_EXPORTER_AUTH_PASSWORD` | `exporterAuthPassword` | -       | Basic Auth password for the exporter. |

### Endpoint alias sections

Named endpoints keep Prometheus scrape URLs short. Each key is a scrape target and its value uses the same
`vpnFilter|itemFilter[|metricFilter]` syntax as the `/solace` parameters:

```ini
[endpoint.solace-custom]
ClientStats = *|*
VpnStats    = *|*
```

The example above is served at `http://<host>:9628/solace-custom`. To use the same target more than once (for
example with different item filters), suffix the key with `.<n>`:

```ini
[endpoint.my-sample]
QueueRates.0 = *|internal*
QueueRates.1 = *|bridge_*
```

See [`docs/CONFIG.md`](docs/CONFIG.md) for the complete settings reference, the SEMP v1 vs v2 comparison, and the
metric-collision notes.

## Metrics

Every exported series is prefixed with `solace_`. Metrics are grouped into families, each produced by a scrape
target:

| Group            | Example scrape targets                                                             | What it covers |
|------------------|------------------------------------------------------------------------------------|----------------|
| Broker / system  | `Version`, `Health`, `Memory`, `Spool`, `SpoolStats`, `GlobalStats`, `GlobalSystemInfo`, `Interface` | Broker version and uptime, health, memory, message-spool usage, global client stats, NICs. |
| Redundancy / DR  | `Redundancy`, `ConfigSync`, `ConfigSyncRouter`, `ReplicationStats`                 | HA redundancy, config-sync state, replication (DR) statistics. |
| Appliance hardware | `Disk`, `Raid`, `Environment`, `Hardware`, `Alarm`, `ClockDetail`, `InterfaceHW` | Hardware-only metrics (enabled via `isHWBroker`). |
| Message VPN      | `Vpn`, `VpnStats`, `VpnSpool`, `VpnReplication`, `ConfigSyncVpn`                   | Per-VPN state, throughput, spool usage and replication. |
| Clients          | `Client`, `ClientStats`, `ClientConnections`, `ClientProfile`, `ClientSlowSubscriber`, `ClientMessageSpoolStats`, `ClientMessageSpoolEgress` | Connected clients, per-client stats, slow subscribers, per-client spool usage. |
| Queues           | `QueueStats`, `QueueStatsV2`, `QueueDetails`, `QueueRates` *(deprecated)*          | Spooled messages/bytes, discards, redelivery and other per-queue counters. |
| Topic endpoints  | `TopicEndpointStats`, `TopicEndpointDetails`, `TopicEndpointRates` *(deprecated)*  | Per-topic-endpoint statistics and details. |
| Bridges          | `Bridge`, `BridgeStats`, `BridgeDetail`, `BridgeRemote`, `BridgeClientCert`        | Bridge state, throughput, remote connections and client certificates. |
| REST delivery    | `RdpInfo`, `RdpStats`, `RestConsumerStats`                                         | REST Delivery Point info/stats and REST consumer statistics. |
| Cluster / MQTT   | `ClusterLinks`, `MqttSession`                                                      | Cluster link state and MQTT session details. |

In addition, every scrape emits a `solace_up{error, endpoint}` gauge (`1` when the target scraped successfully, `0`
otherwise) so you can alert on broker or target-level failures.

> **Metric collisions:** some metrics (for example `solace_client_slow_subscriber`) are produced by more than one
> target with different label sets. Avoid enabling colliding targets in the same scrape, or Prometheus will reject
> the sample. See [`docs/CONFIG.md`](docs/CONFIG.md#-metric-collisions) for details.

## Running

### Binary

```bash
make build
./bin/solace_prometheus_exporter --config-file=configs/solace_prometheus_exporter.ini
```

### Docker

```bash
docker run -d \
  -p 9628:9628 \
  -e SOLACE_SCRAPE_URI=http://<your-broker-ip>:8080 \
  -e SOLACE_USERNAME=admin \
  -e SOLACE_PASSWORD=admin \
  solacecommunity/solace-prometheus-exporter
```

The image ships with the sample config at `/etc/solace/solace_prometheus_exporter.ini`. To supply your own, mount
it over that path:

```bash
docker run -d -p 9628:9628 \
  -v $(pwd)/my-config.ini:/etc/solace/solace_prometheus_exporter.ini \
  solacecommunity/solace-prometheus-exporter
```

### Kubernetes

A minimal Deployment and Service configured entirely through environment variables:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: solace-exporter
spec:
  replicas: 1
  selector:
    matchLabels: { app: solace-exporter }
  template:
    metadata:
      labels: { app: solace-exporter }
    spec:
      containers:
        - name: exporter
          image: solacecommunity/solace-prometheus-exporter
          ports:
            - containerPort: 9628
          env:
            - name: SOLACE_SCRAPE_URI
              value: "http://your-broker:8080"
            - name: SOLACE_USERNAME
              value: "monitoring"
            - name: SOLACE_PASSWORD
              valueFrom:
                secretKeyRef: { name: solace-exporter, key: password }
---
apiVersion: v1
kind: Service
metadata:
  name: solace-exporter
spec:
  selector: { app: solace-exporter }
  ports:
    - port: 9628
      targetPort: 9628
```

For large or multi-broker deployments, [`examples/nginx_reverse_proxy`](examples/nginx_reverse_proxy) provides a
Helm chart that scales the exporter out behind NGINX and keeps broker credentials out of your Prometheus config.

### Prometheus scrape config

Point Prometheus at the endpoint you want and relabel the broker address, so credentials stay out of the scrape
config and the broker name becomes the `instance` label:

```yaml
- job_name: 'solace-std'
  scrape_interval: 15s
  metrics_path: /solace-std
  static_configs:
    - targets:
        - https://USER:PASSWORD@first-broker:943
        - https://USER:PASSWORD@second-broker:943
  relabel_configs:
    - source_labels: [__address__]
      target_label: __param_target
    - source_labels: [__param_target]
      target_label: instance
    - target_label: __address__
      replacement: solace-exporter:9628
```

See [`examples/grafana/README.md`](examples/grafana/README.md) for more scrape patterns.

## Grafana dashboards

Ready-to-import dashboards for brokers, VPNs and bridges live in [`examples/grafana`](examples/grafana) (with PDF
previews). Some panels require the
[`flant-statusmap-panel`](https://grafana.com/grafana/plugins/flant-statusmap-panel/) plugin. The dashboards
identify each broker via the `instance` label described above.

## Building & Testing

The project uses a standard Go toolchain and a `Makefile`:

```bash
make build          # build ./bin/solace_prometheus_exporter
make test           # go test -short ./...
make test-coverage  # write an HTML coverage report to reports/
make vet            # go vet
make lint           # golangci-lint run
```

Unit tests are table-driven against captured SEMP payloads under [`test/data`](test/data). An OAuth end-to-end
suite (Keycloak + a Solace broker + a scrape check) is defined in
[`test/oauth/docker-compose.yaml`](test/oauth) and runs in CI. Continuous integration lints, checks `go mod tidy`,
runs the tests and publishes the Docker image.

## Contributing

Contributions are welcome, whether it is a bug report, a new scrape target, or improved documentation. Please read
the [Contribution Guide](CONTRIBUTING.MD) and our [Code of Conduct](CODE_OF_CONDUCT.md). Questions about Solace
technologies are also welcome in the [Solace community](https://solace.community).

If you are adding a metric or scrape target, see [`AGENTS.md`](AGENTS.md) for the project's coding conventions and a
step-by-step guide.

## Resources

* Video: [Integrating Prometheus and Grafana with Solace](https://youtu.be/72Wz5rrStAU?t=35)
* Blog: [How to Use OAuth with solace-prometheus-exporter](https://dev.to/pascalre/securing-solace-metrics-how-to-use-oauth-with-solace-prometheus-exporter-2i6l)
* Blog: [Your Solace Prometheus Exporter Deserves a Better Pipeline](https://dev.to/pascalre/your-solace-prometheus-exporter-deserves-a-better-pipeline-3i6i)

## License

Distributed under the [MIT License](LICENSE).
