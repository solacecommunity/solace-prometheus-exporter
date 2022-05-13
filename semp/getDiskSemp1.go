package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"strings"
)

// Get system disk information (for Appliance)
func (e *Semp) GetDiskSemp1(ch chan<- prometheus.Metric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Disk struct {
					DiskInfos struct {
						DiskInfo []struct {
							Path        string  `xml:"mounted-on"`
							DeviceName  string  `xml:"file-system"`
							TotalBlocks float64 `xml:"blocks"`
							UsedBlocks  float64 `xml:"used"`
							AvailBlocks float64 `xml:"available"`
							UsedPercent string  `xml:"use"`
						} `xml:"disk-info"`
					} `xml:"disk-infos"`
				} `xml:"disk"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><disk><detail/></disk></show></rpc>"
	body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape DiskSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml DiskSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
		return 0, errors.New("unexpected result: see log")
	}

	blockSize := 1024.0
	for _, disk := range target.RPC.Show.Disk.DiskInfos.DiskInfo {
		var usedPercent float64
		usedPercent, _ = strconv.ParseFloat(strings.Trim(disk.UsedPercent, "%"), 64)
		ch <- prometheus.MustNewConstMetric(MetricDesc["Disk"]["system_disk_used_percent"], prometheus.GaugeValue, usedPercent, disk.Path, disk.DeviceName)
		ch <- prometheus.MustNewConstMetric(MetricDesc["Disk"]["system_disk_used_bytes"], prometheus.GaugeValue, disk.UsedBlocks*blockSize, disk.Path, disk.DeviceName)
		ch <- prometheus.MustNewConstMetric(MetricDesc["Disk"]["system_disk_avail_bytes"], prometheus.GaugeValue, disk.AvailBlocks*blockSize, disk.Path, disk.DeviceName)
	}

	return 1, nil
}
