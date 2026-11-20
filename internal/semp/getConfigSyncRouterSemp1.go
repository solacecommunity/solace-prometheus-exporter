package semp

import (
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// Config Sync Status for Broker and Vpn
func (semp *Semp) GetConfigSyncRouterSemp1(ch chan<- PrometheusMetric) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				ConfigSync struct {
					Database struct {
						Local struct {
							Tables struct {
								Table []struct {
									Type               string  `xml:"type"`
									TimeInStateSeconds float64 `xml:"time-in-state-seconds"`
									Name               string  `xml:"name"`
									Ownership          string  `xml:"ownership"`
									SyncState          string  `xml:"sync-state"`
									TimeInState        string  `xml:"time-in-state"`
								} `xml:"table"`
							} `xml:"tables"`
						} `xml:"local"`
					} `xml:"database"`
				} `xml:"config-sync"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	command := "<rpc><show><config-sync><database/><router/></config-sync></show></rpc>"
	var target Data
	if up, err := semp.fetchSemp1(command, "ConfigSyncRouterSemp1", 1, &target); err != nil {
		return up, err
	}

	for _, table := range target.RPC.Show.ConfigSync.Database.Local.Tables.Table {
		ch <- semp.NewMetric(MetricDesc["ConfigSyncRouter"]["configsync_table_type"], prometheus.GaugeValue, encodeMetricMulti(table.Type, []string{"Router", "Vpn", "Unknown", "None", "All"}), table.Name)
		ch <- semp.NewMetric(MetricDesc["ConfigSyncRouter"]["configsync_table_timeinstateseconds"], prometheus.CounterValue, table.TimeInStateSeconds, table.Name)
		ch <- semp.NewMetric(MetricDesc["ConfigSyncRouter"]["configsync_table_ownership"], prometheus.GaugeValue, encodeMetricMulti(table.Ownership, []string{"Master", "Slave", "Unknown"}), table.Name)
		ch <- semp.NewMetric(MetricDesc["ConfigSyncRouter"]["configsync_table_syncstate"], prometheus.GaugeValue, encodeMetricMulti(table.SyncState, []string{"Down", "Up", "Unknown", "In-Sync", "Reconciling", "Blocked", "Out-Of-Sync"}), table.Name)
	}

	return 1, nil
}
