package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Get rates for each individual queue of all vpn's
// This can result in heavy system load for lots of queues
// Deprecated: in facor of: getQueueStatsSemp1
func (e *Semp) GetQueueRatesSemp1(ch chan<- prometheus.Metric, vpnFilter string, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Queue struct {
					Queues struct {
						Queue []struct {
							QueueName string `xml:"name"`
							Info      struct {
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
						} `xml:"queue"`
					} `xml:"queues"`
				} `xml:"queue"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	var lastQueueName = ""
	for nextRequest := "<rpc><show><queue><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><rates/><count/><num-elements>100</num-elements></queue></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape QueueRatesSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode QueueRatesSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
			return 0, errors.New("unexpected result: see log")
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, queue := range target.RPC.Show.Queue.Queues.Queue {
			queueKey := queue.Info.MsgVpnName + "___" + queue.QueueName
			if queueKey == lastQueueName {
				continue
			}
			lastQueueName = queueKey
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueRates"]["queue_rx_msg_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.RxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueRates"]["queue_tx_msg_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.TxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueRates"]["queue_rx_byte_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.RxByteRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueRates"]["queue_tx_byte_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.TxByteRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueRates"]["queue_rx_msg_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageRxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueRates"]["queue_tx_msg_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageTxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueRates"]["queue_rx_byte_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageRxByteRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueRates"]["queue_tx_byte_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageTxByteRate, queue.Info.MsgVpnName, queue.QueueName)
		}
		body.Close()
	}

	return 1, nil
}
