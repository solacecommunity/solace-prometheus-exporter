package semp

import (
	"encoding/xml"
	"solace_exporter/internal/semp/types"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// GetBridgeDetailSemp1 Get status of bridges for all VPNs
func (semp *Semp) GetBridgeDetailSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Bridge struct {
					Bridges struct {
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
							LocalQueueName                  string  `xml:"local-queue-name"`
                            RemoteMessageVPNList                struct {
                                RemoteMessageVPN                    []struct {
                                    VpnName                             string  `xml:"vpn-name"`
                                    RouterName                          string  `xml:"router-name"`
                                    AdminState                          string  `xml:"admin-state"`
                                    Compressed                          string  `xml:"compressed"`
                                    SSL                                 string  `xml:"ssl"`
                                    ConnectionState                     string  `xml:"connection-state"`
                                    LastConnectionFailureReason         string  `xml:"last-connection-failure-reason"`
                                    QueueName                           string  `xml:"queue-name"`
                                    QueueBindState                      string  `xml:"queue-bind-state"`
                                } `xml:"remote-message-vpn"`
                            } `xml:"remote-message-vpn-list"`
						} `xml:"bridge"`
					} `xml:"bridges"`
				} `xml:"bridge"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	command := "<rpc><show><bridge><bridge-name-pattern>" + itemFilter + "</bridge-name-pattern><vpn-name-pattern>" + vpnFilter + "</vpn-name-pattern><detail/></bridge></show></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "BridgeDetailSemp1", 1)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't scrape BridgeDetailSemp1", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't decode Xml BridgeDetailSemp1", "err", err, "broker", semp.brokerURI)
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
		ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_admin_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.AdminState, []string{"Enabled", "Disabled", "-", "N/A"}), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName)
		ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_connection_establisher"], prometheus.GaugeValue, encodeMetricMulti(bridge.ConnectionEstablisher, []string{"NotApplicable", "Local", "Remote", "Invalid"}), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName)
		ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_inbound_operational_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.InboundOperationalState, opStates), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName)
		ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_inbound_operational_failure_reason"], prometheus.GaugeValue, encodeMetricMulti(bridge.InboundOperationalFailureReason, failReasons), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName)
		ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_outbound_operational_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.OutboundOperationalState, opStates), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName)
		ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_queue_operational_state"], prometheus.GaugeValue, encodeMetricMulti(bridge.QueueOperationalState, []string{"NotApplicable", "Bound", "Unbound"}), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName)
		ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_redundancy"], prometheus.GaugeValue, encodeMetricMulti(bridge.Redundancy, []string{"NotApplicable", "auto", "primary", "backup", "static", "none"}), vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName)
		ch <- semp.NewMetric(MetricDesc["BridgeDetail"]["bridge_detail_connection_uptime_in_seconds"], prometheus.GaugeValue, bridge.ConnectionUptimeInSeconds, vpnName, bridgeName, connectedRemoteVpnName, connectedRemoteRouter, localQueueName)
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
	return 1, nil
}
