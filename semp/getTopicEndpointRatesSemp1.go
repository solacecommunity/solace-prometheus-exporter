package semp

import (
	"encoding/xml"
	"errors"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// GetTopicEndpointRatesSemp1 Get rates for each individual topic-endpoint of all VPNs
// This can result in heavy system load for lots of topic-endpoints
// Deprecated: in favor of: getTopicEndpointStatsSemp1
func (semp *Semp) GetTopicEndpointRatesSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string) (ok float64, err error) {
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
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
			Reason string `xml:"reason,attr"`
		} `xml:"execute-result"`
	}

	var page = 1
	var lastTopicEndpointName = ""
	for nextRequest := "<rpc><show><topic-endpoint><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><rates/><count/><num-elements>100</num-elements></topic-endpoint></show></rpc>"; nextRequest != ""; {
		body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", nextRequest, "TopicEndpointRatesSemp1", page)
		page++

		if err != nil {
			_ = level.Error(semp.logger).Log("msg", "Can't scrape TopicEndpointRatesSemp1", "err", err, "broker", semp.brokerURI)
			return -1, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(semp.logger).Log("msg", "Can't decode TopicEndpointRatesSemp1", "err", err, "broker", semp.brokerURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(semp.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", semp.brokerURI)
			return 0, errors.New("unexpected result: " + target.ExecuteResult.Reason + ". see log for further details")
		}

		nextRequest = target.MoreCookie.RPC

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
		body.Close()
	}

	return 1, nil
}
