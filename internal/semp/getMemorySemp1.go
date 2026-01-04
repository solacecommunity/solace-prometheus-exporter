package semp

import (
	"encoding/xml"
	"solace_exporter/internal/semp/types"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// GetMemorySemp1 Get system memory information
func (semp *Semp) GetMemorySemp1(ch chan<- PrometheusMetric) (float64, error) {
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
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	command := "<rpc><show><memory/></show></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "MemorySemp1", 1)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't scrape MemorySemp1", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't decode Xml MemorySemp1", "err", err, "broker", semp.brokerURI)
		return 0, err
	}
	if err := target.ExecuteResult.OK(); err != nil {
		_ = level.Error(semp.logger).Log(
			"msg", "unexpected result",
			"command", command,
			"result", target.ExecuteResult.Result,
			"reason", target.ExecuteResult.Reason,
			"broker", semp.brokerURI,
		)
		return 0, err
	}

	ch <- semp.NewMetric(MetricDesc["Memory"]["system_memory_physical_usage_percent"], prometheus.GaugeValue, target.RPC.Show.Memory.PhysicalUsagePercent)
	ch <- semp.NewMetric(MetricDesc["Memory"]["system_memory_subscription_usage_percent"], prometheus.GaugeValue, target.RPC.Show.Memory.SubscriptionUsagePercent)
	ch <- semp.NewMetric(MetricDesc["Memory"]["system_nab_buffer_load_factor"], prometheus.GaugeValue, target.RPC.Show.Memory.SlotInfos.SlotInfo[0].NabBufLoadFactor)

	return 1, nil
}
