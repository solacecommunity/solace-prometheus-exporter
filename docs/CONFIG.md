# Configuration Guide
The Solace Prometheus Exporter can be configured using four methods (in order of precedence):

* URL Parameters (overwrites everything for dynamic scrapes)
* HTTP Headers (e.g. `x-solace-broker-username`)
* Configuration File (`.ini`)
* Environment Variables

## ⚙️ Settings
| Environment Variable                | Config Key                | Default        | Description                                                                                                                                                                                                 |
|-------------------------------------|---------------------------|----------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `PREFETCH_INTERVAL`                 | `prefetchInterval`        | `0s`           | 0s means disabled. When set an interval, all well configured endpoints will fetched async. This may help you to deal with slower broker or extreme amount of results.                                       |
| `SOLACE_DEFAULT_VPN`                | `defaultVpn`              | `default`      | Message VPN name                                                                                                                                                                                            |
| `SOLACE_EXPORTER_AUTH_PASSWORD`     | `exporterAuthPassword`    | -              | Password for basic auth                                                                                                                                                                                     |
| `SOLACE_EXPORTER_AUTH_SCHEME`       | `exporterAuthScheme`      | `none`         | Enables authentication for the exporters own HTTP endpoints. Allowed values: `none` or `basic`.                                                                                                             |
| `SOLACE_EXPORTER_AUTH_USERNAME`     | `exporterAuthUsername`    | -              | Username for basic auth                                                                                                                                                                                     |
| `SOLACE_IS_HW_BROKER`               | `isHWBroker`              | `false`        | Flag that enables HW Broker specific targets and disables SW specific ones                                                                                                                                  |
| `SOLACE_LISTEN_ADDR`                | `listenAddr`              | `0.0.0.0:9628` | Address to listen on for web interface and telemetry                                                                                                                                                        |
| `SOLACE_LISTEN_CERTTYPE`            | `certType`                | -              | Set the certificate type PEM                                                                                                                                                                                | PKCS12. Make sure to provide certificate and private key files for PEM or PKCS12 file and password |
| `SOLACE_LISTEN_TLS`                 | `enableTLS`               | `true`         | Enable TLS on listenAddr endpoint. Make sure to provide certificate and private key files when using certType=PEM or or PKCS12 file and password when using PKCS12                                          |
| `SOLACE_LOG_BROKER_IS_SLOW_WARNING` | `logBrokerToSlowWarnings` | `true`         |                                                                                                                                                                                                             |
| `SOLACE_OAUTH_CLIENT_ID`            | `oAuthClientID`           | -              |                                                                                                                                                                                                             |
| `SOLACE_OAUTH_CLIENT_SCOPE`         | `oAuthClientScope`        | -              |                                                                                                                                                                                                             |
| `SOLACE_OAUTH_CLIENT_SECRET`        | `oAuthClientSecret`       | -              |                                                                                                                                                                                                             |
| `SOLACE_OAUTH_ISSUER`               | `oAuthIssuer`             | -              |                                                                                                                                                                                                             |
| `SOLACE_OAUTH_TOKEN_URL`            | `oAuthTokenURL`           | -              |                                                                                                                                                                                                             |
| `SOLACE_PARALLEL_SEMP_CONNECTIONS`  | `parallelSempConnections` | `1`            | Maximum connections to the configured broker. Keep in mind solace advices us to use max 10 SEMP connects per seconds. Don't increase this value if your broker may have more thant 100 clients, queues, ... |
| `SOLACE_PASSWORD`                   | `password`                | `admin`        | Basic Auth password for HTTP scrape requests to Solace broker                                                                                                                                               |
| `SOLACE_PKCS12_FILE`                | `pkcs12File`              | -              | Path to the server certificate (including intermediates and CA's certificate)                                                                                                                               |
| `SOLACE_PKCS12_PASS`                | `pkcs12Pass`              | -              | Password to decrypt PKCS12 file                                                                                                                                                                             |
| `SOLACE_PRIVATE_KEY`                | `privateKey`              | -              | Path to the private key pem file                                                                                                                                                                            |
| `SOLACE_SCRAPE_URI`                 | `scrapeURI`               | -              | URI on which to scrape Solace broker                                                                                                                                                                        |
| `SOLACE_SERVER_CERT`                | `certificate`             | -              | Path to the server certificate (including intermediates and CA's certificate)                                                                                                                               |
| `SOLACE_SEMP_PAGE_SIZE`             | `sempPageSize`            | `100`          | Number of elements per SEMP v1 paging request                                                                                                                                                               |
| `SOLACE_SSL_VERIFY`                 | `sslVerify`               | `false`        | Flag that enables SSL certificate verification for the scrape URI                                                                                                                                           |
| `SOLACE_TIMEOUT`                    | `timeout`                 | `5s`           | Timeout for HTTP scrape requests to Solace broker                                                                                                                                                           |
| `SOLACE_USERNAME`                   | `username`                | `admin`        | Basic Auth username for HTTP scrape requests to Solace broker                                                                                                                                               |
| `SECRET_BACKEND`                    | `secretBackend`           | -              | Selects the secret-manager backend. `hashicorp` enables HashiCorp Vault; unset or `none` = skip vault resolution. See [Secret Management](#-secret-management).                                             |
| `SECRET_CACHE_TTL`                  | `secretCacheTTL`          | `60s`          | How long a resolved *static* (non-leased) Vault secret is cached before being re-read. Set to `0s` to disable caching entirely. Has no effect on dynamic/leased secrets, which are always cached for half their actual lease duration. See [Secret Management](#-secret-management).                     |

> **Note:** Config file keys in the `[solace]` section are matched case-insensitively, so historically diverging
> spellings remain interchangeable — for example `scrapeURI` and `scrapeUri` (and the `scrapeURI`/`scrapeUri` URL
> parameter) all resolve to the same setting.

## 🧩 Modular Endpoints
The `/solace` endpoint allows you to granularly define which metrics to collect using HTTP GET parameters. This is the recommended way to optimize performance and reduce broker load.

### Dynamic Configuration via URL Parameters or HTTP Headers
You can overwrite the broker configuration dynamically for each request. This is useful when using the exporter as a generic proxy for multiple brokers.

| URL Parameter | HTTP Header                 | Description                                                                                     |
|---------------|-----------------------------|-------------------------------------------------------------------------------------------------|
| `username`    | `x-solace-broker-username`  | Basic Auth username for Solace broker                                                           |
| `password`    | `x-solace-broker-password`  | Basic Auth password for Solace broker                                                           |
| `scrapeURI`   | `x-solace-broker-scrapeuri` | URI of the Solace broker                                                                        |
| `timeout`     | `x-solace-broker-timeout`   | Timeout for the request (e.g. `10s`)                                                            |
| `secretBackend` | `x-solace-secret-backend` | *(unset)* uses the global `SECRET_BACKEND`; `none` skips vault resolution (plain text).         |   

**Priority**: URL Parameter > HTTP Header > Configuration File / Environment Variable.

### Parameter Syntax
Each parameter key must be a **scrape target** (see list below) prefixed by `m.`. The value consists of **2–3 parts**, delimited by a pipe `|`:
1. VPN Filter: Wildcards (`*`) are supported for SEMP v1.
2. Item Filter: Wildcards (`*`) are supported for SEMP v1.
3. Metric Filter: (SEMP v2 only) A comma-separated list of specific metrics to return.
**Example**: `m.QueueStats=myVpn|ARBON*` fetches stats for all queues starting with "ARBON" in "myVpn".

### SEMP v1 vs. SEMP v2 Endpoints
| Feature       | SEMP v1 Endpoints                 | SEMP v2 Endpoints (Experimental)                                                                                           |
|---------------|-----------------------------------|----------------------------------------------------------------------------------------------------------------------------|
| VPN Filter    | Supports wildcards (`*`).         | No wildcards. Must be a specific name.                                                                                     |
| Item Filter   | Supports wildcards.               | Supports full [v2 filters](https://docs.solace.com/Admin/SEMP/SEMP-Features.htm#Filtering) (e.g., `queueName!=internal*`). |
| Metric Filter | Not supported.                    | Supported. Limits returned fields to save resources.                                                                       |
| Performance   | Fast (e.g., 37s for 4.5k queues). | Slower (e.g., 136s for 4.5k queues).                                                                                       |

### Supported Scrape Targets
| Scrape Target                         | VPN Filter | Item Filter | Metrics Filter | Performance Impact                                                    | Corresponding CLI Command                                                          | Supported By        |
|:--------------------------------------|:-----------|:------------|----------------|:----------------------------------------------------------------------|:-----------------------------------------------------------------------------------|:--------------------|
| Alarm                                 | no         | no          | no             | dont harm broker                                                      | show alarm                                                                         | appliance           |
| Bridge                                | yes        | yes         | no             | dont harm broker                                                      | show bridge itemFilter message-vpn vpnFilter                                       | software, appliance |
| BridgeDetail                          | yes        | yes         | no             | may harm broker if many bridges                                       | show bridge itemFilter message-vpn vpnFilter detail                                | software, appliance |
| BridgeClientCert                      | yes        | yes         | no             | dont harm broker                                                      | show bridge itemFilter message-vpn vpnFilter client-certificate                    | software, appliance |
| BridgeRemote                          | yes        | yes         | no             | dont harm broker                                                      | show bridge itemFilter message-vpn vpnFilter                                       | software, appliance |
| BridgeStats                           | yes        | yes         | no             | has a very small performance down site                                | show bridge itemFilter message-vpn vpnFilter stats                                 | software, appliance |
| Client                                | yes        | yes         | no             | may harm broker if many clients                                       | show client itemFilter message-vpn vpnFilter connected                             | software, appliance |
| ClientConnections                     | yes        | no          | no             | may harm broker if many clients                                       | show client itemFilter stats                                                       | software, appliance |
| ClientMessageSpoolEgress              | no         | yes         | no             | may harm broker if many clients                                       | show client itemFilter message-spool egress connected                              | software, appliance |
| ClientMessageSpoolStats               | no         | yes         | no             | may harm broker if many clients                                       | show client itemFilter stats                                                       | software, appliance |
| ClientProfile                         | yes        | no          | no             | dont harm                                                             | show client-profile * message-vpn vpnFilter detail                                 | software, appliance |
| ClientSlowSubscriber                  | yes        | yes         | no             | may harm broker if many clients but less expensive than `ClientStats` | show client itemFilter message-vpn vpnFilter slow-subscriber                       | software, appliance |
| ClientStats                           | no         | no          | no             | may harm broker if many clients                                       | show client itemFilter stats count 100 (paged)                                     | software, appliance |
| ClockDetail                           | no         | no          | no             | dont harm broker                                                      | show clock detail                                                                  | appliance           |
| ClusterLinks                          | no         | yes         | no             | dont harm broker                                                      | show the state of the cluster links. Filters are for clusterName and linkName      | software, appliance |
| ConfigSync (only for HA broker)       | no         | no          | no             | dont harm broker                                                      | show config-sync                                                                   | software, appliance |
| ConfigSyncRouter (only for HA broker) | no         | no          | no             | dont harm broker                                                      | show config-sync database router                                                   | software, appliance |
| ConfigSyncVpn (only for HA broker)    | yes        | no          | no             | dont harm broker                                                      | show config-sync database message-vpn vpnFilter                                    | software, appliance |
| Disk                                  | no         | no          | no             | dont harm broker                                                      | show disk detail                                                                   | appliance           |
| Environment                           | yes        | no          | no             | dont harm broker                                                      | show environment                                                                   | appliance           |
| GlobalStats                           | no         | no          | no             | dont harm broker                                                      | show stats client                                                                  | software, appliance |
| GlobalSystemInfo                      | no         | no          | no             | dont harm broker                                                      | show system                                                                        | software, appliance |
| Hardware                              | no         | no          | no             | dont harm broker                                                      | show hardware                                                                      | appliance           |
| Health                                | no         | no          | no             | dont harm broker                                                      | show system health                                                                 | software            |
| Interface                             | no         | yes         | no             | dont harm broker                                                      | show interface interfaceFilter                                                     | software, appliance |
| InterfaceHW                           | no         | yes         | no             | dont harm broker                                                      | show interface interfaceFilter                                                     | appliance           |
| Memory                                | no         | no          | no             | dont harm broker                                                      | show memory                                                                        | software, appliance |
| MqttSession                           | yes        | yes         | no             | may harm broker if many mqtt sessions                                 | show message-vpn vpnFilter mqtt mqtt-session itemFilter count 100 (paged)          | software, appliance |
| QueueDetails                          | yes        | yes         | no             | may harm broker if many queues                                        | SempV2 monitoring /queue/getMsgVpnQueues 100 (paged)                               | software, appliance |
| QueueRates                            | yes        | yes         | no             | DEPRECATED: may harm broker if many queues                            | show queue itemFilter message-vpn vpnFilter rates count 100 (paged)                | software, appliance |
| QueueStats                            | yes        | yes         | no             | may harm broker if many queues                                        | show queue itemFilter message-vpn vpnFilter rates count 100 (paged)                | software, appliance |
| QueueStatsV2                          | yes        | yes         | yes            | may harm broker if many queues                                        | show queue itemFilter message-vpn vpnFilter rates count 100 (paged)                | software, appliance |
| Raid                                  | no         | no          | no             | dont harm broker                                                      | show disk                                                                          | appliance           |
| RDP/ Rest Consumers                   | yes        | yes         | no             | may harm broker if many REST consumers                                | show message-vpn <vpnFiler> rest rest-consumer <itemFiler> stats count 100 (paged) | software, appliance |
| Redundancy (only for HA broker)       | no         | no          | no             | dont harm broker                                                      | show redundancy                                                                    | software, appliance |
| Replication (only for DR broker)      | no         | no          | no             | dont harm broker                                                      | show replication stats                                                             | software, appliance |
| Spool                                 | no         | no          | no             | dont harm broker                                                      | show message-spool                                                                 | software, appliance |
| StorageElement                        | no         | yes         | no             | dont harm broker                                                      | show storage-element storageElementFilter                                          | software            |
| TopicEndpointDetails                  | yes        | yes         | no             | may harm broker if many topic-endpoints                               | show topic-endpoint itemFilter message-vpn vpnFilter detail count 100 (paged)      | software, appliance |
| TopicEndpointRates                    | yes        | yes         | no             | DEPRECATED: may harm broker if many topic-endpoints                   | show topic-endpoint itemFilter message-vpn vpnFilter rates count 100 (paged)       | software, appliance |
| TopicEndpointStats                    | yes        | yes         | no             | may harm broker if many topic-endpoint                                | show topic-endpoint itemFilter message-vpn vpnFilter rates count 100 (paged)       | software, appliance |
| Version                               | no         | no          | no             | dont harm broker                                                      | show version                                                                       | software, appliance |
| Vpn                                   | yes        | no          | no             | dont harm broker                                                      | show message-vpn vpnFilter                                                         | software, appliance |
| VpnReplication                        | yes        | no          | no             | dont harm broker                                                      | show message-vpn vpnFilter replication                                             | software, appliance |
| VpnSpool                              | yes        | no          | no             | dont harm broker                                                      | show message-spool message-vpn vpnFilter                                           | software, appliance |
| VpnStats                              | yes        | no          | no             | has a very small performance down site                                | show message-vpn vpnFilter stats count 100 (paged)                                 | software, appliance |

### ⚠️ Metric Collisions
There are metrics that may be provided by multiple endpoints. But not with the same labels. Avoid using these simultaneously. Otherwise it will cause Prometheus errors.
For example:

| Scrape Target        | Sample Metric                                                                                                                                                            |
|:---------------------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| ClientSlowSubscriber | `solace_client_slow_subscriber{client_name="Try-Me-Pub/solclientjs/chrome-120.0.0-Windows-0.0.0/4120211072/0001",client_address="10.170.74.225",vpn_name="AaaBbbCcc"} 1` |
| ClientStats          | `solace_client_slow_subscriber{client_name="Try-Me-Pub/solclientjs/chrome-120.0.0-Windows-0.0.0/4120211072/0001",client_username="my_username",vpn_name="AaaBbbCcc"} 1`  |

## 🔐 Secret Management

The exporter supports pluggable secret backends via the `SECRET_BACKEND` setting (environment variable or `[solace]` config key; env var takes precedence). Any config field that accepts credentials (`username`, `password`, `oAuthClientSecret`, `pkcs12Pass`, `exporterAuthUsername`, `exporterAuthPassword`) can hold a `vault:<path>#<field>` reference that is resolved at startup or per request. Plain text values pass through unchanged, so mixed fleets require zero special-casing.

### Backends

| `SECRET_BACKEND` | Backend | Description |
|---|---|---|
| *(unset)* / `none` | No-op | All values treated as plain text. No secret-manager client is created. |
| `hashicorp` | HashiCorp Vault | Reads standard `VAULT_*` env vars, authenticates, and optionally renews tokens in the background. |

Adding a new backend (AWS Secrets Manager, Azure Key Vault, GCP Secret Manager, CyberArk, ...) means implementing the `Backend` interface in `internal/secret/` and adding a `case` to `NewResolverFromConfig`.

### Per-request backend override

The per-request `secretBackend` parameter (or `x-solace-secret-backend` header) can override the global backend:

| Value | Behaviour |
|---|---|
| *(unset)* | Uses the global `SECRET_BACKEND` setting (vault resolution if configured). |
| `none` | Skips vault resolution entirely — `username` and `password` are used as-is (plain text). |

This is useful when one Prometheus scrape job targets brokers with plain text credentials while another targets vault-backed brokers.

### HashiCorp Vault setup sample

#### KV layout

```
secret/data/solace/solace01        { username: "monitoring", password: "s3cret" }
secret/data/solace/solace02        { username: "monitoring", password: "..." }
```

Both KV v1 and KV v2 are supported transparently (v2 is detected by the presence of a `metadata` key alongside the `data` wrapper).

#### Vault policy

```hcl
path "secret/data/solace/*" {
  capabilities = ["read"]
}
```

#### Token options

**VAULT_TOKEN env var** (simplest) — set `VAULT_TOKEN` directly, e.g. via a Kubernetes Secret:

```yaml
env:
  - name: VAULT_TOKEN
    valueFrom:
      secretKeyRef:
        name: vault-token
        key: token
```

**VAULT_TOKEN_FILE** (init-container pattern) — a short-lived wrapped token dropped into a tmpfs emptyDir:

```yaml
initContainers:
  - name: vault-init
    image: vault:latest
    command: ["vault", "write", "-format=json", "-wrap-ttl=60s",
              "auth/token/create", "policy=solace-exporter"]
    volumeMounts:
      - name: vault-token
        mountPath: /vault/token
containers:
  - name: exporter
    env:
      - name: VAULT_TOKEN_FILE
        value: /vault/token/vault-token
    volumeMounts:
      - name: vault-token
        mountPath: /vault/token
        readOnly: true
volumes:
  - name: vault-token
    emptyDir:
      medium: Memory
```

The token file is deleted immediately after being read.

#### Vault env vars

| Variable | Purpose |
|---|---|
| `VAULT_ADDR` | Vault server URL (e.g. `https://vault.internal:8200`) |
| `VAULT_TOKEN` | Auth token (preferred for K8s Secret env vars) |
| `VAULT_TOKEN_FILE` | Path to a one-shot token file (e.g. init-container wrapping). Deleted after read. |
| `VAULT_CACERT` | CA certificate for TLS |
| `VAULT_SKIP_VERIFY` | Skip TLS verification (`true`/`false`) |

#### Dynamic secrets

For dynamic database-style credentials (e.g. `database/creds/solace-role`), the resolver caches values at half the lease duration and refreshes automatically. No special exporter config is needed — just use the same `vault:<path>#<field>` syntax.

#### Static secret caching

Static secrets (plain KV fields, which don't carry a Vault lease) are cached for `SECRET_CACHE_TTL` / `secretCacheTTL` (default `60s`) before being re-read, so a burst of scrapes for the same broker doesn't hit Vault on every request. Lower it for faster pickup of a rotated secret at the cost of more Vault reads; set it to `0s` to disable caching for static secrets entirely. This setting has no effect on dynamic/leased secrets (see above).

#### Token renewal

If the token from `VAULT_TOKEN` / `VAULT_TOKEN_FILE` is renewable, the exporter renews it in the background at half its remaining TTL. Before starting that loop, it looks the token up once (`auth/token/lookup-self`) to check whether it's actually renewable — a root token or other non-renewable token is never eligible for renewal, so the exporter logs this once and skips the loop entirely rather than retrying forever.

If a renewal attempt itself fails (e.g. Vault is briefly unreachable), the exporter retries with exponential backoff — starting at 30s and doubling up to a 10-minute cap — instead of logging an error every 30 seconds indefinitely. Each delay is spread by ±20% of jitter so a fleet of exporters sharing one Vault doesn't retry in lockstep. The backoff resets after the next successful renewal.

#### Config examples

**Enable Vault in the INI config:**

```ini
[solace]
secretBackend = hashicorp
```

Or via environment variable:

```bash
export SECRET_BACKEND=hashicorp
```

**Single static broker (ini):**

```ini
[solace]
scrapeUri = https://solace01:943
username  = vault:secret/data/solace/solace01#username
password  = vault:secret/data/solace/solace01#password
```

**Multiple brokers via Prometheus relabel:**

```yaml
scrape_configs:
  - job_name: solace-multi
    metrics_path: /solace-std
    static_configs:
      - targets: ["solace01:8080"]
        labels:
          __tmp_username: "vault:secret/data/solace/solace01#username"
          __tmp_password: "vault:secret/data/solace/solace01#password"
      - targets: ["solace02:8080"]
        labels:
          __tmp_username: "vault:secret/data/solace/solace02#username"
          __tmp_password: "vault:secret/data/solace/solace02#password"
    relabel_configs:
      - source_labels: [__tmp_username]
        target_label: __param_username
      - source_labels: [__tmp_password]
        target_label: __param_password
      - source_labels: [__address__]
        target_label: __param_scrapeURI
        replacement: "http://${1}"
      - target_label: __address__
        replacement: solace-exporter:9628
```

### 🛠 Custom Endpoint Aliases (INI Config)
To keep your Prometheus scrape URLs short, you can define aliases in your `.ini`:
```ini
[endpoint.solace-custom]
ClientStats = *|*
VpnStats = *|*
```
**Usage**: Access these combined metrics via `http://<exporter-ip>:9628/solace-custom`.

If you want to use wildcards to only have a subset but need more than one wildcard,
you have to add a dot and an incrementing number. Like this:

```ini
[endpoint.my-sample]
QueueRates.0 = *|internal*
QueueRates.1 = *|bridge_*
```

#### 💡 Examples
* **Legacy Equivalent**: Get the same result as the `solace-det` endpoint, but only from VPN `myVpn`: `.../solace?m.ClientStats=myVpn|*&m.VpnStats=myVpn|*&m.BridgeStats=myVpn|*&m.QueueRates=myVpn|*&m.QueueDetails=myVpn|*`
* **Targeted Scrape**: Get all queue information, where the queue name starts with `BRAVO` or `ARBON` and only from VPN `myVpn`: `.../solace?m.QueueStatsV2=myVpn|queueName!=internal*|solace_queue_msg_shutdown_discarded`
* **Multi-Broker**: Overwrite the target broker dynamically: `.../solace?m.VpnStats=*|*&scrapeURI=http://another-broker:8080&username=monitoring&password=monitoring`
