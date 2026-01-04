package semp

import (
	"encoding/xml"
	"math"
	"solace_exporter/internal/semp/types"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// GetQueueDetailsSemp1 Get some statistics for each individual queue of all VPNs
// This can result in heavy system load for lots of queues
func (semp *Semp) GetQueueDetailsSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string) (float64, error) {
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
		MoreCookie    types.MoreCookie    `xml:"more-cookie,omitempty"`
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	var lastQueueName = ""
	var page = 1
	for command := "<rpc><show><queue><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><detail/><count/><num-elements>100</num-elements></queue></show></rpc>"; command != ""; {
		body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "QueueDetailsSemp1", page)
		page++

		if err != nil {
			_ = level.Error(semp.logger).Log("msg", "Can't scrape QueueDetailsSemp1", "err", err, "broker", semp.brokerURI)
			return -1, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(semp.logger).Log("msg", "Can't decode QueueDetailsSemp1", "err", err, "broker", semp.brokerURI)
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
			ch <- semp.NewMetric(MetricDesc["QueueDetails"]["queue_spool_quota_bytes"], prometheus.GaugeValue, math.Round(queue.Info.Quota*1048576.0), queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueDetails"]["queue_spool_usage_bytes"], prometheus.CounterValue, math.Round(queue.Info.Usage*1048576.0), queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueDetails"]["queue_spool_usage_msgs"], prometheus.GaugeValue, queue.Info.SpooledMsgCount, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueDetails"]["queue_binds"], prometheus.GaugeValue, queue.Info.BindCount, queue.Info.MsgVpnName, queue.QueueName)
			ch <- semp.NewMetric(MetricDesc["QueueDetails"]["queue_subscriptions"], prometheus.GaugeValue, queue.Info.TopicSubscriptionCount, queue.Info.MsgVpnName, queue.QueueName)
		}
		body.Close()
	}

	return 1, nil
}
