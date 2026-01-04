package semp

import (
	"encoding/xml"
	"solace_exporter/internal/semp/types"

	"github.com/go-kit/log/level"
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
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	command := "<rpc><show><client-profile><name>*</name><vpn-name>" + vpnFilter + "</vpn-name><detail/></client-profile></show></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "DiskSemp1", 1)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't scrape ClientProfiles", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't decode Xml ClientProfiles", "err", err, "broker", semp.brokerURI)
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
