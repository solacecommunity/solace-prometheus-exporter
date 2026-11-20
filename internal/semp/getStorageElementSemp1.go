package semp

import (
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetStorageElementSemp1 Get system storage-element information (for Software Broker)
func (semp *Semp) GetStorageElementSemp1(ch chan<- PrometheusMetric, storageElementFilter string) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				StorageElements struct {
					StorageElement []struct {
						Name        string  `xml:"name"`
						Path        string  `xml:"path"`
						DeviceName  string  `xml:"device-name"`
						TotalBlocks float64 `xml:"total-blocks"`
						UsedBlocks  float64 `xml:"used-blocks"`
						AvailBlocks float64 `xml:"available-blocks"`
						UsedPercent float64 `xml:"used-percentage"`
					} `xml:"storage-element"`
				} `xml:"storage-element"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	command := "<rpc><show><storage-element><pattern>" + semp1Escape(storageElementFilter) + "</pattern></storage-element></show></rpc>"
	var target Data
	if up, err := semp.fetchSemp1(command, "StorageElementSemp1", 1, &target); err != nil {
		return up, err
	}

	blockSize := 1024.0

	for _, element := range target.RPC.Show.StorageElements.StorageElement {
		ch <- semp.NewMetric(MetricDesc["StorageElement"]["system_storage_used_percent"], prometheus.GaugeValue, element.UsedPercent, element.Path, element.DeviceName, element.Name)
		ch <- semp.NewMetric(MetricDesc["StorageElement"]["system_storage_used_bytes"], prometheus.GaugeValue, element.UsedBlocks*blockSize, element.Path, element.DeviceName, element.Name)
		ch <- semp.NewMetric(MetricDesc["StorageElement"]["system_storage_avail_bytes"], prometheus.GaugeValue, element.AvailBlocks*blockSize, element.Path, element.DeviceName, element.Name)
	}

	return 1, nil
}
