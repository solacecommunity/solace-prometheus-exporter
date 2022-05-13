package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Get system storage-element information (for Software Broker)
func (e *Semp) GetStorageElementSemp1(ch chan<- prometheus.Metric, storageElementFilter string) (ok float64, err error) {
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
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><storage-element><pattern>" + storageElementFilter + "</pattern></storage-element></show></rpc>"
	body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape StorageElementSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml StorageElementSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
		return 0, errors.New("unexpected result: see log")
	}

	blockSize := 1024.0

	for _, element := range target.RPC.Show.StorageElements.StorageElement {
		ch <- prometheus.MustNewConstMetric(MetricDesc["StorageElement"]["system_storage_used_percent"], prometheus.GaugeValue, element.UsedPercent, element.Path, element.DeviceName, element.Name)
		ch <- prometheus.MustNewConstMetric(MetricDesc["StorageElement"]["system_storage_used_bytes"], prometheus.GaugeValue, element.UsedBlocks*blockSize, element.Path, element.DeviceName, element.Name)
		ch <- prometheus.MustNewConstMetric(MetricDesc["StorageElement"]["system_storage_avail_bytes"], prometheus.GaugeValue, element.AvailBlocks*blockSize, element.Path, element.DeviceName, element.Name)
	}

	return 1, nil
}
