# Configuration Guide
The Solace Prometheus Exporter can be configured using three methods (in order of precedence):

* URL Parameters (overwrites everything for dynamic scrapes)
* Configuration File (`.ini`)
* Environment Variables

## ⚙️ Settings
| Environment Variable | Config Key | Default | Description |
| ------------- | ------------- | ------------- | ------------- |
| `PREFETCH_INTERVAL` | `prefetchInterval` | `0s` | 0s means disabled. When set an interval, all well configured endpoints will fetched async. This may help you to deal with slower broker or extreme amount of results. |
| `SOLACE_DEFAULT_VPN` | `defaultVpn` | `default` | Message VPN name |
| `SOLACE_EXPORTER_AUTH_PASSWORD` | `exporterAuthPassword` | - | Password for basic auth |
| `SOLACE_EXPORTER_AUTH_SCHEME` | `exporterAuthScheme` | `none` | Enables authentication for the exporters own HTTP endpoints. Allowed values: `none` or `basic`. |
| `SOLACE_EXPORTER_AUTH_USERNAME` | `exporterAuthUsername` | - | Username for basic auth |
| `SOLACE_IS_HW_BROKER` | `isHWBroker` | `false` | Flag that enables HW Broker specific targets and disables SW specific ones |
| `SOLACE_LISTEN_ADDR` | `listenAddr` | `0.0.0.0:9628` | Address to listen on for web interface and telemetry |
| `SOLACE_LISTEN_CERTTYPE` | `certType` | - | Set the certificate type PEM | PKCS12. Make sure to provide certificate and private key files for PEM or PKCS12 file and password |
| `SOLACE_LISTEN_TLS` | `enableTLS` | `true` | Enable TLS on listenAddr endpoint. Make sure to provide certificate and private key files when using certType=PEM or or PKCS12 file and password when using PKCS12 |
| `SOLACE_LOG_BROKER_IS_SLOW_WARNING` | `logBrokerToSlowWarnings` | `true` |  |
| `SOLACE_OAUTH_CLIENT_ID` | `oAuthClientID` | - |  |
| `SOLACE_OAUTH_CLIENT_SCOPE` | `oAuthClientScope` | - |  |
| `SOLACE_OAUTH_CLIENT_SECRET` | `oAuthClientSecret` | - |  |
| `SOLACE_OAUTH_ISSUER` | `oAuthIssuer` | - |  |
| `SOLACE_OAUTH_TOKEN_URL` | `oAuthTokenURL` | - |  |
| `SOLACE_PARALLEL_SEMP_CONNECTIONS` | `parallelSempConnections` | `1` | Maximum connections to the configured broker. Keep in mind solace advices us to use max 10 SEMP connects per seconds. Don't increase this value if your broker may have more thant 100 clients, queues, ... |
| `SOLACE_PASSWORD` | `password` | `admin` | Basic Auth password for HTTP scrape requests to Solace broker |
| `SOLACE_PKCS12_FILE` | `pkcs12File` | - | Path to the server certificate (including intermediates and CA's certificate) |
| `SOLACE_PKCS12_PASS` | `pkcs12Pass` | - | Password to decrypt PKCS12 file |
| `SOLACE_PRIVATE_KEY` | `privateKey` | - | Path to the private key pem file |
| `SOLACE_SCRAPE_URI` | `scrapeUri` | - | URI on which to scrape Solace broker |
| `SOLACE_SERVER_CERT` | `certificate` | - | Path to the server certificate (including intermediates and CA's certificate) |
| `SOLACE_SSL_VERIFY` | `sslVerify` | `false` | Flag that enables SSL certificate verification for the scrape URI |
| `SOLACE_TIMEOUT` | `timeout` | `5s` | Timeout for HTTP scrape requests to Solace broker |
| `SOLACE_USERNAME` | `username` | `admin` | Basic Auth username for HTTP scrape requests to Solace broker |

## 🧩 Modular Endpoints
The `/solace` endpoint allows you to granularly define which metrics to collect using HTTP GET parameters. This is the recommended way to optimize performance and reduce broker load.

### Parameter Syntax
Each parameter key must be a **scrape target** (see list below) prefixed by `m.`. The value consists of **2–3 parts**, delimited by a pipe `|`:
1. VPN Filter: Wildcards (`*`) are supported for SEMP v1.
2. Item Filter: Wildcards (`*`) are supported for SEMP v1.
3. Metric Filter: (SEMP v2 only) A comma-separated list of specific metrics to return.
**Example**: `m.QueueStats=myVpn|ARBON*` fetches stats for all queues starting with "ARBON" in "myVpn".

### SEMP v1 vs. SEMP v2 Endpoints
| Feature | SEMP v1 Endpoints | SEMP v2 Endpoints (Experimental)|
|-----|-----|----|
| VPN Filter | Supports wildcards (`*`). | No wildcards. Must be a specific name. |
| Item Filter | Supports wildcards. | Supports full [v2 filters](https://docs.solace.com/Admin/SEMP/SEMP-Features.htm#Filtering) (e.g., `queueName!=internal*`).|
| Metric Filter | Not supported. | Supported. Limits returned fields to save resources.|
| Performance | Fast (e.g., 37s for 4.5k queues). | Slower (e.g., 136s for 4.5k queues).|

### 📋 Supported Scrape Targets
| Scrape Target                         | VPN Filter | Item Filter | Metrics Filter | Performance Impact                                                    | Corresponding CLI Command                                                              | Supported By        |
|:--------------------------------------|:--------------------|:----------------------|--------------------------|:----------------------------------------------------------------------|:-----------------------------------------------------------------------------------|:--------------------|
| Alarm                                 | no                  | no                    | no                       | dont harm broker                                                      | show alarm                                                                         | appliance           |
| Bridge                                | yes                 | yes                   | no                       | dont harm broker                                                      | show bridge itemFilter message-vpn vpnFilter                                       | software, appliance |
| BridgeRemote                          | yes                 | yes                   | no                       | dont harm broker                                                      | show bridge itemFilter message-vpn vpnFilter                                       | software, appliance |
| BridgeStats                           | yes                 | yes                   | no                       | has a very small performance down site                                | show bridge itemFilter message-vpn vpnFilter stats                                 | software, appliance |
| Client                                | yes                 | yes                   | no                       | may harm broker if many clients                                       | show client itemFilter message-vpn vpnFilter connected                             | software, appliance |
| ClientConnections                     | yes                 | no                    | no                       | may harm broker if many clients                                       | show client itemFilter stats                                                       | software, appliance |
| ClientMessageSpoolStats               | yes                 | no                    | no                       | may harm broker if many clients                                       | show client itemFilter stats(paged)                                                | software, appliance |
| ClientProfile                         | yes                 | no                    | no                       | dont harm                                                             | show client-profile * message-vpn vpnFilter detail                                 | software, appliance |
| ClientSlowSubscriber                  | yes                 | yes                   | no                       | may harm broker if many clients but less expensive than `ClientStats` | show client itemFilter message-vpn vpnFilter slow-subscriber                       | software, appliance |
| ClientStats                           | yes                 | no                    | no                       | may harm broker if many clients                                       | show client itemFilter stats (paged)                                               | software, appliance |
| ClusterLinks                          | yes                 | yes                   | no                       | dont harm broker                                                      | show the state of the cluster links. Filters are for clusterName and linkName      | software, appliance |
| ConfigSync (only for HA broker)       | no                  | no                    | no                       | dont harm broker                                                      | show config-sync                                                                   | software, appliance |
| ConfigSyncRouter (only for HA broker) | no                  | no                    | no                       | dont harm broker                                                      | show config-sync database router                                                   | software, appliance |
| ConfigSyncVpn (only for HA broker)    | yes                 | no                    | no                       | dont harm broker                                                      | show config-sync database message-vpn vpnFilter                                    | software, appliance |
| Disk                                  | no                  | no                    | no                       | dont harm broker                                                      | show disk detail                                                                   | appliance           |
| Environment                           | yes                 | no                    | no                       | dont harm broker                                                      | show environment                                                                   | appliance           |
| GlobalStats                           | no                  | no                    | no                       | dont harm broker                                                      | show stats client                                                                  | software, appliance |
| GlobalSystemInfo                      | no                  | no                    | no                       | dont harm broker                                                      | show system                                                                        | software, appliance |
| Hardware                              | no                  | no                    | no                       | dont harm broker                                                      | show hardware                                                                      | appliance           |
| Health                                | no                  | no                    | no                       | dont harm broker                                                      | show system health                                                                 | software            |
| Interface                             | no                  | yes                   | no                       | dont harm broker                                                      | show interface interfaceFilter                                                     | software, appliance |
| InterfaceHW                           | no                  | yes                   | no                       | dont harm broker                                                      | show interface interfaceFilter                                                     | appliance           |
| Memory                                | no                  | no                    | no                       | dont harm broker                                                      | show memory                                                                        | software, appliance |
| QueueDetails                          | yes                 | yes                   | no                       | may harm broker if many queues                                        | SempV2 monitoring /queue/getMsgVpnQueues 100 (paged)                               | software, appliance |
| QueueRates                            | yes                 | yes                   | no                       | DEPRECATED: may harm broker if many queues                            | show queue itemFilter message-vpn vpnFilter rates count 100 (paged)                | software, appliance |
| QueueStats                            | yes                 | yes                   | no                       | may harm broker if many queues                                        | show queue itemFilter message-vpn vpnFilter rates count 100 (paged)                | software, appliance |
| QueueStatsV2                          | yes                 | yes                   | yes                      | may harm broker if many queues                                        | show queue itemFilter message-vpn vpnFilter rates count 100 (paged)                | software, appliance |
| Raid                                  | no                  | no                    | no                       | dont harm broker                                                      | show disk                                                                          | appliance           |
| RDP/ Rest Consumers                   | yes                 | yes                   | no                       | may harm broker if many REST consumers                                | show message-vpn <vpnFiler> rest rest-consumer <itemFiler> stats count 100 (paged) | software, appliance |
| Redundancy (only for HA broker)       | no                  | no                    | no                       | dont harm broker                                                      | show redundancy                                                                    | software, appliance |
| Replication (only for DR broker)      | no                  | no                    | no                       | dont harm broker                                                      | show replication stats                                                             | software, appliance |
| Spool                                 | no                  | no                    | no                       | dont harm broker                                                      | show message-spool                                                                 | software, appliance |
| StorageElement                        | no                  | yes                   | no                       | dont harm broker                                                      | show storage-element storageElementFilter                                          | software            |
| TopicEndpointDetails                  | yes                 | yes                   | no                       | may harm broker if many topic-endpoints                               | show topic-endpoint itemFilter message-vpn vpnFilter detail count 100 (paged)      | software, appliance |
| TopicEndpointRates                    | yes                 | yes                   | no                       | DEPRECATED: may harm broker if many topic-endpoints                   | show topic-endpoint itemFilter message-vpn vpnFilter rates count 100 (paged)       | software, appliance |
| TopicEndpointStats                    | yes                 | yes                   | no                       | may harm broker if many topic-endpoint                                | show topic-endpoint itemFilter message-vpn vpnFilter rates count 100 (paged)       | software, appliance |
| Version                               | no                  | no                    | no                       | dont harm broker                                                      | show version                                                                       | software, appliance |
| Vpn                                   | yes                 | no                    | no                       | dont harm broker                                                      | show message-vpn vpnFilter                                                         | software, appliance |
| VpnReplication                        | yes                 | no                    | no                       | dont harm broker                                                      | show message-vpn vpnFilter replication                                             | software, appliance |
| VpnSpool                              | yes                 | no                    | no                       | dont harm broker                                                      | show message-spool message-vpn vpnFilter                                           | software, appliance |
| VpnStats                              | yes                 | no                    | no                       | has a very small performance down site                                | show message-vpn vpnFilter stats                                                   | software, appliance |

### ⚠️ Metric Collisions
There are metrics that may be provided by multiple endpoints. But not with the same labels. Avoid using these simultaneously. Otherwise it will cause Prometheus errors.
For example:

| Scrape Target        | Sample Metric                   |
|:---------------------|:--------------------------------------------------------------------------------------------------------------------------|
| ClientSlowSubscriber | `solace_client_slow_subscriber{client_name="Try-Me-Pub/solclientjs/chrome-120.0.0-Windows-0.0.0/4120211072/0001",client_address="10.170.74.225",vpn_name="AaaBbbCcc"} 1` |
| ClientStats          | `solace_client_slow_subscriber{client_name="Try-Me-Pub/solclientjs/chrome-120.0.0-Windows-0.0.0/4120211072/0001",client_username="my_username",vpn_name="AaaBbbCcc"} 1`  |

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
