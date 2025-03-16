package semp

import (
	"encoding/xml"
	"errors"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// GetQueueStatsSemp1 Get rates for each individual queue of all VPNs
// This can result in heavy system load for lots of queues
func (semp *Semp) GetQueueStatsSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string) (ok float64, err error) {
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
									TTLDiscarded           float64 `xml:"total-ttl-expired-discard-messages"`
									TTLDmq                 float64 `xml:"total-ttl-expired-to-dmq-messages"`
									TTLDmqFailed           float64 `xml:"total-ttl-expired-to-dmq-failures"`
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
			Reason string `xml:"reason,attr"`
		} `xml:"execute-result"`
	}

	var page = 1
	var lastQueueName = ""
	for nextRequest := "<rpc><show><queue><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><stats/><count/><num-elements>100</num-elements></queue></show></rpc>"; nextRequest != ""; {
		body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", nextRequest, "QueueStatsSemp1", page)
		page++

		if err != nil {
			_ = level.Error(semp.logger).Log("msg", "Can't scrape QueueStatsSemp1", "err", err, "broker", semp.brokerURI)
			return -1, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(semp.logger).Log("msg", "Can't decode QueueStatsSemp1", "err", err, "broker", semp.brokerURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(semp.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", semp.brokerURI)
			return 0, errors.New("unexpected result: " + target.ExecuteResult.Reason + ". see log for further details")
		}

		_ = level.Debug(semp.logger).Log("msg", "Result of QueueStatsSemp1", "results", len(target.RPC.Show.Queue.Queues.Queue), "page", page-1)

		nextRequest = target.MoreCookie.RPC
		for _, queue := range target.RPC.Show.Queue.Queues.Queue {
			queueKey := queue.Info.MsgVpnName + "___" + queue.QueueName
			if queueKey == lastQueueName {
				continue
			}
			lastQueueName = queueKey
			ch <- semp.NewMetric(MetricDesc["QueueStats"]["total_bytes_spooled"], prometheus.CounterValue, queue.Stats.MessageSpoolStats.TotalByteSpooled, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueStats"]["total_messages_spooled"], prometheus.CounterValue, queue.Stats.MessageSpoolStats.TotalMsgSpooled, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueStats"]["messages_redelivered"], prometheus.CounterValue, queue.Stats.MessageSpoolStats.MsgRedelivered, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueStats"]["messages_transport_retransmitted"], prometheus.CounterValue, queue.Stats.MessageSpoolStats.MsgRetransmit, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueStats"]["spool_usage_exceeded"], prometheus.CounterValue, queue.Stats.MessageSpoolStats.SpoolUsageExceeded, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueStats"]["max_message_size_exceeded"], prometheus.CounterValue, queue.Stats.MessageSpoolStats.MsgSizeExceeded, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueStats"]["total_deleted_messages"], prometheus.CounterValue, queue.Stats.MessageSpoolStats.Deleted, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueStats"]["messages_shutdown_discarded"], prometheus.CounterValue, queue.Stats.MessageSpoolStats.SpoolShutdownDiscard, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueStats"]["messages_ttl_discarded"], prometheus.CounterValue, queue.Stats.MessageSpoolStats.TTLDiscarded, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueStats"]["messages_ttl_dmq"], prometheus.CounterValue, queue.Stats.MessageSpoolStats.TTLDmq, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueStats"]["messages_ttl_dmq_failed"], prometheus.CounterValue, queue.Stats.MessageSpoolStats.TTLDmqFailed, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueStats"]["messages_max_redelivered_discarded"], prometheus.CounterValue, queue.Stats.MessageSpoolStats.MaxRedeliveryDiscarded, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueStats"]["messages_max_redelivered_dmq"], prometheus.CounterValue, queue.Stats.MessageSpoolStats.MaxRedeliveryDmq, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueStats"]["messages_max_redelivered_dmq_failed"], prometheus.CounterValue, queue.Stats.MessageSpoolStats.MaxRedeliveryDmqFailed, queue.Info.MsgVpnName, queue.QueueName)
		}
		_ = body.Close()
	}

	return 1, nil
}
