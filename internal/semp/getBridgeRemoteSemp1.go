package semp

import (
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetBridgeRemoteSemp1 Get status of bridges for all VPNs
// Same as GetBridge but adds labels for remote VPN and remote router
func (semp *Semp) GetBridgeRemoteSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Bridge struct {
					Bridges struct {
						NumTotalBridgesValue                 float64 `xml:"num-total-bridges"`
						MaxNumTotalBridgesValue              float64 `xml:"max-num-total-bridges"`
						NumLocalBridgesValue                 float64 `xml:"num-local-bridges"`
						MaxNumLocalBridgesValue              float64 `xml:"max-num-local-bridges"`
						NumRemoteBridgesValue                float64 `xml:"num-remote-bridges"`
						MaxNumRemoteBridgesValue             float64 `xml:"max-num-remote-bridges"`
						NumTotalRemoteBridgeSubscriptions    float64 `xml:"num-total-remote-bridge-subscriptions"`
						MaxNumTotalRemoteBridgeSubscriptions float64 `xml:"max-num-total-remote-bridge-subscriptions"`
						Bridge                               []struct {
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
						} `xml:"bridge"`
					} `xml:"bridges"`
				} `xml:"bridge"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	command := "<rpc><show><bridge><bridge-name-pattern>" + semp1Escape(itemFilter) + "</bridge-name-pattern><vpn-name-pattern>" + semp1Escape(vpnFilter) + "</vpn-name-pattern></bridge></show></rpc>"
	var target Data
	if up, err := semp.fetchSemp1(command, "BridgeRemoteSemp1", 1, &target); err != nil {
		return up, err
	}
	ch <- semp.NewMetric(MetricDesc["Bridge"]["bridges_num_total_bridges"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.NumTotalBridgesValue)
	ch <- semp.NewMetric(MetricDesc["Bridge"]["bridges_max_num_total_bridges"], prometheus.CounterValue, target.RPC.Show.Bridge.Bridges.MaxNumTotalBridgesValue)
	ch <- semp.NewMetric(MetricDesc["Bridge"]["bridges_num_local_bridges"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.NumLocalBridgesValue)
	ch <- semp.NewMetric(MetricDesc["Bridge"]["bridges_max_num_local_bridges"], prometheus.CounterValue, target.RPC.Show.Bridge.Bridges.MaxNumLocalBridgesValue)
	ch <- semp.NewMetric(MetricDesc["Bridge"]["bridges_num_remote_bridges"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.NumRemoteBridgesValue)
	ch <- semp.NewMetric(MetricDesc["Bridge"]["bridges_max_num_remote_bridges"], prometheus.CounterValue, target.RPC.Show.Bridge.Bridges.MaxNumRemoteBridgesValue)
	ch <- semp.NewMetric(MetricDesc["Bridge"]["bridges_num_total_remote_bridge_subscriptions"], prometheus.GaugeValue, target.RPC.Show.Bridge.Bridges.NumTotalRemoteBridgeSubscriptions)
	ch <- semp.NewMetric(MetricDesc["Bridge"]["bridges_max_num_total_remote_bridge_subscriptions"], prometheus.CounterValue, target.RPC.Show.Bridge.Bridges.MaxNumTotalRemoteBridgeSubscriptions)
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
		remoteVpnName := bridge.ConnectedRemoteVpnName
		remoteRouter := bridge.ConnectedRemoteRouterName
		ch <- semp.NewMetric(MetricDesc["BridgeRemote"]["bridge_remote_admin_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.AdminState, []string{"Enabled", "Disabled", "-", "N/A"}), vpnName, bridgeName, remoteVpnName, remoteRouter)
		ch <- semp.NewMetric(MetricDesc["BridgeRemote"]["bridge_remote_connection_establisher"], prometheus.GaugeValue, encodeMetricMulti(bridge.ConnectionEstablisher, []string{"NotApplicable", "Local", "Remote", "Invalid"}), vpnName, bridgeName, remoteVpnName, remoteRouter)
		ch <- semp.NewMetric(MetricDesc["BridgeRemote"]["bridge_remote_inbound_operational_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.InboundOperationalState, opStates), vpnName, bridgeName, remoteVpnName, remoteRouter)
		ch <- semp.NewMetric(MetricDesc["BridgeRemote"]["bridge_remote_inbound_operational_failure_reason"], prometheus.GaugeValue, encodeMetricMulti(bridge.InboundOperationalFailureReason, failReasons), vpnName, bridgeName, remoteVpnName, remoteRouter)
		ch <- semp.NewMetric(MetricDesc["BridgeRemote"]["bridge_remote_outbound_operational_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.OutboundOperationalState, opStates), vpnName, bridgeName, remoteVpnName, remoteRouter)
		ch <- semp.NewMetric(MetricDesc["BridgeRemote"]["bridge_remote_queue_operational_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.QueueOperationalState, []string{"NotApplicable", "Bound", "Unbound"}), vpnName, bridgeName, remoteVpnName, remoteRouter)
		ch <- semp.NewMetric(MetricDesc["BridgeRemote"]["bridge_remote_redundancy"], prometheus.GaugeValue, encodeMetricMulti(bridge.Redundancy, []string{"NotApplicable", "auto", "primary", "backup", "static", "none"}), vpnName, bridgeName, remoteVpnName, remoteRouter)
		ch <- semp.NewMetric(MetricDesc["BridgeRemote"]["bridge_remote_connection_uptime_in_seconds"], prometheus.GaugeValue, bridge.ConnectionUptimeInSeconds, vpnName, bridgeName, remoteVpnName, remoteRouter)
	}
	return 1, nil
}
