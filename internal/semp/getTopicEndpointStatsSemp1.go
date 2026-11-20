package semp

import (
	"fmt"
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetTopicEndpointStatsSemp1 Get rates for each individual topic-endpoint of all VPNs
// This can result in heavy system load for lots of topc-endpoints
func (semp *Semp) GetTopicEndpointStatsSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string, sempPageSize int64) (float64, error) {
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
						} `xml:"topic-endpoint"`
					} `xml:"topic-endpoints"`
				} `xml:"topic-endpoint"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	var lastTopicEndpointName = ""
	command := fmt.Sprintf("<rpc><show><topic-endpoint><name>"+itemFilter+"</name><vpn-name>"+vpnFilter+"</vpn-name><stats/><count/><num-elements>%d</num-elements></topic-endpoint></show></rpc>", sempPageSize)
	return fetchSemp1Pages(semp, command, "TopicEndpointStatsSemp1", func(target *Data) {
		for _, topicEndpoint := range target.RPC.Show.TopicEndpoint.TopicEndpoints.TopicEndpoint {
			topicEndpointKey := topicEndpoint.Info.MsgVpnName + "___" + topicEndpoint.TopicEndpointName
			if topicEndpointKey == lastTopicEndpointName {
				continue
			}
			lastTopicEndpointName = topicEndpointKey
			ch <- semp.NewMetric(MetricDesc["TopicEndpointStats"]["total_bytes_spooled"], prometheus.CounterValue, topicEndpoint.Stats.MessageSpoolStats.TotalByteSpooled, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointStats"]["total_messages_spooled"], prometheus.CounterValue, topicEndpoint.Stats.MessageSpoolStats.TotalMsgSpooled, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointStats"]["messages_redelivered"], prometheus.CounterValue, topicEndpoint.Stats.MessageSpoolStats.MsgRedelivered, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointStats"]["messages_transport_retransmitted"], prometheus.CounterValue, topicEndpoint.Stats.MessageSpoolStats.MsgRetransmit, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointStats"]["spool_usage_exceeded"], prometheus.CounterValue, topicEndpoint.Stats.MessageSpoolStats.SpoolUsageExceeded, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointStats"]["max_message_size_exceeded"], prometheus.CounterValue, topicEndpoint.Stats.MessageSpoolStats.MsgSizeExceeded, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointStats"]["total_deleted_messages"], prometheus.CounterValue, topicEndpoint.Stats.MessageSpoolStats.Deleted, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointStats"]["messages_shutdown_discarded"], prometheus.CounterValue, topicEndpoint.Stats.MessageSpoolStats.SpoolShutdownDiscard, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointStats"]["messages_ttl_discarded"], prometheus.CounterValue, topicEndpoint.Stats.MessageSpoolStats.TTLDiscarded, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointStats"]["messages_ttl_dmq"], prometheus.CounterValue, topicEndpoint.Stats.MessageSpoolStats.TTLDmq, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointStats"]["messages_ttl_dmq_failed"], prometheus.CounterValue, topicEndpoint.Stats.MessageSpoolStats.TTLDmqFailed, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointStats"]["messages_max_redelivered_discarded"], prometheus.CounterValue, topicEndpoint.Stats.MessageSpoolStats.MaxRedeliveryDiscarded, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointStats"]["messages_max_redelivered_dmq"], prometheus.CounterValue, topicEndpoint.Stats.MessageSpoolStats.MaxRedeliveryDmq, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
			ch <- semp.NewMetric(MetricDesc["TopicEndpointStats"]["messages_max_redelivered_dmq_failed"], prometheus.CounterValue, topicEndpoint.Stats.MessageSpoolStats.MaxRedeliveryDmqFailed, topicEndpoint.Info.MsgVpnName, topicEndpoint.TopicEndpointName)
		}
	})
}
