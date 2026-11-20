package semp

import (
	"fmt"
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetTopicEndpointRatesSemp1 Get rates for each individual topic-endpoint of all VPNs
// This can result in heavy system load for lots of topic-endpoints
// Deprecated: in favor of: getTopicEndpointStatsSemp1
func (semp *Semp) GetTopicEndpointRatesSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string, sempPageSize int64) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				TopicEndpoint struct {
					TopicEndpoints struct {
						TopicEndpoint []struct {
							TopicEndpointName string `xml:"name"`
							Info              struct {
								MsgVpnName string `xml:"message-vpn"`
							} `xml:"info"`
							Rates struct {
								Qendpt struct {
									AverageRxByteRate float64 `xml:"average-ingress-byte-rate-per-minute"`
									AverageRxMsgRate  float64 `xml:"average-ingress-rate-per-minute"`
									AverageTxByteRate float64 `xml:"average-egress-byte-rate-per-minute"`
									AverageTxMsgRate  float64 `xml:"average-egress-rate-per-minute"`
									RxByteRate        float64 `xml:"current-ingress-byte-rate-per-second"`
									RxMsgRate         float64 `xml:"current-ingress-rate-per-second"`
									TxByteRate        float64 `xml:"current-egress-byte-rate-per-second"`
									TxMsgRate         float64 `xml:"current-egress-rate-per-second"`
								} `xml:"qendpt-data-rates"`
							} `xml:"rates"`
						} `xml:"topic-endpoint"`
					} `xml:"topic-endpoints"`
				} `xml:"topic-endpoint"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	var lastTopicEndpointName = ""
	command := fmt.Sprintf("<rpc><show><topic-endpoint><name>"+itemFilter+"</name><vpn-name>"+vpnFilter+"</vpn-name><rates/><count/><num-elements>%d</num-elements></topic-endpoint></show></rpc>", sempPageSize)
	return fetchSemp1Pages(semp, command, "TopicEndpointRatesSemp1", func(target *Data) {
		for _, topicEndpoint := range target.RPC.Show.TopicEndpoint.TopicEndpoints.TopicEndpoint {
			topicEndpointKey := topicEndpoint.Info.MsgVpnName + "___" + topicEndpoint.TopicEndpointName
			if topicEndpointKey == lastTopicEndpointName {
				continue
			}
			lastTopicEndpointName = topicEndpointKey
			ch <- semp.NewMetric(MetricDesc["TopicEndpointRates"]["rx_msg_rate"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.RxMsgRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointRates"]["tx_msg_rate"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.TxMsgRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointRates"]["rx_byte_rate"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.RxByteRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointRates"]["tx_byte_rate"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.TxByteRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointRates"]["rx_msg_rate_avg"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.AverageRxMsgRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointRates"]["tx_msg_rate_avg"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.AverageTxMsgRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointRates"]["rx_byte_rate_avg"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.AverageRxByteRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointRates"]["tx_byte_rate_avg"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.AverageTxByteRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
		}
	})
}
