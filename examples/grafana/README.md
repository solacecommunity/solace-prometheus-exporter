# Grafana dashboards

# Requirements

For some minor parts of the boards, an additional plugin is required. ["flant-statusmap-pane"](https://grafana.com/grafana/plugins/flant-statusmap-panel/)

# Data scraping

For the dashboards, the broker is identified via the "instance" label.

Here you can either use an automatically generated label:

Generate "instance" label based on the broker url.
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

Or give it as well-defined custom label:

```prometheus
- job_name: 'solace-std'
  scrape_interval: 15s
  metrics_path: /solace-std
  static_configs:
    - targets:
      - https://USER:PASSWORD@first-broker:943
      labels:
        instance: name-of-first-broker
        
    - targets:
      - https://USER:PASSWORD@second-broker:943
      labels:
        instance: name-of-second-broker
        
    - targets:
      - https://USER:PASSWORD@third-broker:943
      labels:
        instance: name-of-third-broker

  relabel_configs:
    - source_labels: [__address__]
      target_label: __param_target
    - target_label: __address__
      replacement: solace-exporter:9628
```

If you don't like to have broker credentials in your prometheus configuration, you have two options:
- Use the exporter proxy as sidecar to each broker and provide the credentials for each broker in its dedicated sidecar.
- Use a nginx reverse proxy. @See examples/nginx_reverse_proxy. This might be helpful if your requirements are:
  - Monitor solace cloud broker
  - If you have a central database of all your broker
  - If you want to provide the exporter centralized to avoid a high-version distribution. What dashboards might be complicated.
  - If you like to provide prometheus target url to you monitoring without everyone let to know your read only monitoring user for the broker.
