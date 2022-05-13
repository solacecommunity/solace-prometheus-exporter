package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"math"
)

// Get some statistics for each individual queue of all vpn's
// This can result in heavy system load for lots of queues
func (e *Semp) GetQueueDetailsSemp1(ch chan<- prometheus.Metric, vpnFilter string, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Queue struct {
					Queues struct {
						Queue []struct {
							QueueName string `xml:"name"`
							Info      struct {
								MsgVpnName      string  `xml:"message-vpn"`
								Quota           float64 `xml:"quota"`
								Usage           float64 `xml:"current-spool-usage-in-mb"`
								SpooledMsgCount float64 `xml:"num-messages-spooled"`
								BindCount       float64 `xml:"bind-count"`
							} `xml:"info"`
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
	for nextRequest := "<rpc><show><queue><name>" + itemFilter + "</name><vpn-name>" + vpnFilter + "</vpn-name><detail/><count/><num-elements>100</num-elements></queue></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape QueueDetailsSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode QueueDetailsSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "Can't scrape QueueDetailsSemp1", "err", err, "broker", e.brokerURI)
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
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueDetails"]["queue_spool_quota_bytes"], prometheus.GaugeValue, math.Round(queue.Info.Quota*1048576.0), queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueDetails"]["queue_spool_usage_bytes"], prometheus.GaugeValue, math.Round(queue.Info.Usage*1048576.0), queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueDetails"]["queue_spool_usage_msgs"], prometheus.GaugeValue, queue.Info.SpooledMsgCount, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(MetricDesc["QueueDetails"]["queue_binds"], prometheus.GaugeValue, queue.Info.BindCount, queue.Info.MsgVpnName, queue.QueueName)
		}
		body.Close()
	}

	return 1, nil
}
