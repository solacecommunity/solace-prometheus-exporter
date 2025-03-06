package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// GetHealthSemp1 Get system health information
func (semp *Semp) GetHealthSemp1(ch chan<- PrometheusMetric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				System struct {
					Health struct {
						DiskLatencyMinimumValue     float64 `xml:"disk-latency-minimum-value"`
						DiskLatencyMaximumValue     float64 `xml:"disk-latency-maximum-value"`
						DiskLatencyAverageValue     float64 `xml:"disk-latency-average-value"`
						DiskLatencyCurrentValue     float64 `xml:"disk-latency-current-value"`
						ComputeLatencyMinimumValue  float64 `xml:"compute-latency-minimum-value"`
						ComputeLatencyMaximumValue  float64 `xml:"compute-latency-maximum-value"`
						ComputeLatencyAverageValue  float64 `xml:"compute-latency-average-value"`
						ComputeLatencyCurrentValue  float64 `xml:"compute-latency-current-value"`
						MateLinkLatencyMinimumValue float64 `xml:"mate-link-latency-minimum-value"`
						MateLinkLatencyMaximumValue float64 `xml:"mate-link-latency-maximum-value"`
						MateLinkLatencyAverageValue float64 `xml:"mate-link-latency-average-value"`
						MateLinkLatencyCurrentValue float64 `xml:"mate-link-latency-current-value"`
					} `xml:"health"`
				} `xml:"system"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
			Reason string `xml:"reason,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><system><health/></system></show ></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "HealthSemp1", 1)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't scrape HealthSemp1. Attention this is only supported by software broker not by appliances", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't decode Xml HealthSemp1", "err", err, "broker", semp.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(semp.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "reason", target.ExecuteResult.Reason, "broker", semp.brokerURI)
		return 0, errors.New("unexpected result: " + target.ExecuteResult.Reason + ". see log for further details")
	}

	ch <- semp.NewMetric(MetricDesc["Health"]["system_disk_latency_min_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.DiskLatencyMinimumValue/1e6)
	ch <- semp.NewMetric(MetricDesc["Health"]["system_disk_latency_max_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.DiskLatencyMaximumValue/1e6)
	ch <- semp.NewMetric(MetricDesc["Health"]["system_disk_latency_avg_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.DiskLatencyAverageValue/1e6)
	ch <- semp.NewMetric(MetricDesc["Health"]["system_disk_latency_cur_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.DiskLatencyCurrentValue/1e6)
	ch <- semp.NewMetric(MetricDesc["Health"]["system_compute_latency_min_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.ComputeLatencyMinimumValue/1e6)
	ch <- semp.NewMetric(MetricDesc["Health"]["system_compute_latency_max_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.ComputeLatencyMaximumValue/1e6)
	ch <- semp.NewMetric(MetricDesc["Health"]["system_compute_latency_avg_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.ComputeLatencyAverageValue/1e6)
	ch <- semp.NewMetric(MetricDesc["Health"]["system_compute_latency_cur_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.ComputeLatencyCurrentValue/1e6)
	ch <- semp.NewMetric(MetricDesc["Health"]["system_mate_link_latency_min_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.MateLinkLatencyMinimumValue/1e6)
	ch <- semp.NewMetric(MetricDesc["Health"]["system_mate_link_latency_max_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.MateLinkLatencyMaximumValue/1e6)
	ch <- semp.NewMetric(MetricDesc["Health"]["system_mate_link_latency_avg_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.MateLinkLatencyAverageValue/1e6)
	ch <- semp.NewMetric(MetricDesc["Health"]["system_mate_link_latency_cur_seconds"], prometheus.GaugeValue, target.RPC.Show.System.Health.MateLinkLatencyCurrentValue/1e6)

	return 1, nil
}
