package semp

import (
	"encoding/xml"
	"solace_exporter/internal/semp/types"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// GetQueueRatesSemp1 Get rates for each individual queue of all VPNs
// This can result in heavy system load for lots of queues
// Deprecated: in facor of: getQueueStatsSemp1
func (semp *Semp) GetQueueRatesSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string) (float64, error) {
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
		MoreCookie    types.MoreCookie    `xml:"more-cookie,omitempty"`
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	var page = 1
	var lastQueueName = ""
	for command := "<rpc><show><queue><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><rates/><count/><num-elements>100</num-elements></queue></show></rpc>"; command != ""; {
		body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "QueueRatesSemp1", page)
		page++

		if err != nil {
			_ = level.Error(semp.logger).Log("msg", "Can't scrape QueueRatesSemp1", "err", err, "broker", semp.brokerURI)
			return -1, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(semp.logger).Log("msg", "Can't decode QueueRatesSemp1", "err", err, "broker", semp.brokerURI)
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

		command = target.MoreCookie.RPC

		for _, queue := range target.RPC.Show.Queue.Queues.Queue {
			queueKey := queue.Info.MsgVpnName + "___" + queue.QueueName
			if queueKey == lastQueueName {
				continue
			}
			lastQueueName = queueKey
			ch <- semp.NewMetric(MetricDesc["QueueRates"]["queue_rx_msg_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.RxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueRates"]["queue_tx_msg_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.TxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueRates"]["queue_rx_byte_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.RxByteRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueRates"]["queue_tx_byte_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.TxByteRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueRates"]["queue_rx_msg_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageRxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueRates"]["queue_tx_msg_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageTxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueRates"]["queue_rx_byte_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageRxByteRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueRates"]["queue_tx_byte_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageTxByteRate, queue.Info.MsgVpnName, queue.QueueName)
		}
		body.Close()
	}

	return 1, nil
}
