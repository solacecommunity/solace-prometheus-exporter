package semp

import (
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// Replication Config and status
func (semp *Semp) GetVpnReplicationSemp1(ch chan<- PrometheusMetric, vpnFilter string) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				MessageVpn struct {
					Replication struct {
						MessageVpns struct {
							MessageVpn []struct {
								VpnName                    string `xml:"vpn-name"`
								AdminState                 string `xml:"admin-state"`
								ConfigState                string `xml:"config-state"`
								TransactionReplicationMode string `xml:"transaction-replication-mode"`
							} `xml:"message-vpn"`
						} `xml:"message-vpns"`
					} `xml:"replication"`
				} `xml:"message-vpn"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	command := "<rpc><show><message-vpn><vpn-name>" + vpnFilter + "</vpn-name><replication/></message-vpn></show></rpc>"
	var target Data
	if up, err := semp.fetchSemp1(command, "VpnReplicationSemp1", 1, &target); err != nil {
		return up, err
	}

	for _, vpn := range target.RPC.Show.MessageVpn.Replication.MessageVpns.MessageVpn {
		ch <- semp.NewMetric(MetricDesc["VpnReplication"]["vpn_replication_admin_state"], prometheus.GaugeValue, encodeMetricMulti(vpn.AdminState, []string{"shutdown", "enabled", "n/a"}), vpn.VpnName)
		ch <- semp.NewMetric(MetricDesc["VpnReplication"]["vpn_replication_config_state"], prometheus.GaugeValue, encodeMetricMulti(vpn.ConfigState, []string{"standby", "active", "n/a"}), vpn.VpnName)
		ch <- semp.NewMetric(MetricDesc["VpnReplication"]["vpn_replication_transaction_replication_mode"], prometheus.GaugeValue, encodeMetricMulti(vpn.TransactionReplicationMode, []string{"async", "sync", "n/a"}), vpn.VpnName)
	}

	return 1, nil
}
