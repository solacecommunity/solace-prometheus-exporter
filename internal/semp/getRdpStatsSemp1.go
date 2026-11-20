package semp

import (
	"fmt"
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetRdpStatsSemp1 Get rates for each individual queue of all VPNs
// This can result in heavy system load for lots of queues
func (semp *Semp) GetRdpStatsSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string, sempPageSize int64) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				MessageVpn struct {
					Rest struct {
						RestDeliveryPoints struct {
							RestDeliveryPoint []struct {
								RdpName    string `xml:"name"`
								MsgVpnName string `xml:"message-vpn"`
								Stats      struct {
									ReqMsgSent                 float64 `xml:"http-post-request-messages-sent"`
									ReqMsgSentOutstanding      float64 `xml:"http-post-request-messages-sent-outstanding"`
									ReqMsgSentConnectionClosed float64 `xml:"http-post-request-messages-sent-connection-closed"`
									ReqMsgSentTimedOut         float64 `xml:"http-post-request-messages-sent-timed-out"`
									RespMsgReceived            float64 `xml:"http-post-response-messages-received"`
									RespMsgReceivedSuccessful  float64 `xml:"http-post-response-messages-received-successful"`
									RespMsgReceivedError       float64 `xml:"http-post-response-messages-received-error"`
									ReqBytesSent               float64 `xml:"http-post-request-bytes-sent"`
									RespByesReceived           float64 `xml:"http-post-response-bytes-received"`
								} `xml:"stats"`
							} `xml:"rest-delivery-point"`
						} `xml:"rest-delivery-points"`
					} `xml:"rest"`
				} `xml:"message-vpn"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}
	var lastRdpName = ""
	command := fmt.Sprintf("<rpc><show><message-vpn><vpn-name>"+vpnFilter+"</vpn-name><rest></rest><rest-delivery-point></rest-delivery-point><rdp-name>"+itemFilter+"</rdp-name><stats/><count/><num-elements>%d</num-elements></message-vpn></show></rpc>", sempPageSize)
	return fetchSemp1Pages(semp, command, "RdpStatsSemp1", func(target *Data) {
		for _, rdp := range target.RPC.Show.MessageVpn.Rest.RestDeliveryPoints.RestDeliveryPoint {
			rdpKey := rdp.MsgVpnName + "___" + rdp.RdpName
			if rdpKey == lastRdpName {
				continue
			}
			lastRdpName = rdpKey
			ch <- semp.NewMetric(MetricDesc["RdpStats"]["http_post_request_bytes_sent"], prometheus.CounterValue, rdp.Stats.ReqBytesSent, rdp.MsgVpnName, rdp.RdpName)
			ch <- semp.NewMetric(MetricDesc["RdpStats"]["http_post_request_messages_sent"], prometheus.CounterValue, rdp.Stats.ReqMsgSent, rdp.MsgVpnName, rdp.RdpName)
			ch <- semp.NewMetric(MetricDesc["RdpStats"]["http_post_request_messages_sent_connection_closed"], prometheus.CounterValue, rdp.Stats.ReqMsgSentConnectionClosed, rdp.MsgVpnName, rdp.RdpName)
			ch <- semp.NewMetric(MetricDesc["RdpStats"]["http_post_request_messages_sent_outstanding"], prometheus.GaugeValue, rdp.Stats.ReqMsgSentOutstanding, rdp.MsgVpnName, rdp.RdpName)
			ch <- semp.NewMetric(MetricDesc["RdpStats"]["http_post_request_messages_sent_timed_out"], prometheus.CounterValue, rdp.Stats.ReqMsgSentTimedOut, rdp.MsgVpnName, rdp.RdpName)
			ch <- semp.NewMetric(MetricDesc["RdpStats"]["http_post_response_bytes_received"], prometheus.CounterValue, rdp.Stats.RespByesReceived, rdp.MsgVpnName, rdp.RdpName)
			ch <- semp.NewMetric(MetricDesc["RdpStats"]["http_post_response_messages_received"], prometheus.CounterValue, rdp.Stats.RespMsgReceived, rdp.MsgVpnName, rdp.RdpName)
			ch <- semp.NewMetric(MetricDesc["RdpStats"]["http_post_response_messages_received_error"], prometheus.CounterValue, rdp.Stats.RespMsgReceivedError, rdp.MsgVpnName, rdp.RdpName)
			ch <- semp.NewMetric(MetricDesc["RdpStats"]["http_post_response_messages_received_successful"], prometheus.CounterValue, rdp.Stats.RespMsgReceivedSuccessful, rdp.MsgVpnName, rdp.RdpName)
		}
	})
}
