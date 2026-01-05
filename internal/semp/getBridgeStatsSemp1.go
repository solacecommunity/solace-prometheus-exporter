package semp

import (
	"encoding/xml"
	"solace_exporter/internal/semp/types"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// GetBridgeStatsSemp1 statistics of bridges for all VPNs
func (semp *Semp) GetBridgeStatsSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Bridge struct {
					Bridges struct {
						Bridge []struct {
							BridgeName                string `xml:"bridge-name"`
							LocalVpnName              string `xml:"local-vpn-name"`
							ConnectedRemoteVpnName    string `xml:"connected-remote-vpn-name"`
							ConnectedRemoteRouterName string `xml:"connected-remote-router-name"`
							ConnectedViaAddr          string `xml:"connected-via-addr"`
							ConnectedViaInterface     string `xml:"connected-via-interface"`
							Redundancy                string `xml:"redundancy"`
							AdminState                string `xml:"admin-state"`
							ConnectionEstablisher     string `xml:"connection-establisher"`
							Client                    struct {
								ClientAddress    string  `xml:"client-address"`
								Name             string  `xml:"name"`
								NumSubscriptions float64 `xml:"num-subscriptions"`
								ClientID         float64 `xml:"client-id"`
								MessageVpn       string  `xml:"message-vpn"`
								SlowSubscriber   bool    `xml:"slow-subscriber"`
								ClientUsername   string  `xml:"client-username"`
								Stats            struct {
									TotalClientMessagesReceived         float64 `xml:"total-client-messages-received"`
									TotalClientMessagesSent             float64 `xml:"total-client-messages-sent"`
									ClientDataMessagesReceived          float64 `xml:"client-data-messages-received"`
									ClientDataMessagesSent              float64 `xml:"client-data-messages-sent"`
									ClientPersistentMessagesReceived    float64 `xml:"client-persistent-messages-received"`
									ClientPersistentMessagesSent        float64 `xml:"client-persistent-messages-sent"`
									ClientNonPersistentMessagesReceived float64 `xml:"client-non-persistent-messages-received"`
									ClientNonPersistentMessagesSent     float64 `xml:"client-non-persistent-messages-sent"`
									ClientDirectMessagesReceived        float64 `xml:"client-direct-messages-received"`
									ClientDirectMessagesSent            float64 `xml:"client-direct-messages-sent"`

									TotalClientBytesReceived         float64 `xml:"total-client-bytes-received"`
									TotalClientBytesSent             float64 `xml:"total-client-bytes-sent"`
									ClientDataBytesReceived          float64 `xml:"client-data-bytes-received"`
									ClientDataBytesSent              float64 `xml:"client-data-bytes-sent"`
									ClientPersistentBytesReceived    float64 `xml:"client-persistent-bytes-received"`
									ClientPersistentBytesSent        float64 `xml:"client-persistent-bytes-sent"`
									ClientNonPersistentBytesReceived float64 `xml:"client-non-persistent-bytes-received"`
									ClientNonPersistentBytesSent     float64 `xml:"client-non-persistent-bytes-sent"`
									ClientDirectBytesReceived        float64 `xml:"client-direct-bytes-received"`
									ClientDirectBytesSent            float64 `xml:"client-direct-bytes-sent"`

									LargeMessagesReceived       float64 `xml:"large-messages-received"`
									DeniedDuplicateClients      float64 `xml:"denied-duplicate-clients"`
									NotEnoughSpaceMsgsSent      float64 `xml:"not-enough-space-msgs-sent"`
									MaxExceededMsgsSent         float64 `xml:"max-exceeded-msgs-sent"`
									SubscribeClientNotFound     float64 `xml:"subscribe-client-not-found"`
									NotFoundMsgsSent            float64 `xml:"not-found-msgs-sent"`
									CurrentIngressRatePerSecond float64 `xml:"current-ingress-rate-per-second"`
									CurrentEgressRatePerSecond  float64 `xml:"current-egress-rate-per-second"`
									IngressDiscards             struct {
										TotalIngressDiscards       float64 `xml:"total-ingress-discards"`
										NoSubscriptionMatch        float64 `xml:"no-subscription-match"`
										TopicParseError            float64 `xml:"topic-parse-error"`
										ParseError                 float64 `xml:"parse-error"`
										MsgTooBig                  float64 `xml:"msg-too-big"`
										TTLExceeded                float64 `xml:"ttl-exceeded"`
										WebParseError              float64 `xml:"web-parse-error"`
										PublishTopicACL            float64 `xml:"publish-topic-acl"`
										MsgSpoolDiscards           float64 `xml:"msg-spool-discards"`
										MessagePromotionCongestion float64 `xml:"message-promotion-congestion"`
										MessageSpoolCongestion     float64 `xml:"message-spool-congestion"`
									} `xml:"ingress-discards"`
									EgressDiscards struct {
										TotalEgressDiscards        float64 `xml:"total-egress-discards"`
										TransmitCongestion         float64 `xml:"transmit-congestion"`
										CompressionCongestion      float64 `xml:"compression-congestion"`
										MessageElided              float64 `xml:"message-elided"`
										TTLExceeded                float64 `xml:"ttl-exceeded"`
										PayloadCouldNotBeFormatted float64 `xml:"payload-could-not-be-formatted"`
										MessagePromotionCongestion float64 `xml:"message-promotion-congestion"`
										MessageSpoolCongestion     float64 `xml:"message-spool-congestion"`
										ClientNotConnected         float64 `xml:"client-not-connected"`
									} `xml:"egress-discards"`
									ManagedSubscriptions struct {
										AddBySubscriptionManager    float64 `xml:"add-by-subscription-manager"`
										RemoveBySubscriptionManager float64 `xml:"remove-by-subscription-manager"`
									} `xml:"managed-subscriptions"`
								} `xml:"stats"`
							} `xml:"client"`
						} `xml:"bridge"`
					} `xml:"bridges"`
				} `xml:"bridge"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	command := "<rpc><show><bridge><bridge-name-pattern>" + itemFilter + "</bridge-name-pattern><vpn-name-pattern>" + vpnFilter + "</vpn-name-pattern><stats/></bridge></show></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "BridgeStatsSemp1", 1)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't scrape BridgeSemp1", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't decode Xml BridgeSemp1", "err", err, "broker", semp.brokerURI)
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
	for _, bridge := range target.RPC.Show.Bridge.Bridges.Bridge {
		bridgeName := bridge.BridgeName
		vpnName := bridge.LocalVpnName
		remoteRouterName := bridge.ConnectedRemoteRouterName
		remoteVpnName := bridge.ConnectedRemoteVpnName
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_num_subscriptions"], prometheus.GaugeValue, bridge.Client.NumSubscriptions, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_slow_subscriber"], prometheus.GaugeValue, encodeMetricBool(bridge.Client.SlowSubscriber), vpnName, bridgeName, remoteRouterName, remoteVpnName)

		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_total_client_messages_received"], prometheus.CounterValue, bridge.Client.Stats.TotalClientMessagesReceived, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_total_client_messages_sent"], prometheus.CounterValue, bridge.Client.Stats.TotalClientMessagesSent, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_data_messages_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientDataMessagesReceived, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_data_messages_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientDataMessagesSent, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_persistent_messages_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientPersistentMessagesReceived, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_persistent_messages_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientPersistentMessagesSent, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_nonpersistent_messages_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientNonPersistentMessagesReceived, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_nonpersistent_messages_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientNonPersistentMessagesSent, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_direct_messages_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientDirectMessagesReceived, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_direct_messages_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientDirectMessagesSent, vpnName, bridgeName, remoteRouterName, remoteVpnName)

		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_total_client_bytes_received"], prometheus.CounterValue, bridge.Client.Stats.TotalClientBytesReceived, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_total_client_bytes_sent"], prometheus.CounterValue, bridge.Client.Stats.TotalClientBytesSent, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_data_bytes_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientDataBytesReceived, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_data_bytes_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientDataBytesSent, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_persistent_bytes_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientPersistentBytesReceived, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_persistent_bytes_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientPersistentBytesSent, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_nonpersistent_bytes_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientNonPersistentBytesReceived, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_nonpersistent_bytes_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientNonPersistentBytesSent, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_direct_bytes_received"], prometheus.GaugeValue, bridge.Client.Stats.ClientDirectBytesReceived, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_direct_bytes_sent"], prometheus.GaugeValue, bridge.Client.Stats.ClientDirectBytesSent, vpnName, bridgeName, remoteRouterName, remoteVpnName)

		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_client_large_messages_received"], prometheus.GaugeValue, bridge.Client.Stats.LargeMessagesReceived, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_denied_duplicate_clients"], prometheus.GaugeValue, bridge.Client.Stats.DeniedDuplicateClients, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_not_enough_space_msgs_sent"], prometheus.GaugeValue, bridge.Client.Stats.NotEnoughSpaceMsgsSent, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_max_exceeded_msgs_sent"], prometheus.GaugeValue, bridge.Client.Stats.MaxExceededMsgsSent, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_subscribe_client_not_found"], prometheus.GaugeValue, bridge.Client.Stats.SubscribeClientNotFound, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_not_found_msgs_sent"], prometheus.GaugeValue, bridge.Client.Stats.NotFoundMsgsSent, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_current_ingress_rate_per_second"], prometheus.GaugeValue, bridge.Client.Stats.CurrentIngressRatePerSecond, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_current_egress_rate_per_second"], prometheus.GaugeValue, bridge.Client.Stats.CurrentEgressRatePerSecond, vpnName, bridgeName, remoteRouterName, remoteVpnName)

		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_total_ingress_discards"], prometheus.CounterValue, bridge.Client.Stats.IngressDiscards.TotalIngressDiscards, vpnName, bridgeName, remoteRouterName, remoteVpnName)
		ch <- semp.NewMetric(MetricDesc["BridgeStats"]["bridge_total_egress_discards"], prometheus.CounterValue, bridge.Client.Stats.EgressDiscards.TotalEgressDiscards, vpnName, bridgeName, remoteRouterName, remoteVpnName)
	}
	return 1, nil
}
