package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Get system-wide basic redundancy information for HA triples
func (e *Semp) GetRedundancySemp1(ch chan<- prometheus.Metric) (ok float64, err error) {
	var f float64

	type Data struct {
		RPC struct {
			Show struct {
				Red struct {
					ConfigStatus      string `xml:"config-status"`
					RedundancyStatus  string `xml:"redundancy-status"`
					OperatingMode     string `xml:"operating-mode"`
					ActiveStandbyRole string `xml:"active-standby-role"`
					MateRouterName    string `xml:"mate-router-name"`
					VirtualRouters    struct {
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
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><redundancy/></show></rpc>"
	body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", command)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape RedundancySemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml RedundancySemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
		return 0, errors.New("unexpected result: see log")
	}

	mateRouterName := "" + target.RPC.Show.Red.MateRouterName
	ch <- prometheus.MustNewConstMetric(MetricDesc["Redundancy"]["system_redundancy_config"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Red.ConfigStatus, []string{"Disabled", "Enabled", "Shutdown"}), mateRouterName)
	ch <- prometheus.MustNewConstMetric(MetricDesc["Redundancy"]["system_redundancy_up"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Red.RedundancyStatus, []string{"Down", "Up"}), mateRouterName)
	ch <- prometheus.MustNewConstMetric(MetricDesc["Redundancy"]["system_redundancy_role"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Red.ActiveStandbyRole, []string{"Backup", "Primary", "Monitor", "Undefined"}), mateRouterName)

	if target.RPC.Show.Red.ActiveStandbyRole == "Primary" && target.RPC.Show.Red.VirtualRouters.Primary.Status.Activity == "Local Active" ||
		target.RPC.Show.Red.ActiveStandbyRole == "Backup" && target.RPC.Show.Red.VirtualRouters.Backup.Status.Activity == "Local Active" {
		f = 1
	} else {
		f = 0
	}
	ch <- prometheus.MustNewConstMetric(MetricDesc["Redundancy"]["system_redundancy_local_active"], prometheus.GaugeValue, f, mateRouterName)

	return 1, nil
}
