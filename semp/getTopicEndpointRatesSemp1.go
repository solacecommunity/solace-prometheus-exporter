package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Get rates for each individual topic-endpoint of all vpn's
// This can result in heavy system load for lots of topic-endpoints
// Deprecated: in facor of: getTopicEndpointStatsSemp1
func (e *Semp) GetTopicEndpointRatesSemp1(ch chan<- prometheus.Metric, vpnFilter string, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				TopicEndpoint struct {
					TopicEndpoints struct {
						TopicEndpoint []struct {
							TopicEndointName string `xml:"name"`
							Info             struct {
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
		} `xml:"execute-result"`
	}

	var lastTopicEndpointName = ""
	for nextRequest := "<rpc><show><topic-endpoint><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><rates/><count/><num-elements>100</num-elements></topic-endpoint></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape TopicEndpointRatesSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode TopicEndpointRatesSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
			return 0, errors.New("unexpected result: see log")
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, topicEndpoint := range target.RPC.Show.TopicEndpoint.TopicEndpoints.TopicEndpoint {
			topicEndpointKey := topicEndpoint.Info.MsgVpnName + "___" + topicEndpoint.TopicEndointName
			if topicEndpointKey == lastTopicEndpointName {
				continue
			}
			lastTopicEndpointName = topicEndpointKey
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointRates"]["rx_msg_rate"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.RxMsgRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointRates"]["tx_msg_rate"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.TxMsgRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointRates"]["rx_byte_rate"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.RxByteRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointRates"]["tx_byte_rate"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.TxByteRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointRates"]["rx_msg_rate_avg"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.AverageRxMsgRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointRates"]["tx_msg_rate_avg"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.AverageTxMsgRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointRates"]["rx_byte_rate_avg"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.AverageRxByteRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["TopicEndpointRates"]["tx_byte_rate_avg"], prometheus.GaugeValue, topicEndpoint.Rates.Qendpt.AverageTxByteRate, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndointName)
		}
		body.Close()
	}

	return 1, nil
}
