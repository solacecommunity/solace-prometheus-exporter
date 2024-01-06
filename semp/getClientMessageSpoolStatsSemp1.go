package semp

import (
	"encoding/xml"
	"errors"
	"strconv"

	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Get some statistics for each individual client of all vpn's
// This can result in heavy system load for lots of clients
func (e *Semp) GetClientMessageSpoolStatsSemp1(ch chan<- prometheus.Metric, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Client struct {
					PrimaryVirtualRouter struct {
						Client []struct {
							ClientName        string  `xml:"name"`
							ClientUsername    string  `xml:"client-username"`
							MsgVpnName        string  `xml:"message-vpn"`
							ClientProfile     string  `xml:"profile"`
							AclProfile        string  `xml:"acl-profile"`
							SlowSubscriber    bool    `xml:"slow-subscriber"`
							ElidingTopics     float64 `xml:"eliding-topics"`
							FlowsIngress      float64 `xml:"total-ingress-flows"`
							FlowsEgress       float64 `xml:"total-egress-flows"`
							MessageSpoolStats struct {
								IngressFlowStats []struct {
									SpoolingNotReady               float64 `xml:"spooling-not-ready"`
									OutOfOrderMessagesReceived     float64 `xml:"out-of-order-messages-received"`
									DuplicateMessagesReceived      float64 `xml:"duplicate-messages-received"`
									NoEligibleDestinations         float64 `xml:"no-eligible-destinations"`
									GuaranteedMessages             float64 `xml:"guaranteed-messages"`
									NoLocalDelivery                float64 `xml:"no-local-delivery"`
									SeqNumRollover                 float64 `xml:"seq-num-rollover"`
									SeqNumMessagesDiscarded        float64 `xml:"seq-num-messages-discarded"`
									TransactedMessagesNotSequenced float64 `xml:"transacted-messages-not-sequenced"`
									DestinationGroupError          float64 `xml:"destination-group-error"`
									SmfTtlExceeded                 float64 `xml:"smf-ttl-exceeded"`
									PublishAclDenied               float64 `xml:"publish-acl-denied"`
									WindowSize                     float64 `xml:"window-size"`
								} `xml:"ingress-flow-stats>ingress-flow-stat"`
								EgressFlowStats []struct {
									WindowSize                        float64 `xml:"window-size"`
									UsedWindow                        float64 `xml:"used-window"`
									WindowClosed                      float64 `xml:"window-closed"`
									MessageRedelivered                float64 `xml:"message-redelivered"`
									MessageTransportRetransmit        float64 `xml:"message-transport-retransmit"`
									MessageConfirmedDelivered         float64 `xml:"message-confirmed-delivered"`
									ConfirmedDeliveredStoreAndForward float64 `xml:"confirmed-delivered-store-and-forward"`
									ConfirmedDeliveredCutThrough      float64 `xml:"confirmed-delivered-cut-through"`
									UnackedMessages                   float64 `xml:"unacked-messages"`
								} `xml:"egress-flow-stats>egress-flow-stat"`
							} `xml:"message-spool-stats"`
						} `xml:"client"`
					} `xml:",any"`
				} `xml:"client"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	var page = 1
	for nextRequest := "<rpc><show><client><name>" + itemFilter + "</name><message-spool-stats/></client></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", nextRequest, "ClientMessageSpoolStatsSemp1", page)
		page++

		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape ClientMessageSpoolStatsSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode ClientMessageSpoolStatsSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
			return 0, errors.New("unexpected result: see log")
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, client := range target.RPC.Show.Client.PrimaryVirtualRouter.Client {
			ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["client_flows_ingress"], prometheus.GaugeValue, client.FlowsIngress, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile)
			ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["client_flows_egress"], prometheus.GaugeValue, client.FlowsEgress, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile)
			ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["client_slow_subscriber"], prometheus.GaugeValue, encodeMetricBool(client.SlowSubscriber), client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile)

			for flowId, ingressFlow := range client.MessageSpoolStats.IngressFlowStats {
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["spooling_not_ready"], prometheus.CounterValue, ingressFlow.SpoolingNotReady, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["out_of_order_messages_received"], prometheus.CounterValue, ingressFlow.OutOfOrderMessagesReceived, client.MsgVpnName, client.ClientName, client.ClientProfile, client.AclProfile, client.ClientUsername, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["duplicate_messages_received"], prometheus.CounterValue, ingressFlow.DuplicateMessagesReceived, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["no_eligible_destinations"], prometheus.CounterValue, ingressFlow.NoEligibleDestinations, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["guaranteed_messages"], prometheus.CounterValue, ingressFlow.GuaranteedMessages, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["no_local_delivery"], prometheus.CounterValue, ingressFlow.NoLocalDelivery, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["seq_num_rollover"], prometheus.CounterValue, ingressFlow.SeqNumRollover, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["seq_num_messages_discarded"], prometheus.CounterValue, ingressFlow.SeqNumMessagesDiscarded, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["transacted_messages_not_sequenced"], prometheus.CounterValue, ingressFlow.TransactedMessagesNotSequenced, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["destination_group_error"], prometheus.CounterValue, ingressFlow.DestinationGroupError, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["smf_ttl_exceeded"], prometheus.CounterValue, ingressFlow.SmfTtlExceeded, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["publish_acl_denied"], prometheus.CounterValue, ingressFlow.PublishAclDenied, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["ingress_window_size"], prometheus.CounterValue, ingressFlow.WindowSize, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
			}
			for flowId, egressFlow := range client.MessageSpoolStats.EgressFlowStats {
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["egress_window_size"], prometheus.CounterValue, egressFlow.WindowSize, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["used_window"], prometheus.CounterValue, egressFlow.UsedWindow, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["window_closed"], prometheus.CounterValue, egressFlow.WindowClosed, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["message_redelivered"], prometheus.CounterValue, egressFlow.MessageRedelivered, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["message_transport_retransmit"], prometheus.CounterValue, egressFlow.MessageTransportRetransmit, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["message_confirmed_delivered"], prometheus.CounterValue, egressFlow.MessageConfirmedDelivered, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["confirmed_delivered_store_and_forward"], prometheus.CounterValue, egressFlow.ConfirmedDeliveredStoreAndForward, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["confirmed_delivered_cut_through"], prometheus.CounterValue, egressFlow.ConfirmedDeliveredCutThrough, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
				ch <- prometheus.MustNewConstMetric(MetricDesc["ClientMessageSpoolStats"]["unacked_messages"], prometheus.CounterValue, egressFlow.UnackedMessages, client.MsgVpnName, client.ClientName, client.ClientUsername, client.ClientProfile, client.AclProfile, strconv.Itoa(flowId))
			}
		}

		body.Close()
	}

	return 1, nil
}
