package semp

import (
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
						Protocol           string `xml:"protocol" optional:"yes"`
						AdminState         bool   `xml:"admin-state" optional:"yes"`
						NTPServerAddr      string `xml:"ntp-server-address" optional:"yes"`
						NTPServerReachable bool   `xml:"ntp-server-reachable" optional:"yes"`
					} `xml:"detail"`
				} `xml:"clock"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	command := "<rpc><show><clock><detail/></clock></show></rpc>"
	var target Data
	if up, err := semp.fetchSemp1(command, "ClockDetailSemp1", 1, &target); err != nil {
		return up, err
	}

	ch <- semp.NewMetric(MetricDesc["ClockDetail"]["system_clock_detail_admin_state"], prometheus.GaugeValue, encodeMetricBool(target.RPC.Show.Clock.Detail.AdminState), target.RPC.Show.Clock.Detail.Protocol)
	ch <- semp.NewMetric(MetricDesc["ClockDetail"]["system_clock_detail_ntp_server_reachable"], prometheus.GaugeValue, encodeMetricBool(target.RPC.Show.Clock.Detail.NTPServerReachable), target.RPC.Show.Clock.Detail.NTPServerAddr)

	return 1, nil
}
