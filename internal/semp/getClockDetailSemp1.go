package semp

import (
	"encoding/xml"
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetClockDetailSemp1 Clock details for Broker and Vpn
func (semp *Semp) GetClockDetailSemp1(ch chan<- PrometheusMetric) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Clock struct {
					Detail struct {
						Protocol            string  `xml:"protocol" optional:"yes"`
						AdminState          bool    `xml:"admin-state" optional:"yes"`
						NTPServerAddr       string  `xml:"ntp-server-address" optional:"yes"`
						NTPServerReachable  bool    `xml:"ntp-server-reachable" optional:"yes"`
					} `xml:"detail"`
				} `xml:"clock"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	command := "<rpc><show><clock><detail/></clock></show></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "ClockDetailSemp1", 1)
	if err != nil {
		semp.logger.Error("Can't scrape ClockDetailSemp1", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		semp.logger.Error("Can't decode Xml ClockDetailSemp1", "err", err, "broker", semp.brokerURI)
		return 0, err
	}
	if err := target.ExecuteResult.OK(); err != nil {
		semp.logger.Error(
			"unexpected result",
			"command", command,
			"result", target.ExecuteResult.Result,
			"reason", target.ExecuteResult.Reason,
			"broker", semp.brokerURI,
		)
		return 0, err
	}

	ch <- semp.NewMetric(MetricDesc["ClockDetail"]["system_clock_detail_admin_state"], prometheus.GaugeValue, encodeMetricBool(target.RPC.Show.Clock.Detail.AdminState), target.RPC.Show.Clock.Detail.Protocol)
	ch <- semp.NewMetric(MetricDesc["ClockDetail"]["system_clock_detail_ntp_server_reachable"], prometheus.GaugeValue, encodeMetricBool(target.RPC.Show.Clock.Detail.NTPServerReachable), target.RPC.Show.Clock.Detail.NTPServerAddr)

	return 1, nil
}
