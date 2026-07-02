package semp

import (
	"encoding/xml"
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetMemorySemp1 Get system memory information
func (semp *Semp) GetMemorySemp1(ch chan<- PrometheusMetric) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Memory struct {
					PhysicalMemory          struct {
						MemoryInfo []struct {
							MemoryType       string  `xml:"type"`
							TotalInKB        float64 `xml:"total-in-kb"`
							UsedInKB         float64 `xml:"used-in-kb"`
							FreeInKB         float64 `xml:"free-in-kb"`
							BuffersInKB      float64 `xml:"buffers-in-kb" optional:"yes"`
							CachedInKB       float64 `xml:"cached-in-kb" optional:"yes"`
						} `xml:"memory-info"`
					} `xml:"physical-memory"`
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
		semp.logger.Error("Can't scrape MemorySemp1", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer func() { _ = body.Close() }()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		semp.logger.Error("Can't decode Xml MemorySemp1", "err", err, "broker", semp.brokerURI)
		return 0, err
	}
	if err := target.ExecuteResult.OK(); err != nil {
		semp.logger.Error("unexpected result",
			"command", command,
			"result", target.ExecuteResult.Result,
			"reason", target.ExecuteResult.Reason,
			"broker", semp.brokerURI,
		)
		return 0, err
	}

    for _, memoryInfo := range target.RPC.Show.Memory.PhysicalMemory.MemoryInfo {
        memoryType := memoryInfo.MemoryType
        totalInKB := memoryInfo.TotalInKB
        usedInKB := memoryInfo.UsedInKB
        freeInKB := memoryInfo.FreeInKB
        ch <- semp.NewMetric(MetricDesc["Memory"]["system_memory_physical_total_kb"], prometheus.GaugeValue, totalInKB, memoryType)
        ch <- semp.NewMetric(MetricDesc["Memory"]["system_memory_physical_used_kb"], prometheus.GaugeValue, usedInKB, memoryType)
        ch <- semp.NewMetric(MetricDesc["Memory"]["system_memory_physical_free_kb"], prometheus.GaugeValue, freeInKB, memoryType)
        if memoryInfo.MemoryType == "Memory" {
            buffersInKB := memoryInfo.BuffersInKB
            cachedInKB := memoryInfo.CachedInKB
            ch <- semp.NewMetric(MetricDesc["Memory"]["system_memory_physical_buffers_kb"], prometheus.GaugeValue, buffersInKB, memoryType)
            ch <- semp.NewMetric(MetricDesc["Memory"]["system_memory_physical_cached_kb"], prometheus.GaugeValue, cachedInKB, memoryType)
        }
    }

	ch <- semp.NewMetric(MetricDesc["Memory"]["system_memory_physical_usage_percent"], prometheus.GaugeValue, target.RPC.Show.Memory.PhysicalUsagePercent)
	ch <- semp.NewMetric(MetricDesc["Memory"]["system_memory_subscription_usage_percent"], prometheus.GaugeValue, target.RPC.Show.Memory.SubscriptionUsagePercent)
	// SlotInfo may be empty on software/cloud brokers or malformed replies; guard the index to avoid a panic that
	// would crash the whole exporter for every broker.
	if slotInfos := target.RPC.Show.Memory.SlotInfos.SlotInfo; len(slotInfos) > 0 {
		ch <- semp.NewMetric(MetricDesc["Memory"]["system_nab_buffer_load_factor"], prometheus.GaugeValue, slotInfos[0].NabBufLoadFactor)
	}

	return 1, nil
}
