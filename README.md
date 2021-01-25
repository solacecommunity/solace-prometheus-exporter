
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v2.0%20adopted-ff69b4.svg)](CODE_OF_CONDUCT.md)

# solace-prometheus-exporter, a Prometheus Exporter for Solace Message Brokers

## Overview
TODO: Fill in with quick explanation and maybe an arch diagram from the video. 

Video Intro available on youtube: [Integrating Prometheus and Grafana with Solace PubSub+ | Solace Community Lightning Talk
](https://youtu.be/72Wz5rrStAU?t=35)

## Features

The exporter is written in go, based on the Solace Legacy SEMP protocol.<br/>
It implements the following endpoints:<br/>
```
http://<host>:<port>/         Document page showing list of endpoints
http://<host>:<port>/metrics             Golang and standard Prometheus metrics
http://<host>:<port>/solace-std          Solace metrics for System and VPN levels
http://<host>:<port>/solace-det          Solace metrics for Messaging Clients and Queues
http://<host>:<port>/solace-broker-std   Solace Broker only Standard Metrics (System)
http://<host>:<port>/solace-vpn-std      Solace Vpn only Standard Metrics (VPN), available to non-global access right admins
http://<host>:<port>/solace-vpn-stats    Solace Vpn only Statistics Metrics (VPN), available to non-global access right admins
http://<host>:<port>/solace-vpn-det      Solace Vpn only Detailed Metrics (VPN), available to non-global access right admins
```
The [registered](https://github.com/prometheus/prometheus/wiki/Default-port-allocations) default port for Solace is 9628<br/>

## Usage

```
solace_exporter -h
usage: solace_exporter [&lt;flags&gt;]

Flags:
  -h, --help                     Show context-sensitive help (also try --help-long and --help-man).
      --log.level=info           Only log messages with the given severity or above. One of: [debug, info, warn, error]
      --log.format=logfmt        Output format of log messages. One of: [logfmt, json]
      --config-file=CONFIG-FILE  Path and name of config file. See sample file solace_exporter.ini.</code></pre>
```

The configuration parameters can be placed into a config file or into a set of environment variables or can be given via URL. For Docker you should prefer the environment variable configuration method (see below).<br/> If the exporter is started with a config file argument then the config file entries have precedence over the environment variables. If a parameter is neither found in URL nor the config file nor in the environment the exporter exits with an error.<br/>

### Config File

```bash
solace_exporter --config-file /path/to/config/file.ini
```

Sample config file:
```ini
[solace]
# Address to listen on for web interface and telemetry.
listenAddr=0.0.0.0:9628

# Base URI on which to scrape Solace broker.
scrapeUri=http://localhost:8080

# Basic Auth username for http scrape requests to Solace broker.
username=admin

# Basic Auth password for http scrape requests to Solace broker.
password=admin

# Timeout for http scrape requests to Solace broker.
timeout=5s

# Flag that enables SSL certificate verification for the scrape URI.
sslVerify=false

# Flag that enables scrape of redundancy metrics. Should be used for broker HA groups.
redundancy=false
```

### Environment Variables
Sample environment variables:
```bash
SOLACE_LISTEN_ADDR=0.0.0.0:9628
SOLACE_SCRAPE_URI=http://localhost:8080
SOLACE_USERNAME=admin
SOLACE_PASSWORD=admin
SOLACE_TIMEOUT=5s
SOLACE_SSL_VERIFY=false
SOLACE_REDUNDANCY=false
```

### URL

You can call:
https://your_exporter:9628/solace-vpn-std?scrapeURI=https%3A%2F%2Fyour-broker%3A943&username=monitoring&password=monitoring

This allows you to over write the parameters:
- scrapeURI
- username
- password

This provides you a single exporter for all your on prem broker.

Security: Only use this feature with HTTPS.

#### Sample prometheus config

```prometheus
- job_name: 'solace-std'
  scrape_interval: 15s
  metrics_path: /solace-std
  static_configs:
    - targets:
      - https://USER:PASSWORD@first-broker:943
      - https://USER:PASSWORD@second-broker:943
      - https://USER:PASSWORD@third-broker:943
  relabel_configs:
    - source_labels: [__address__]
      target_label: __param_target
    - source_labels: [__param_target]
      target_label: instance
    - target_label: __address__
      replacement: solace-exporter:9628
```

## Build

### Default Build
```bash
cd &lt;some-directory&gt;/solace_exporter
go build
```

## Docker

### Build Docker Image

A build Dockerfile is included in the repository.<br/>
This is used to automatically build and push the latest image to the Dockerhub repository [dabgmx/solace-exporter](https://hub.docker.com/r/dabgmx/solace-exporter)

### Run Docker Image

Environment variables are recommended to parameterize the exporter in Docker.<br/>
Put the following parameters, adapted to your situation, into a file on the local host, e.g. env.txt:<br/>
```bash
SOLACE_LISTEN_ADDR=0.0.0.0:9628
SOLACE_SCRAPE_URI=http://localhost:8080
SOLACE_USERNAME=admin
SOLACE_PASSWORD=admin
SOLACE_TIMEOUT=5s
SOLACE_SSL_VERIFY=false
SOLACE_REDUNDANCY=false
```

Then run
```bash
docker run -d \
 -p 9628:9628 \
 --env-file env.txt \
 --name solace-exporter \
 dabgmx/solace-exporter
```

## Bonus Material

The sub directory **testfiles** contains some sample curl commands and their outputs. This is just fyi and not needed for building.

## Resources

For more information try these resources:

- The Solace Developer Portal website at: https://solace.dev
- Ask the [Solace Community](https://solace.community)

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct, and the process for submitting pull requests to us.

## Authors

See the list of [contributors](https://github.com/solacecommunity/<github-repo>/graphs/contributors) who participated in this project.

## License

See the [LICENSE](LICENSE) file for details.
