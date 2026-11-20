package semp

import (
	"fmt"
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetBridgeDetailSemp1 Get status of bridges for all VPNs
func (semp *Semp) GetBridgeDetailSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string, sempPageSize int64) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Bridge struct {
					Bridges struct {
						Bridge []struct {
							BridgeName                      string  `xml:"bridge-name"`
							LocalVpnName                    string  `xml:"local-vpn-name"`
							ConnectedRemoteVpnName          string  `xml:"connected-remote-vpn-name"`
							ConnectedRemoteRouterName       string  `xml:"connected-remote-router-name"`
							AdminState                      string  `xml:"admin-state"`
							ConnectionEstablisher           string  `xml:"connection-establisher"`
							InboundOperationalState         string  `xml:"inbound-operational-state"`
							InboundOperationalFailureReason string  `xml:"inbound-operational-failure-reason"`
							OutboundOperationalState        string  `xml:"outbound-operational-state"`
							QueueOperationalState           string  `xml:"queue-operational-state"`
							Redundancy                      string  `xml:"redundancy"`
							ConnectionUptimeInSeconds       float64 `xml:"connection-uptime-in-seconds"`
							Authentication                  struct {
								AuthScheme string `xml:"auth-scheme"`
								Basic      struct {
									ClientUsername     string `xml:"client-username"`
									PasswordConfigured string `xml:"password-configured"`
								} `xml:"basic"`
								ClientCertificate struct {
									CertificateFile        string `xml:"certificate-file"`
									UsingServerCertificate bool   `xml:"using-server-certificate"`
								} `xml:"client-certificate"`
							} `xml:"authentication"`
							LocalQueueName       string `xml:"local-queue-name"`
							RemoteMessageVPNList struct {
								RemoteMessageVPN []struct {
									VpnName                     string `xml:"vpn-name"`
									RouterName                  string `xml:"router-name"`
									AdminState                  string `xml:"admin-state"`
									Compressed                  string `xml:"compressed"`
									SSL                         string `xml:"ssl"`
									ConnectionState             string `xml:"connection-state"`
									LastConnectionFailureReason string `xml:"last-connection-failure-reason"`
									QueueName                   string `xml:"queue-name"`
									QueueBindState              string `xml:"queue-bind-state"`
								} `xml:"remote-message-vpn"`
							} `xml:"remote-message-vpn-list"`
						} `xml:"bridge"`
					} `xml:"bridges"`
				} `xml:"bridge"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	var lastBridgeName = ""
	command := fmt.Sprintf("<rpc><show><bridge><bridge-name-pattern>%s</bridge-name-pattern><vpn-name-pattern>%s</vpn-name-pattern><detail/><count/><num-elements>%d</num-elements></bridge></show></rpc>", semp1Escape(itemFilter), semp1Escape(vpnFilter), sempPageSize)
	return fetchSemp1Pages(semp, command, "BridgeDetailSemp1", func(target *Data) {
		opStates := []string{"Init", "Shutdown", "NoShutdown", "Prepare", "Prepare-WaitToConnect",
			"Prepare-FetchingDNS", "NotReady", "NotReady-Connecting", "NotReady-Handshaking", "NotReady-WaitNext",
			"NotReady-WaitReuse", "NotRead-WaitBridgeVersionMismatch", "NotReady-WaitCleanup", "Ready", "Ready-Subscribing",
			"Ready-InSync", "NotApplicable", "Invalid"}
		failReasons := []string{"Bridge disabled", "No remote message-vpns configured", "SMF service is disabled", "Msg Backbone is disabled",
			"Local message-vpn is disabled", "Active-Standby Role Mismatch", "Invalid Active-Standby Role", "Redundancy Disabled", "Not active",
			"Replication standby", "Remote message-vpns disabled", "Enforce-trusted-common-name but empty trust-common-name list", "SSL transport used but cipher-suite list is empty", "Authentication Scheme is Client-Certificate but no certificate is configured",
			"Client-Certificate Authentication Scheme used but not all Remote Message VPNs use SSL", "Basic Authentication Scheme used but Basic Client Username not configured", "Cluster Down", "Cluster Link Down", ""}
		for _, bridge := range target.RPC.Show.Bridge.Bridges.Bridge {
			bridgeName := bridge.BridgeName
			vpnName := bridge.LocalVpnName
			connectedRemoteVpnName := bridge.ConnectedRemoteVpnName
			connectedRemoteRouter := bridge.ConnectedRemoteRouterName
			localQueueName := bridge.LocalQueueName
			bridgeKey := vpnName + "___" + bridgeName
			if bridgeKey == lastBridgeName {
				continue
			}
			lastBridgeName = bridgeKey
			ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_admin_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.AdminState, []string{"Enabled", "Disabled", "-", "N/A"}), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName)
			ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_connection_establisher"], prometheus.GaugeValue, encodeMetricMulti(bridge.ConnectionEstablisher, []string{"NotApplicable", "Local", "Remote", "Invalid"}), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName)
			ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_inbound_operational_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.InboundOperationalState, opStates), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName)
			ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_inbound_operational_failure_reason"], prometheus.GaugeValue, encodeMetricMulti(bridge.InboundOperationalFailureReason, failReasons), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName)
			ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_outbound_operational_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.OutboundOperationalState, opStates), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName)
			ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_queue_operational_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.QueueOperationalState, []string{"NotApplicable", "Bound", "Unbound"}), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName)
			ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_redundancy"], prometheus.GaugeValue, encodeMetricMulti(bridge.Redundancy, []string{"NotApplicable", "auto", "primary", "backup", "static", "none"}), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName)
			ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_connection_uptime_in_seconds"], prometheus.GaugeValue, bridge.ConnectionUptimeInSeconds, vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName)
			ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_authentication_scheme"], prometheus.GaugeValue, encodeMetricMulti(bridge.Authentication.AuthScheme, []string{"NotApplicable", "Basic", "Client-Certificate", "TLS-PSK"}), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName, bridge.Authentication.Basic.ClientUsername, bridge.Authentication.ClientCertificate.CertificateFile)
			for _, remoteVpn := range bridge.RemoteMessageVPNList.RemoteMessageVPN {
				remoteVpnName := remoteVpn.VpnName
				remoteRouter := remoteVpn.RouterName
				if remoteRouter == "" {
					remoteRouter = bridge.ConnectedRemoteRouterName
				}
				compressed := remoteVpn.Compressed
				ssl := remoteVpn.SSL
				remoteQueueName := remoteVpn.QueueName
				ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_remote_admin_state"], prometheus.GaugeValue, encodeMetricMulti(remoteVpn.AdminState, []string{"Enabled", "Disabled", "-", "N/A"}), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName, remoteVpnName, remoteRouter, compressed, ssl, remoteQueueName)
				ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_remote_connection_state"], prometheus.GaugeValue, encodeMetricMulti(remoteVpn.ConnectionState, []string{"Down", "Up"}), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, remoteVpnName, localQueueName, remoteRouter, compressed, ssl, remoteQueueName)
				ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_remote_last_conn_failure_reason"], prometheus.GaugeValue, encodeMetricMulti(remoteVpn.LastConnectionFailureReason, failReasons), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName, remoteVpnName, remoteRouter, compressed, ssl, remoteQueueName)
				ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_remote_queue_bind_state"], prometheus.GaugeValue, encodeMetricMulti(remoteVpn.QueueBindState, []string{"Down", "Up"}), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName, remoteVpnName, remoteRouter, compressed, ssl, remoteQueueName)
			}
		}
	})
}
