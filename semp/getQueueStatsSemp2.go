package semp

import (
	"encoding/json"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
)

// Get rates for each individual queue of all vpn's
// This can result in heavy system load for lots of queues
func (e *Semp) GetQueueStatsSemp2(ch chan<- PrometheusMetric, vpnName string, itemFilter string, metricFilter []string) (ok float64, err error) {
	type Response struct {
		Queue []struct {
			QueueName                           string  `json:"queueName"`
			MsgVpnName                          string  `json:""`
			TotalByteSpooled                    float64 `json:"spooledByteCount"`
			TotalMsgSpooled                     float64 `json:"spooledMsgCount"`
			MsgRedelivered                      float64 `json:"redeliveredMsgCount"`
			MsgRetransmit                       float64 `json:"transportRetransmitMsgCount"`
			SpoolUsageExceeded                  float64 `json:"maxMsgSpoolUsageExceededDiscardedMsgCount"`
			MsgSizeExceeded                     float64 `json:"maxMsgSizeExceededDiscardedMsgCount"`
			SpoolShutdownDiscard                float64 `json:"disabledDiscardedMsgCount"`
			DestinationGroupError               float64 `json:"destinationGroupErrorDiscardedMsgCount"`
			LowPrioMsgDiscard                   float64 `json:"lowPriorityMsgCongestionDiscardedMsgCount"`
			Deleted                             float64 `json:"deletedMsgCount"`
			TtlDiscarded                        float64 `json:"maxTtlExpiredDiscardedMsgCount"`
			TtlDmq                              float64 `json:"maxTtlExpiredToDmqMsgCount"`
			TtlDmqFailed                        float64 `json:"maxTtlExpiredToDmqFailedMsgCount"`
			MaxRedeliveryDiscarded              float64 `json:"maxRedeliveryExceededDiscardedMsgCount"`
			MaxRedeliveryDmq                    float64 `json:"maxRedeliveryExceededToDmqMsgCount"`
			MaxRedeliveryDmqFailed              float64 `json:"maxRedeliveryExceededToDmqFailedMsgCount"`
			TxUnackedMsg                        float64 `json:"txUnackedMsgCount"`
			TransactionNotSupportedDiscardedMsg float64 `json:"xaTransactionNotSupportedDiscardedMsgCount"`
		} `json:"data"`
		Meta struct {
			Count        int64 `json:"count"`
			ResponseCode int   `json:"responseCode"`
			Paging       struct {
				CursorQuery string `json:"cursorQuery"`
				NextPageUri string `json:"nextPageUri"`
			} `json:",paging"`
			Error struct {
				Code        int    `json:"code"`
				Description string `json:"description"`
				Status      string `json:"status"`
			} `json:",error"`
		} `json:"meta"`
	}

	var getParameter = "count=100"
	if len(strings.TrimSpace(itemFilter)) > 0 && itemFilter != "*" {
		if strings.Contains(itemFilter, "=") {
			getParameter += "&where=" + queryEscape(itemFilter)
		} else {
			getParameter += "&where=" + queryEscape("queueName=="+itemFilter)
		}
	}

	var fieldsToSelect []string
	if len(metricFilter) > 0 {
		fieldsToSelect, err = getSempV2FieldsToSelect(
			metricFilter,
			[]string{"queueName", "msgVpnName"},
			QueueStats,
		)

		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Unable to map metric filter", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		getParameter += "&select=" + strings.Join(fieldsToSelect, ",")
	}

	var page = 1
	var lastQueueName = ""
	for nextUrl := e.brokerURI + "/SEMP/v2/monitor/msgVpns/" + vpnName + "/queues?" + getParameter; nextUrl != ""; {
		body, err := e.getHTTPbytes(nextUrl, "application/json ", "QueueStatsSemp2", page)
		page++

		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape QueueStatsSemp2", "command", nextUrl, "err", err, "broker", e.brokerURI)
			return 0, err
		}

		var response Response
		err = json.Unmarshal(body, &response)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode QueueStatsSemp2", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		if response.Meta.ResponseCode != 200 {
			_ = level.Error(e.logger).Log("msg", "unexpected result", "command", nextUrl, "remoteError", response.Meta.Error.Description, "broker", e.brokerURI)
			return 0, errors.New("unexpected result: see log")
		}

		_ = level.Debug(e.logger).Log("msg", "Result of QueueStatsSemp2", "results", len(response.Queue), "page", page-1)

		//fmt.Printf("Next request: %v\n", response.Meta.Paging.NextPageUri)
		nextUrl = response.Meta.Paging.NextPageUri
		for _, queue := range response.Queue {
			queueKey := queue.MsgVpnName + "___" + queue.QueueName
			if queueKey == lastQueueName {
				continue
			}
			lastQueueName = queueKey

			var values = []SempV2Result{
				{v2Desc: QueueStats["total_bytes_spooled"], value: queue.TotalByteSpooled},
				{v2Desc: QueueStats["messages_redelivered"], value: queue.MsgRedelivered},
				{v2Desc: QueueStats["messages_transport_retransmited"], value: queue.MsgRetransmit},
				{v2Desc: QueueStats["spool_usage_exceeded"], value: queue.SpoolUsageExceeded},
				{v2Desc: QueueStats["max_message_size_exceeded"], value: queue.MsgSizeExceeded},
				{v2Desc: QueueStats["total_deleted_messages"], value: queue.Deleted},
				{v2Desc: QueueStats["messages_shutdown_discarded"], value: queue.SpoolShutdownDiscard},
				{v2Desc: QueueStats["messages_ttl_discarded"], value: queue.TtlDiscarded},
				{v2Desc: QueueStats["messages_ttl_dmq"], value: queue.TtlDmq},
				{v2Desc: QueueStats["messages_ttl_dmq_failed"], value: queue.TtlDmqFailed},
				{v2Desc: QueueStats["messages_max_redelivered_discarded"], value: queue.MaxRedeliveryDiscarded},
				{v2Desc: QueueStats["messages_max_redelivered_dmq"], value: queue.MaxRedeliveryDmq},
				{v2Desc: QueueStats["messages_max_redelivered_dmq_failed"], value: queue.MaxRedeliveryDmqFailed},
			}

			for _, v := range values {
				if v.v2Desc.isSelected(fieldsToSelect) {
					ch <- e.NewMetric(v.v2Desc, prometheus.GaugeValue, v.value, queue.MsgVpnName, queue.QueueName)
				}
			}
		}
	}

	return 1, nil
}
