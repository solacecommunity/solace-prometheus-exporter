package semp

import (
	"encoding/xml"
	"errors"
	"strconv"

	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Get rates for each individual queue of all vpn's
// This can result in heavy system load for lots of queues
func (e *Semp) GetRestConsumerStatsSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				MessageVpn struct {
					RestConsumerInfo struct {
						StatsInfo []struct {
							RestConsumerName           string  `xml:"name"`
							RdpName                    string  `xml:"rdp-name"`
							MsgVpnName                 string  `xml:"vpn-name"`
							ReqMsgSent                 float64 `xml:"http-post-request-messages-sent"`
							ReqMsgSentOutstanding      float64 `xml:"http-post-request-messages-sent-outstanding"`
							ReqMsgSentConnectionClosed float64 `xml:"http-post-request-messages-sent-connection-closed"`
							ReqMsgSentTimedOut         float64 `xml:"http-post-request-messages-sent-timed-out"`
							RespMsgReceived            float64 `xml:"http-post-response-messages-received"`
							RespMsgReceivedSuccessful  float64 `xml:"http-post-response-messages-received-successful"`
							RespMsgReceivedError       float64 `xml:"http-post-response-messages-received-error"`
							ReqBytesSent               float64 `xml:"http-post-request-bytes-sent"`
							RespByesReceived           float64 `xml:"http-post-response-bytes-received"`
						} `xml:"stats-info"`
					} `xml:"rest-consumer-info"`
				} `xml:"message-vpn"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	var page = 1
	var lastConsumerName = ""
	numOfElementsPerRequest := int64(100)
	for nextRequest := "<rpc><show><message-vpn><vpn-name>" + vpnFilter + "</vpn-name><rest></rest><rest-consumer></rest-consumer><rest-consumer-name>" + itemFilter + "</rest-consumer-name><stats></stats><count/><num-elements>" + strconv.FormatInt(numOfElementsPerRequest, 10) + "</num-elements></message-vpn></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", nextRequest, "RestConsumerStatsSemp1", page)
		page++

		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape RestConsumerStatsSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode Xml RestConsumerStatsSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
			return 0, errors.New("unexpected result: see log")
		}

		_ = level.Debug(e.logger).Log("msg", "Result of RestConsumerStatsSemp1", "results", len(target.RPC.Show.MessageVpn.RestConsumerInfo.StatsInfo), "page", page-1)

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, consumerStats := range target.RPC.Show.MessageVpn.RestConsumerInfo.StatsInfo {
			consumerKey := consumerStats.MsgVpnName + "___" + consumerStats.RdpName + "___" + consumerStats.RestConsumerName
			if consumerKey == lastConsumerName {
				continue
			}
			lastConsumerName = consumerKey
			ch <- e.NewMetric(MetricDesc["RestConsumerStats"]["http_post_request_bytes_sent"], prometheus.CounterValue, consumerStats.ReqBytesSent, consumerStats.MsgVpnName, consumerStats.RdpName, consumerStats.RestConsumerName)
			ch <- e.NewMetric(MetricDesc["RestConsumerStats"]["http_post_request_messages_sent"], prometheus.CounterValue, consumerStats.ReqMsgSent, consumerStats.MsgVpnName, consumerStats.RdpName, consumerStats.RestConsumerName)
			ch <- e.NewMetric(MetricDesc["RestConsumerStats"]["http_post_request_messages_sent_connection_closed"], prometheus.CounterValue, consumerStats.ReqMsgSentConnectionClosed, consumerStats.MsgVpnName, consumerStats.RdpName, consumerStats.RestConsumerName)
			ch <- e.NewMetric(MetricDesc["RestConsumerStats"]["http_post_request_messages_sent_outstanding"], prometheus.GaugeValue, consumerStats.ReqMsgSentOutstanding, consumerStats.MsgVpnName, consumerStats.RdpName, consumerStats.RestConsumerName)
			ch <- e.NewMetric(MetricDesc["RestConsumerStats"]["http_post_request_messages_sent_timed_out"], prometheus.CounterValue, consumerStats.ReqMsgSentTimedOut, consumerStats.MsgVpnName, consumerStats.RdpName, consumerStats.RestConsumerName)
			ch <- e.NewMetric(MetricDesc["RestConsumerStats"]["http_post_response_bytes_received"], prometheus.CounterValue, consumerStats.RespByesReceived, consumerStats.MsgVpnName, consumerStats.RdpName, consumerStats.RestConsumerName)
			ch <- e.NewMetric(MetricDesc["RestConsumerStats"]["http_post_response_messages_received"], prometheus.CounterValue, consumerStats.RespMsgReceived, consumerStats.MsgVpnName, consumerStats.RdpName, consumerStats.RestConsumerName)
			ch <- e.NewMetric(MetricDesc["RestConsumerStats"]["http_post_response_messages_received_error"], prometheus.CounterValue, consumerStats.RespMsgReceivedError, consumerStats.MsgVpnName, consumerStats.RdpName, consumerStats.RestConsumerName)
			ch <- e.NewMetric(MetricDesc["RestConsumerStats"]["http_post_response_messages_received_successful"], prometheus.CounterValue, consumerStats.RespMsgReceivedSuccessful, consumerStats.MsgVpnName, consumerStats.RdpName, consumerStats.RestConsumerName)
		}
		_ = body.Close()
	}

	return 1, nil
}
