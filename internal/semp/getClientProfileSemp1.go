package semp

import (
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetDiskSemp1 Get system disk information (for Appliance)
func (semp *Semp) GetClientProfileSemp1(ch chan<- PrometheusMetric, vpnFilter string) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				ClientProfile struct {
					Profiles struct {
						Profile []struct {
							Name                            string  `xml:"name"`
							MsgVpnName                      string  `xml:"message-vpn"`
							NumUsers                        float64 `xml:"num-users"`
							MaxConnectionsPerClientUsername float64 `xml:"max-connections-per-client-username"`
							MaxEndpointsPerClientUsername   float64 `xml:"maximum-endpoints-per-client-username-effective"`
							MaxEgressFlows                  float64 `xml:"maximum-egress-flows-effective"`
							MaxIngressFlows                 float64 `xml:"maximum-ingress-flows-effective"`
							MaxTransactedSessions           float64 `xml:"maximum-transacted-sessions-effective"`
							MaxSubscriptions                float64 `xml:"max-subscriptions-effective"`
						} `xml:"profile"`
					} `xml:"profiles"`
				} `xml:"client-profile"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	command := "<rpc><show><client-profile><name>*</name><vpn-name>" + semp1Escape(vpnFilter) + "</vpn-name><detail/></client-profile></show></rpc>"
	var target Data
	if up, err := semp.fetchSemp1(command, "DiskSemp1", 1, &target); err != nil {
		return up, err
	}

	for _, clientProfile := range target.RPC.Show.ClientProfile.Profiles.Profile {
		ch <- semp.NewMetric(MetricDesc["ClientProfile"]["clientprofile_max_connections_per_username"], prometheus.GaugeValue, clientProfile.MaxConnectionsPerClientUsername, clientProfile.MsgVpnName, clientProfile.Name)
		ch <- semp.NewMetric(MetricDesc["ClientProfile"]["clientprofile_max_endpoints_per_username"], prometheus.GaugeValue, clientProfile.MaxEndpointsPerClientUsername, clientProfile.MsgVpnName, clientProfile.Name)
		ch <- semp.NewMetric(MetricDesc["ClientProfile"]["clientprofile_max_egress_flows"], prometheus.GaugeValue, clientProfile.MaxEgressFlows, clientProfile.MsgVpnName, clientProfile.Name)
		ch <- semp.NewMetric(MetricDesc["ClientProfile"]["clientprofile_max_ingress_flows"], prometheus.GaugeValue, clientProfile.MaxIngressFlows, clientProfile.MsgVpnName, clientProfile.Name)
		ch <- semp.NewMetric(MetricDesc["ClientProfile"]["clientprofile_max_transacted_sessions_per_client"], prometheus.GaugeValue, clientProfile.MaxTransactedSessions, clientProfile.MsgVpnName, clientProfile.Name)
		ch <- semp.NewMetric(MetricDesc["ClientProfile"]["clientprofile_max_subscriptions"], prometheus.GaugeValue, clientProfile.MaxSubscriptions, clientProfile.MsgVpnName, clientProfile.Name)
		ch <- semp.NewMetric(MetricDesc["ClientProfile"]["clientprofile_num_users"], prometheus.GaugeValue, clientProfile.NumUsers, clientProfile.MsgVpnName, clientProfile.Name)
	}

	return 1, nil
}
