package semp

import (
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetConfigSyncSemp1 Sync Status for Broker and Vpn
func (semp *Semp) GetConfigSyncSemp1(ch chan<- PrometheusMetric) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				ConfigSync struct {
					Status struct {
						AdminStatus string `xml:"admin-status"`
						OperStatus  string `xml:"oper-status"`
					} `xml:"status"`
				} `xml:"config-sync"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	command := "<rpc><show><config-sync></config-sync></show></rpc>"
	var target Data
	if up, err := semp.fetchSemp1(command, "ConfigSyncSemp1", 1, &target); err != nil {
		return up, err
	}

	ch <- semp.NewMetric(MetricDesc["ConfigSync"]["configsync_admin_state"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.ConfigSync.Status.AdminStatus, []string{"Shutdown", "Enabled"}))
	ch <- semp.NewMetric(MetricDesc["ConfigSync"]["configsync_oper_state"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.ConfigSync.Status.OperStatus, []string{"Down", "Up", "Shutting Down"}))

	return 1, nil
}
