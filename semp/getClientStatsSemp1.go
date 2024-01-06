package semp

import (
	"encoding/xml"
	"errors"

	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Get some statistics for each individual client of all vpn's
// This can result in heavy system load for lots of clients
func (e *Semp) GetClientStatsSemp1(ch chan<- PrometheusMetric, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Client struct {
					PrimaryVirtualRouter struct {
						Client []struct {
							ClientName     string `xml:"name"`
							ClientUsername string `xml:"client-username"`
							MsgVpnName     string `xml:"message-vpn"`
							SlowSubscriber bool   `xml:"slow-subscriber"`
							Stats          struct {
								DataRxByteCount   float64 `xml:"client-data-bytes-received"`
								DataRxMsgCount    float64 `xml:"client-data-messages-received"`
								DataTxByteCount   float64 `xml:"client-data-bytes-sent"`
								DataTxMsgCount    float64 `xml:"client-data-messages-sent"`
								AverageRxByteRate float64 `xml:"average-ingress-byte-rate-per-minute"`
								AverageRxMsgRate  float64 `xml:"average-ingress-rate-per-minute"`
								AverageTxByteRate float64 `xml:"average-egress-byte-rate-per-minute"`
								AverageTxMsgRate  float64 `xml:"average-egress-rate-per-minute"`
								RxByteRate        float64 `xml:"current-ingress-byte-rate-per-second"`
								RxMsgRate         float64 `xml:"current-ingress-rate-per-second"`
								TxByteRate        float64 `xml:"current-egress-byte-rate-per-second"`
								TxMsgRate         float64 `xml:"current-egress-rate-per-second"`
								IngressDiscards   struct {
									DiscardedRxMsgCount float64 `xml:"total-ingress-discards"`
								} `xml:"ingress-discards"`
								EgressDiscards struct {
									DiscardedTxMsgCount float64 `xml:"total-egress-discards"`
								} `xml:"egress-discards"`
							} `xml:"stats"`
						} `xml:"client"`
					} `xml:",any"`
				} `xml:"client"`
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
	for nextRequest := "<rpc><show><client><name>" + itemFilter + "</name><stats/></client></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", nextRequest, "ClientStatsSemp1", page)
		page++

		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't scrape ClientStatSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			_ = level.Error(e.logger).Log("msg", "Can't decode ClientStatSemp1", "err", err, "broker", e.brokerURI)
			return 0, err
		}
		if target.ExecuteResult.Result != "ok" {
			_ = level.Error(e.logger).Log("msg", "unexpected result", "command", nextRequest, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
			return 0, errors.New("unexpected result: see log")
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, client := range target.RPC.Show.Client.PrimaryVirtualRouter.Client {
			ch <- e.NewMetric(MetricDesc["ClientStats"]["client_rx_msgs_total"], prometheus.CounterValue, client.Stats.DataRxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- e.NewMetric(MetricDesc["ClientStats"]["client_tx_msgs_total"], prometheus.CounterValue, client.Stats.DataTxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- e.NewMetric(MetricDesc["ClientStats"]["client_rx_bytes_total"], prometheus.CounterValue, client.Stats.DataRxByteCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- e.NewMetric(MetricDesc["ClientStats"]["client_tx_bytes_total"], prometheus.CounterValue, client.Stats.DataTxByteCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- e.NewMetric(MetricDesc["ClientStats"]["client_rx_discarded_msgs_total"], prometheus.CounterValue, client.Stats.IngressDiscards.DiscardedRxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- e.NewMetric(MetricDesc["ClientStats"]["client_tx_discarded_msgs_total"], prometheus.CounterValue, client.Stats.EgressDiscards.DiscardedTxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- e.NewMetric(MetricDesc["ClientStats"]["client_slow_subscriber"], prometheus.GaugeValue, encodeMetricBool(client.SlowSubscriber), client.MsgVpnName, client.ClientName, client.ClientUsername)
		}
		body.Close()
	}

	return 1, nil
}

// Get some statistics for each individual client connections of all vpn's
// This can result in heavy system load for lots of clients
func (e *Semp) GetClientConnectionStatsSemp1(ch chan<- PrometheusMetric, itemFilter string) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Client struct {
					PrimaryVirtualRouter struct {
						Client []struct {
							ClientName string `xml:"name"`
							MsgVpnName string `xml:"message-vpn"`
							Stats      struct {
								Protocol                   string  `xml:"protocol"`
								IsZip                      bool    `xml:"is-zip"`
								IsSsl                      bool    `xml:"is-ssl"`
								ReceiveQueueBytes          float64 `xml:"receive-queue-bytes"`
								ReceiveQueueSegments       float64 `xml:"receive-queue-segments"`
								SendQueueBytes             float64 `xml:"send-queue-bytes"`
								SendQueueSegments          float64 `xml:"send-queue-segments"`
								LocalAddress               string  `xml:"local-address"`
								ForeignAddress             string  `xml:"foreign-address"`
								State                      string  `xml:"state"`
								MaximumSegmentSize         float64 `xml:"maximum-segment-size"`
								BytesSent                  float64 `xml:"bytes-sent-32bits"`
								BytesReceived              float64 `xml:"bytes-received-32bits"`
								RetransmitTimeMs           float64 `xml:"retransmit-time-ms"`
								RoundTripTimeSmoothUs      float64 `xml:"round-trip-time-smooth-us"`
								RoundTripTimeSmoothNs      float64 `xml:"round-trip-time-smooth-ns"`
								RoundTripTimeMinimumUs     float64 `xml:"round-trip-time-minimum-us"`
								RoundTripTimeVarianceUs    float64 `xml:"round-trip-time-variance-us"`
								AdvertisedWindowSize       float64 `xml:"advertised-window-size"`
								TransmitWindowSize         float64 `xml:"transmit-window-size"`
								BandwidthWindowSize        float64 `xml:"bandwidth-window-size"`
								CongestionWindowSize       float64 `xml:"congestion-window-size"`
								SlowStartThresholdSize     float64 `xml:"slow-start-threshold-size"`
								SegmentsReceivedOutOfOrder float64 `xml:"segments-received-out-of-order"`
								FastRetransmits            float64 `xml:"fast-retransmits"`
								TimedRetransmits           float64 `xml:"timed-retransmits"`
								ConnectionUptime           float64 `xml:"connection-uptime-s"`
								BlockedCyclesPercent       float64 `xml:"blocked-cycles-percent"`
								Interface                  string  `xml:"interface"`
							} `xml:"connection"`
						} `xml:"client"`
					} `xml:",any"`
				} `xml:"client"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><client><name>" + itemFilter + "</name><connections/></client></show></rpc>"

	body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", command, "ClientConnectionStatsSemp1", 1)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape GetClientConnectionStatsSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode GetClientConnectionStatsSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
		return 0, errors.New("unexpected result: see log")
	}

	//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)

	for _, client := range target.RPC.Show.Client.PrimaryVirtualRouter.Client {
		if len(client.MsgVpnName) < 1 {
			// Filter empty items
			continue
		}

		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_is_zip"], prometheus.GaugeValue, encodeMetricBool(client.Stats.IsZip), client.MsgVpnName, client.ClientName)
		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_is_ssl"], prometheus.GaugeValue, encodeMetricBool(client.Stats.IsSsl), client.MsgVpnName, client.ClientName)

		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_receive_queue_bytes"], prometheus.GaugeValue, client.Stats.ReceiveQueueBytes, client.MsgVpnName, client.ClientName)
		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_send_queue_bytes"], prometheus.GaugeValue, client.Stats.ReceiveQueueSegments, client.MsgVpnName, client.ClientName)
		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_receive_queue_segments"], prometheus.GaugeValue, client.Stats.SendQueueBytes, client.MsgVpnName, client.ClientName)
		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_send_queue_segments"], prometheus.GaugeValue, client.Stats.SendQueueSegments, client.MsgVpnName, client.ClientName)

		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_maximum_segment_size"], prometheus.GaugeValue, client.Stats.MaximumSegmentSize, client.MsgVpnName, client.ClientName)
		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_sent_bytes"], prometheus.CounterValue, client.Stats.BytesSent, client.MsgVpnName, client.ClientName)
		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_received_bytes"], prometheus.CounterValue, client.Stats.BytesReceived, client.MsgVpnName, client.ClientName)

		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_retransmit_milliseconds"], prometheus.CounterValue, client.Stats.RetransmitTimeMs, client.MsgVpnName, client.ClientName)
		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_roundtrip_smth_microseconds"], prometheus.CounterValue, client.Stats.RoundTripTimeSmoothUs, client.MsgVpnName, client.ClientName)
		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_roundtrip_min_microseconds"], prometheus.CounterValue, client.Stats.RoundTripTimeMinimumUs, client.MsgVpnName, client.ClientName)
		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_roundtrip_var_microseconds"], prometheus.CounterValue, client.Stats.RoundTripTimeVarianceUs, client.MsgVpnName, client.ClientName)

		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_advertised_window"], prometheus.CounterValue, client.Stats.AdvertisedWindowSize, client.MsgVpnName, client.ClientName)
		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_transmit_window"], prometheus.CounterValue, client.Stats.TransmitWindowSize, client.MsgVpnName, client.ClientName)
		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_congestion_window"], prometheus.CounterValue, client.Stats.CongestionWindowSize, client.MsgVpnName, client.ClientName)

		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_slow_start_threshold"], prometheus.CounterValue, client.Stats.SlowStartThresholdSize, client.MsgVpnName, client.ClientName)
		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_received_outoforder"], prometheus.CounterValue, client.Stats.SegmentsReceivedOutOfOrder, client.MsgVpnName, client.ClientName)
		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_fast_retransmit"], prometheus.CounterValue, client.Stats.FastRetransmits, client.MsgVpnName, client.ClientName)
		ch <- e.NewMetric(MetricDesc["ClientConnections"]["connection_timed_retransmit"], prometheus.CounterValue, client.Stats.TimedRetransmits, client.MsgVpnName, client.ClientName)
	}
	body.Close()

	return 1, nil
}
