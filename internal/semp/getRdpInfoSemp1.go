package semp

import (
	"encoding/xml"
	"errors"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// GetRdpInfoSemp1 Get rates for each individual queue of all VPNs
// This can result in heavy system load for lots of queues
func (semp *Semp) GetRdpInfoSemp1(ch chan<- PrometheusMetric, vpnFilter string, itemFilter string) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				MessageVpn struct {
					Rest struct {
						RestDeliveryPoints struct {
							Totals struct {
								TotalRdpsUp                  float64 `xml:"total-rest-delivery-points-up"`
								TotalRdpsConfigured          float64 `xml:"total-rest-delivery-points-configured"`
								TotalRCsUp                   float64 `xml:"total-rest-consumers-up"`
								TotalRCsConfigured           float64 `xml:"total-rest-consumers-configured"`
								TotalRCOutConnUp             float64 `xml:"total-rest-consumers-outgoing-connections-up"`
								TotalRCOutConConfigured      float64 `xml:"total-rest-consumers-outgoing-connections-configured"`
								TotalQueueBindingsUp         float64 `xml:"total-queue-bindings-up"`
								TotalQueueBindingsConfigured float64 `xml:"total-queue-bindings-configured"`
							} `xml:"totals"`
							RestDeliveryPoint []struct {
								RdpName           string  `xml:"name"`
								MsgVpnName        string  `xml:"message-vpn"`
								Enabled           bool    `xml:"enabled"`
								OperatingStatus   bool    `xml:"operating-status"`
								ConsumerOutConnUp float64 `xml:"consumer-out-connections-up"`
								ConsumerOutConn   float64 `xml:"consumer-out-connections"`
								QueueBindingsUp   float64 `xml:"queue-bindings-up"`
								QueueBindings     float64 `xml:"queue-bindings"`
								BlockedConnsPerc  float64 `xml:"blocked-conns-percent"`
							} `xml:"rest-delivery-point"`
						} `xml:"rest-delivery-points"`
					} `xml:"rest"`
				} `xml:"message-vpn"`
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
	var lastRdpName = ""
	for nextRequest := "<rpc><show><message-vpn><vpn-name>" + vpnFilter + "</vpn-name><rest></rest><rest-delivery-point></rest-delivery-point><rdp-name>" + itemFilter + "</rdp-name></message-vpn></show></rpc>"; nextRequest != ""; {
		body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", nextRequest, "RdpInfoSemp1", page)
		page++
		if err != nil {
			_ = level.Error(semp.logger).Log("msg", "Can't scrape RdpInfoSemp1", "err", err, "broker", semp.brokerURI)
			return -1, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(semp.logger).Log("msg", "Can't decode Xml RdpInfoSemp1", "err", err, "broker", semp.brokerURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(semp.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", semp.brokerURI)
			return 0, errors.New("unexpected result: " + target.ExecuteResult.Reason + ". see log for further details")
		}
		_ = level.Debug(semp.logger).Log("msg", "Result of RdpInfoSemp1", "results", len(target.RPC.Show.MessageVpn.Rest.RestDeliveryPoints.RestDeliveryPoint), "page", page-1)
		nextRequest = target.MoreCookie.RPC

		rdpTotals := target.RPC.Show.MessageVpn.Rest.RestDeliveryPoints.Totals
		ch <- semp.NewMetric(MetricDesc["RdpTotals"]["total_rest_delivery_points_up"], prometheus.CounterValue, rdpTotals.TotalRdpsUp)
		ch <- semp.NewMetric(MetricDesc["RdpTotals"]["total_rest_delivery_points_configured"], prometheus.CounterValue, rdpTotals.TotalRdpsConfigured)
		ch <- semp.NewMetric(MetricDesc["RdpTotals"]["total_rest_consumers_up"], prometheus.CounterValue, rdpTotals.TotalRCsUp)
		ch <- semp.NewMetric(MetricDesc["RdpTotals"]["total_rest_consumers_configured"], prometheus.CounterValue, rdpTotals.TotalRCsConfigured)
		ch <- semp.NewMetric(MetricDesc["RdpTotals"]["total_rest_consumer_outgoing_connections_up"], prometheus.CounterValue, rdpTotals.TotalRCOutConnUp)
		ch <- semp.NewMetric(MetricDesc["RdpTotals"]["total_rest_consumer_outgoing_connections_configured"], prometheus.CounterValue, rdpTotals.TotalRCOutConConfigured)
		ch <- semp.NewMetric(MetricDesc["RdpTotals"]["total_queue_bindings_up"], prometheus.CounterValue, rdpTotals.TotalQueueBindingsUp)
		ch <- semp.NewMetric(MetricDesc["RdpTotals"]["total_queue_bindings_configured"], prometheus.CounterValue, rdpTotals.TotalQueueBindingsConfigured)
		for _, rdp := range target.RPC.Show.MessageVpn.Rest.RestDeliveryPoints.RestDeliveryPoint {
			rdpKey := rdp.MsgVpnName + "___" + rdp.RdpName
			if rdpKey == lastRdpName {
				continue
			}
			ch <- semp.NewMetric(MetricDesc["RdpInfo"]["enabled"], prometheus.GaugeValue, encodeMetricBool(rdp.Enabled), rdp.MsgVpnName, rdp.RdpName)
			ch <- semp.NewMetric(MetricDesc["RdpInfo"]["operating_status"], prometheus.GaugeValue, encodeMetricBool(rdp.OperatingStatus), rdp.MsgVpnName, rdp.RdpName)
			ch <- semp.NewMetric(MetricDesc["RdpInfo"]["consumer_out_connections_up"], prometheus.CounterValue, rdp.ConsumerOutConnUp, rdp.MsgVpnName, rdp.RdpName)
			ch <- semp.NewMetric(MetricDesc["RdpInfo"]["consumer_out_connections_configured"], prometheus.CounterValue, rdp.ConsumerOutConn, rdp.MsgVpnName, rdp.RdpName)
			ch <- semp.NewMetric(MetricDesc["RdpInfo"]["queue_bindings_up"], prometheus.CounterValue, rdp.QueueBindingsUp, rdp.MsgVpnName, rdp.RdpName)
			ch <- semp.NewMetric(MetricDesc["RdpInfo"]["queue_bindings_configured"], prometheus.CounterValue, rdp.QueueBindings, rdp.MsgVpnName, rdp.RdpName)
			ch <- semp.NewMetric(MetricDesc["RdpInfo"]["blocked_conns_percent"], prometheus.CounterValue, rdp.BlockedConnsPerc, rdp.MsgVpnName, rdp.RdpName)
		}
		_ = body.Close()
	}
	return 1, nil
}
