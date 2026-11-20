package semp

import (
	"solace_exporter/internal/semp/types"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// GetDiskSemp1 Get system disk information (for Appliance)
func (semp *Semp) GetDiskSemp1(ch chan<- PrometheusMetric) (float64, error) {
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
		types.Reply
	}

	command := "<rpc><show><disk><detail/></disk></show></rpc>"
	var target Data
	if up, err := semp.fetchSemp1(command, "DiskSemp1", 1, &target); err != nil {
		return up, err
	}

	blockSize := 1024.0
	for _, disk := range target.RPC.Show.Disk.DiskInfos.DiskInfo {
		var usedPercent float64
		usedPercent, _ = strconv.ParseFloat(strings.Trim(disk.UsedPercent, "%"), 64)
		ch <- semp.NewMetric(MetricDesc["Disk"]["system_disk_used_percent"], prometheus.GaugeValue, usedPercent, disk.Path, disk.DeviceName)
		ch <- semp.NewMetric(MetricDesc["Disk"]["system_disk_used_bytes"], prometheus.GaugeValue, disk.UsedBlocks*blockSize, disk.Path, disk.DeviceName)
		ch <- semp.NewMetric(MetricDesc["Disk"]["system_disk_avail_bytes"], prometheus.GaugeValue, disk.AvailBlocks*blockSize, disk.Path, disk.DeviceName)
	}

	return 1, nil
}
