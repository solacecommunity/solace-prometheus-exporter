package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Get rates for each individual queue of all vpn's
// This can result in heavy system load for lots of queues
func (e *Semp) GetQueueStatsSemp1(ch chan<- prometheus.Metric, vpnFilter string, itemFilter string) (ok float64, err error) {
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
							Stats struct {
								MessageSpoolStats struct {
									TotalByteSpooled       float64 `xml:"total-bytes-spooled"`
									TotalMsgSpooled        float64 `xml:"total-messages-spooled"`
									MsgRedelivered         float64 `xml:"messages-redelivered"`
									MsgRetransmit          float64 `xml:"messages-transport-retransmit"`
									SpoolUsageExceeded     float64 `xml:"spool-usage-exceeded"`
									MsgSizeExceeded        float64 `xml:"max-message-size-exceeded"`
									SpoolShutdownDiscard   float64 `xml:"spool-shutdown-discard"`
									DestinationGroupError  float64 `xml:"destination-group-error"`
									LowPrioMsgDiscard      float64 `xml:"low-priority-msg-congestion-discard"`
									Deleted                float64 `xml:"total-deleted-messages"`
									TtlDisacarded          float64 `xml:"total-ttl-expired-discard-messages"`
									TtlDmq                 float64 `xml:"total-ttl-expired-to-dmq-messages"`
									TtlDmqFailed           float64 `xml:"total-ttl-expired-to-dmq-failures"`
									MaxRedeliveryDiscarded float64 `xml:"max-redelivery-exceeded-discard-messages"`
									MaxRedeliveryDmq       float64 `xml:"max-redelivery-exceeded-to-dmq-messages"`
									MaxRedeliveryDmqFailed float64 `xml:"max-redelivery-exceeded-to-dmq-failures"`
								} `xml:"message-spool-stats"`
							} `xml:"stats"`
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
	for nextRequest := "<rpc><show><queue><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><stats/><count/><num-elements>100</num-elements></queue></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape QueueStatsSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode QueueStatsSemp1", "err", err, "broker", e.brokerURI)
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
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueStats"]["total_bytes_spooled"], prometheus.GaugeValue, queue.Stats.MessageSpoolStats.TotalByteSpooled, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueStats"]["total_messages_spooled"], prometheus.GaugeValue, queue.Stats.MessageSpoolStats.TotalMsgSpooled, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueStats"]["messages_redelivered"], prometheus.GaugeValue, queue.Stats.MessageSpoolStats.MsgRedelivered, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueStats"]["messages_transport_retransmited"], prometheus.GaugeValue, queue.Stats.MessageSpoolStats.MsgRetransmit, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueStats"]["spool_usage_exceeded"], prometheus.GaugeValue, queue.Stats.MessageSpoolStats.SpoolUsageExceeded, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueStats"]["max_message_size_exceeded"], prometheus.GaugeValue, queue.Stats.MessageSpoolStats.MsgSizeExceeded, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueStats"]["total_deleted_messages"], prometheus.GaugeValue, queue.Stats.MessageSpoolStats.Deleted, queue.Info.MsgVpnName, queue.QueueName)
		}
		body.Close()
	}

	return 1, nil
}
