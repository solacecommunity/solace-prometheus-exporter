package semp

import (
	"encoding/xml"
	"solace_exporter/internal/semp/types"

	"github.com/go-kit/log/level"
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
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	command := "<rpc><show><storage-element><pattern>" + storageElementFilter + "</pattern></storage-element></show></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "StorageElementSemp1", 1)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't scrape StorageElementSemp1", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't decode Xml StorageElementSemp1", "err", err, "broker", semp.brokerURI)
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

	blockSize := 1024.0

	for _, element := range target.RPC.Show.StorageElements.StorageElement {
		ch <- semp.NewMetric(MetricDesc["StorageElement"]["system_storage_used_percent"], prometheus.GaugeValue, element.UsedPercent, element.Path, element.DeviceName, element.Name)
		ch <- semp.NewMetric(MetricDesc["StorageElement"]["system_storage_used_bytes"], prometheus.GaugeValue, element.UsedBlocks*blockSize, element.Path, element.DeviceName, element.Name)
		ch <- semp.NewMetric(MetricDesc["StorageElement"]["system_storage_avail_bytes"], prometheus.GaugeValue, element.AvailBlocks*blockSize, element.Path, element.DeviceName, element.Name)
	}

	return 1, nil
}
