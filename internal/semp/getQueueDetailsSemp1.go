package semp

import (
	"fmt"
	"math"
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetQueueDetailsSemp1 Get some statistics for each individual queue of all VPNs
// This can result in heavy system load for lots of queues
func (semp *Semp) GetQueueDetailsSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string, sempPageSize int64) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Queue struct {
					Queues struct {
						Queue []struct {
							QueueName string `xml:"name"`
							Info      struct {
								MsgVpnName             string  `xml:"message-vpn"`
								Quota                  float64 `xml:"quota"`
								Usage                  float64 `xml:"current-spool-usage-in-mb"`
								SpooledMsgCount        float64 `xml:"num-messages-spooled"`
								BindCount              float64 `xml:"bind-count"`
								TopicSubscriptionCount float64 `xml:"topic-subscription-count"`
							} `xml:"info"`
						} `xml:"queue"`
					} `xml:"queues"`
				} `xml:"queue"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	var lastQueueName = ""
	command := fmt.Sprintf("<rpc><show><queue><name>"+itemFilter+"</name><vpn-name>"+vpnFilter+"</vpn-name><detail/><count/><num-elements>%d</num-elements></queue></show></rpc>", sempPageSize)
	return fetchSemp1Pages(semp, command, "QueueDetailsSemp1", func(target *Data) {
		for _, queue := range target.RPC.Show.Queue.Queues.Queue {
			queueKey := queue.Info.MsgVpnName + "___" + queue.QueueName
			if queueKey == lastQueueName {
				continue
			}
			lastQueueName = queueKey
			ch <- semp.NewMetric(MetricDesc["QueueDetails"]["queue_spool_quota_bytes"], prometheus.GaugeValue, math.Round(queue.Info.Quota*1048576.0), queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueDetails"]["queue_spool_usage_bytes"], prometheus.CounterValue, math.Round(queue.Info.Usage*1048576.0), queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueDetails"]["queue_spool_usage_msgs"], prometheus.GaugeValue, queue.Info.SpooledMsgCount, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueDetails"]["queue_binds"], prometheus.GaugeValue, queue.Info.BindCount, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueDetails"]["queue_subscriptions"], prometheus.GaugeValue, queue.Info.TopicSubscriptionCount, queue.Info.MsgVpnName, queue.QueueName)
		}
	})
}
