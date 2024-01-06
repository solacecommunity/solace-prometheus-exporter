package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Get system memory information
func (e *Semp) GetMemorySemp1(ch chan<- PrometheusMetric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Memory struct {
					PhysicalUsagePercent     float64 `xml:"physical-memory-usage-percent"`
					SubscriptionUsagePercent float64 `xm:"subscription-memory-usage-percent"`
					SlotInfos                struct {
						SlotInfo []struct {
							Slot             string  `xml:"slot"`
							NabBufLoadFactor float64 `xml:"nab-buffer-load-factor"`
						} `xml:"slot-info"`
					} `xml:"slot-infos"`
				} `xml:"memory"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><memory/></show></rpc>"
	body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", command, "MemorySemp1", 1)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape MemorySemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml MemorySemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
		return 0, errors.New("unexpected result: see log")
	}

	ch <- e.NewMetric(MetricDesc["Memory"]["system_memory_physical_usage_percent"], prometheus.GaugeValue, target.RPC.Show.Memory.PhysicalUsagePercent)
	ch <- e.NewMetric(MetricDesc["Memory"]["system_memory_subscription_usage_percent"], prometheus.GaugeValue, target.RPC.Show.Memory.SubscriptionUsagePercent)
	ch <- e.NewMetric(MetricDesc["Memory"]["system_nab_buffer_load_factor"], prometheus.GaugeValue, target.RPC.Show.Memory.SlotInfos.SlotInfo[0].NabBufLoadFactor)

	return 1, nil
}
