package semp

import (
	"encoding/xml"
	"solace_exporter/internal/semp/types"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// GetRedundancySemp1 Get system-wide basic redundancy information for HA triples
func (semp *Semp) GetRedundancySemp1(ch chan<- PrometheusMetric) (float64, error) {
	var redundancyState float64

	type Data struct {
		RPC struct {
			Show struct {
				Red struct {
					ConfigStatus      string `xml:"config-status"`
					RedundancyStatus  string `xml:"redundancy-status"`
					OperatingMode     string `xml:"operating-mode"`
					RedundancyMode    string `xml:"redundancy-mode"`
					ActiveStandbyRole string `xml:"active-standby-role"`
					MateRouterName    string `xml:"mate-router-name"`
					OperationalStatus struct {
						ADBLink  bool `xml:"adb-link-up"`
						ADBHello bool `xml:"adb-hello-up"`
					} `xml:"oper-status"`
					VirtualRouters struct {
						Primary struct {
							Status struct {
								Activity string `xml:"activity"`
							} `xml:"status"`
						} `xml:"primary"`
						Backup struct {
							Status struct {
								Activity string `xml:"activity"`
							} `xml:"status"`
						} `xml:"backup"`
					} `xml:"virtual-routers"`
				} `xml:"redundancy"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	command := "<rpc><show><redundancy/></show></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "RedundancySemp1", 1)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't scrape RedundancySemp1", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't decode Xml RedundancySemp1", "err", err, "broker", semp.brokerURI)
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

	mateRouterName := "" + target.RPC.Show.Red.MateRouterName
	ch <- semp.NewMetric(MetricDesc["Redundancy"]["system_redundancy_config"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Red.ConfigStatus, []string{"Disabled", "Enabled", "Shutdown"}), mateRouterName)
	ch <- semp.NewMetric(MetricDesc["Redundancy"]["system_redundancy_up"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Red.RedundancyStatus, []string{"Down", "Up"}), mateRouterName)
	ch <- semp.NewMetric(MetricDesc["Redundancy"]["system_redundancy_role"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Red.ActiveStandbyRole, []string{"Backup", "Primary", "Monitor", "Undefined"}), mateRouterName)
	if semp.isHWBroker {
		ch <- semp.NewMetric(MetricDesc["RedundancyHW"]["system_redundancy_hw_mode"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Red.RedundancyMode, []string{"Active/Active", "Active/Standby"}), mateRouterName)
		ch <- semp.NewMetric(MetricDesc["RedundancyHW"]["system_redundancy_hw_adb_link"], prometheus.GaugeValue, encodeMetricBool(target.RPC.Show.Red.OperationalStatus.ADBLink), mateRouterName)
		ch <- semp.NewMetric(MetricDesc["RedundancyHW"]["system_redundancy_hw_adb_hello"], prometheus.GaugeValue, encodeMetricBool(target.RPC.Show.Red.OperationalStatus.ADBHello), mateRouterName)
	}

	if target.RPC.Show.Red.ActiveStandbyRole == "Primary" && target.RPC.Show.Red.VirtualRouters.Primary.Status.Activity == "Local Active" ||
		target.RPC.Show.Red.ActiveStandbyRole == "Backup" && target.RPC.Show.Red.VirtualRouters.Backup.Status.Activity == "Local Active" {
		redundancyState = 1
	} else {
		redundancyState = 0
	}
	ch <- semp.NewMetric(MetricDesc["Redundancy"]["system_redundancy_local_active"], prometheus.GaugeValue, redundancyState, mateRouterName)

	return 1, nil
}
