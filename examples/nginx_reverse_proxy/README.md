# Sample nginx + solace exporter

This example should be used in an kubernetes environment.

## Purpose

Having the solace exporter automated scaled out. What helps you to run a very big setup of solace broker.

Or you have multiple broker but a single exporter and not want to expose broker credentials to prometheus.

## Usage

Get helm version 3.x https://helm.sh/

Get kubectl https://kubernetes.io/docs/tasks/tools/

Login to you k8n cluster.

```sh
kubectl login
```

Create a helm config `values-YOUR.yaml` containing your brokers and the hostname where to find the exporter proxy:
```yaml
exporter_proxy:
  hostname: demo.local

broker:
  demo-broker-a:
    sempUrl: http://your-broker-a:8080
    sempUsername: admin
    sempPassword: secret
    vpnFilter: "*"
  demo-broker-b:
    sempUrl: http://your-broker-b:8080
    sempUsername: solace
    sempPassword: solace
    vpnFilter: "*"
```

Deploy script to your namespace:

```sh
helm upgrade --install -n YOUR_NAMESPACE -f values.yaml -f values-YOUR.yaml solace-exporter .
```

Now you will get following endpoints:
- http://demo.local/demo-broker-a/broker
- http://demo.local/demo-broker-a/std 
- http://demo.local/demo-broker-a/stats  
- http://demo.local/demo-broker-a/det   
- http://demo.local/demo-broker-a/queue_stats/YOUR-FILTER
- http://demo.local/demo-broker-a/det/YOUR-FILTER
- http://demo.local/demo-broker-b/broker
- http://demo.local/demo-broker-b/std 
- http://demo.local/demo-broker-b/stats  
- http://demo.local/demo-broker-b/det   
- http://demo.local/demo-broker-b/queue_stats/YOUR-FILTER
- http://demo.local/demo-broker-b/det/YOUR-FILTER

Based on your broker configuration in your helm chart.

## How to scrape those endpoints

You prometheus scrape config could now look like this:

```yaml
scrape_configs: 
  - job_name: 'solace_std'
    scrape_interval: 2m
    scrape_timeout: 1m
    scheme: http
    static_configs:
    - targets:
      - demo-broker-a 
      - demo-broker-b
        
    relabel_configs:
    - source_labels: [__address__]
      target_label: __metrics_path__
      replacement: /$1/std
      
    - source_labels: [__address__]
      target_label: instance
    
    - target_label: __address__
      replacement: demo.local:80  # The exporters proxy address

        
  - job_name: 'solace_stats'
    scrape_interval: 5m
    scrape_timeout: 3m
    scheme: http
    static_configs:
    - targets:
      - demo-broker-a 
      - demo-broker-b
        
    relabel_configs:
    - source_labels: [__address__]
      target_label: __metrics_path__
      replacement: /$1/stats
      
    - source_labels: [__address__]
      target_label: instance
    
    - target_label: __address__
      replacement: demo.local:80  # The exporters proxy address
```