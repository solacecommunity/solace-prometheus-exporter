package semp

import (
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetRaidSemp1 Get system disk information (for Appliance)
func (semp *Semp) GetRaidSemp1(ch chan<- PrometheusMetric) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Disk struct {
					DiskInfos struct {
						InternalDisks struct {
							DiskInfo []struct {
								Number                     string  `xml:"number"`
								AdministrativeStateEnabled bool    `xml:"administrative-state-enabled"`
								State                      string  `xml:"state"`
								DeviceModel                string  `xml:"device-model"`
								Capacity                   float64 `xml:"capacity"`
							} `xml:"disk-info"`
							RaidState      string `xml:"raid-state"`
							ReloadRequired bool   `xml:"reload-required"`
						} `xml:"internal-disks"`
					} `xml:"disk-infos"`
				} `xml:"disk"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	command := "<rpc><show><disk></disk></show></rpc>"
	var target Data
	if up, err := semp.fetchSemp1(command, "RaidSemp1", 1, &target); err != nil {
		return up, err
	}

	for _, disk := range target.RPC.Show.Disk.DiskInfos.InternalDisks.DiskInfo {
		ch <- semp.NewMetric(MetricDesc["Raid"]["system_disk_state"], prometheus.GaugeValue, encodeMetricMulti(disk.State, []string{"Down", "Up", "-"}), disk.Number, disk.DeviceModel)
		ch <- semp.NewMetric(MetricDesc["Raid"]["system_disk_AdministrativeStateEnabled"], prometheus.GaugeValue, encodeMetricBool(disk.AdministrativeStateEnabled), disk.Number, disk.DeviceModel)
	}

	ch <- semp.NewMetric(MetricDesc["Raid"]["system_raid_state"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Disk.DiskInfos.InternalDisks.RaidState, []string{"Disabled", "in fully redundant state", "-"}))
	ch <- semp.NewMetric(MetricDesc["Raid"]["system_reload_required"], prometheus.GaugeValue, encodeMetricBool(target.RPC.Show.Disk.DiskInfos.InternalDisks.ReloadRequired))

	return 1, nil
}
