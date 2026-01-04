package semp

import (
	"encoding/xml"
	"solace_exporter/internal/semp/types"

	"github.com/go-kit/log/level"
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
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	command := "<rpc><show><disk></disk></show></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "RaidSemp1", 1)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't scrape GetRaidSemp1", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't decode Xml GetRaidSemp1", "err", err, "broker", semp.brokerURI)
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

	for _, disk := range target.RPC.Show.Disk.DiskInfos.InternalDisks.DiskInfo {
		ch <- semp.NewMetric(MetricDesc["Raid"]["system_disk_state"], prometheus.GaugeValue, encodeMetricMulti(disk.State, []string{"Down", "Up", "-"}), disk.Number, disk.DeviceModel)
		ch <- semp.NewMetric(MetricDesc["Raid"]["system_disk_AdministrativeStateEnabled"], prometheus.GaugeValue, encodeMetricBool(disk.AdministrativeStateEnabled), disk.Number, disk.DeviceModel)
	}

	ch <- semp.NewMetric(MetricDesc["Raid"]["system_raid_state"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Disk.DiskInfos.InternalDisks.RaidState, []string{"Disabled", "in fully redundant state", "-"}))
	ch <- semp.NewMetric(MetricDesc["Raid"]["system_reload_required"], prometheus.GaugeValue, encodeMetricBool(target.RPC.Show.Disk.DiskInfos.InternalDisks.ReloadRequired))

	return 1, nil
}
