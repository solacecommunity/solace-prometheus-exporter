package semp

import (
	"fmt"
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetConfigSyncVpnSemp1 Sync Status for Broker and Vpn
func (semp *Semp) GetConfigSyncVpnSemp1(ch chan<- PrometheusMetric, vpnFilter string, sempPageSize int64) (float64, error) {
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

	var lastTableName = ""
	command := fmt.Sprintf("<rpc><show><config-sync><database/><message-vpn/><vpn-name>%s</vpn-name><count/><num-elements>%d</num-elements></config-sync></show></rpc>", semp1Escape(vpnFilter), sempPageSize)
	return fetchSemp1Pages(semp, command, "ConfigSyncVpnSemp1", func(target *Data) {
		for _, table := range target.RPC.Show.ConfigSync.Database.Local.Tables.Table {
			tableKey := table.Name
			if tableKey == lastTableName {
				continue
			}
			lastTableName = tableKey
			ch <- semp.NewMetric(MetricDesc["ConfigSyncVpn"]["configsync_table_type"], prometheus.GaugeValue, encodeMetricMulti(table.Type, []string{"Router", "Vpn", "Unknown", "None", "All"}), table.Name)
			ch <- semp.NewMetric(MetricDesc["ConfigSyncVpn"]["configsync_table_timeinstateseconds"], prometheus.CounterValue, table.TimeInStateSeconds, table.Name)
			ch <- semp.NewMetric(MetricDesc["ConfigSyncVpn"]["configsync_table_ownership"], prometheus.GaugeValue, encodeMetricMulti(table.Ownership, []string{"Master", "Slave", "Unknown"}), table.Name)
			ch <- semp.NewMetric(MetricDesc["ConfigSyncVpn"]["configsync_table_syncstate"], prometheus.GaugeValue, encodeMetricMulti(table.SyncState, []string{"Down", "Up", "Unknown", "In-Sync", "Reconciling", "Blocked", "Out-Of-Sync"}), table.Name)
		}
	})
}
