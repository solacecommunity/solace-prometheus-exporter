package semp

import (
	"fmt"
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetRestConsumerStatsSemp1 Get rates for each individual queue of all VPNs
// This can result in heavy system load for lots of queues
func (semp *Semp) GetRestConsumerStatsSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string, sempPageSize int64) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				MessageVpn struct {
					RestConsumerInfo struct {
						StatsInfo []struct {
							RestConsumerName           string  `xml:"name"`
							RdpName                    string  `xml:"rdp-name"`
							MsgVpnName                 string  `xml:"vpn-name"`
							ReqMsgSent                 float64 `xml:"http-post-request-messages-sent"`
							ReqMsgSentOutstanding      float64 `xml:"http-post-request-messages-sent-outstanding"`
							ReqMsgSentConnectionClosed float64 `xml:"http-post-request-messages-sent-connection-closed"`
							ReqMsgSentTimedOut         float64 `xml:"http-post-request-messages-sent-timed-out"`
							RespMsgReceived            float64 `xml:"http-post-response-messages-received"`
							RespMsgReceivedSuccessful  float64 `xml:"http-post-response-messages-received-successful"`
							RespMsgReceivedError       float64 `xml:"http-post-response-messages-received-error"`
							ReqBytesSent               float64 `xml:"http-post-request-bytes-sent"`
							RespByesReceived           float64 `xml:"http-post-response-bytes-received"`
						} `xml:"stats-info"`
					} `xml:"rest-consumer-info"`
				} `xml:"message-vpn"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	var lastConsumerName = ""
	command := fmt.Sprintf("<rpc><show><message-vpn><vpn-name>"+vpnFilter+"</vpn-name><rest></rest><rest-consumer></rest-consumer><rest-consumer-name>"+itemFilter+"</rest-consumer-name><stats></stats><count/><num-elements>%d</num-elements></message-vpn></show></rpc>", sempPageSize)
	return fetchSemp1Pages(semp, command, "RestConsumerStatsSemp1", func(target *Data) {
		for _, consumerStats := range target.RPC.Show.MessageVpn.RestConsumerInfo.StatsInfo {
			consumerKey := consumerStats.MsgVpnName + "___" + consumerStats.RdpName + "___" + consumerStats.RestConsumerName
			if consumerKey == lastConsumerName {
				continue
			}
			lastConsumerName = consumerKey
			ch <- semp.NewMetric(MetricDesc["RestConsumerStats"]["http_post_request_bytes_sent"], prometheus.CounterValue, consumerStats.ReqBytesSent, consumerStats.MsgVpnName, consumerStats.RdpName, consumerStats.RestConsumerName)
			ch <- semp.NewMetric(MetricDesc["RestConsumerStats"]["http_post_request_messages_sent"], prometheus.CounterValue, consumerStats.ReqMsgSent, consumerStats.MsgVpnName, consumerStats.RdpName, consumerStats.RestConsumerName)
			ch <- semp.NewMetric(MetricDesc["RestConsumerStats"]["http_post_request_messages_sent_connection_closed"], prometheus.CounterValue, consumerStats.ReqMsgSentConnectionClosed, consumerStats.MsgVpnName, consumerStats.RdpName, consumerStats.RestConsumerName)
			ch <- semp.NewMetric(MetricDesc["RestConsumerStats"]["http_post_request_messages_sent_outstanding"], prometheus.GaugeValue, consumerStats.ReqMsgSentOutstanding, consumerStats.MsgVpnName, consumerStats.RdpName, consumerStats.RestConsumerName)
			ch <- semp.NewMetric(MetricDesc["RestConsumerStats"]["http_post_request_messages_sent_timed_out"], prometheus.CounterValue, consumerStats.ReqMsgSentTimedOut, consumerStats.MsgVpnName, consumerStats.RdpName, consumerStats.RestConsumerName)
			ch <- semp.NewMetric(MetricDesc["RestConsumerStats"]["http_post_response_bytes_received"], prometheus.CounterValue, consumerStats.RespByesReceived, consumerStats.MsgVpnName, consumerStats.RdpName, consumerStats.RestConsumerName)
			ch <- semp.NewMetric(MetricDesc["RestConsumerStats"]["http_post_response_messages_received"], prometheus.CounterValue, consumerStats.RespMsgReceived, consumerStats.MsgVpnName, consumerStats.RdpName, consumerStats.RestConsumerName)
			ch <- semp.NewMetric(MetricDesc["RestConsumerStats"]["http_post_response_messages_received_error"], prometheus.CounterValue, consumerStats.RespMsgReceivedError, consumerStats.MsgVpnName, consumerStats.RdpName, consumerStats.RestConsumerName)
			ch <- semp.NewMetric(MetricDesc["RestConsumerStats"]["http_post_response_messages_received_successful"], prometheus.CounterValue, consumerStats.RespMsgReceivedSuccessful, consumerStats.MsgVpnName, consumerStats.RdpName, consumerStats.RestConsumerName)
		}
	})
}
