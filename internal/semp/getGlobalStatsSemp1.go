package semp

import (
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetGlobalSystemInfoSemp1 Get global stats information
func (semp *Semp) GetGlobalSystemInfoSemp1(ch chan<- PrometheusMetric) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				System struct {
					UptimeSeconds      float64 `xml:"system-uptime-seconds"`
					ConnectionsQuota   float64 `xml:"max-connections"`
					MessagesQueueQuota float64 `xml:"max-queue-messages"`
					CPUCores           float64 `xml:"cpu-cores"`
					SystemMemory       float64 `xml:"system-memory"`
				} `xml:"system"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	command := "<rpc><show><system/></show></rpc>"
	var target Data
	if up, err := semp.fetchSemp1(command, "GetGlobalSystemInfoSemp1", 1, &target); err != nil {
		return up, err
	}

	ch <- semp.NewMetric(MetricDesc["GlobalStats"]["system_uptime_seconds"], prometheus.CounterValue, target.RPC.Show.System.UptimeSeconds)
	ch <- semp.NewMetric(MetricDesc["GlobalStats"]["system_total_clients_quota"], prometheus.CounterValue, target.RPC.Show.System.ConnectionsQuota)
	ch <- semp.NewMetric(MetricDesc["GlobalStats"]["system_message_spool_quota"], prometheus.GaugeValue, target.RPC.Show.System.MessagesQueueQuota*1000000)
	ch <- semp.NewMetric(MetricDesc["GlobalStats"]["system_cpu_cores"], prometheus.GaugeValue, target.RPC.Show.System.CPUCores)
	ch <- semp.NewMetric(MetricDesc["GlobalStats"]["system_memory_bytes"], prometheus.GaugeValue, target.RPC.Show.System.SystemMemory*1073741824.0)

	return 1, nil
}

func (semp *Semp) GetGlobalStatsSemp1(ch chan<- PrometheusMetric) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Stats struct {
					Client struct {
						Global struct {
							Stats struct {
								ClientsConnected float64 `xml:"total-clients-connected"`
								DataRxMsgCount   float64 `xml:"client-data-messages-received"`
								DataTxMsgCount   float64 `xml:"client-data-messages-sent"`
								DataRxByteCount  float64 `xml:"client-data-bytes-received"`
								DataTxByteCount  float64 `xml:"client-data-bytes-sent"`
								RxMsgsRate       float64 `xml:"current-ingress-rate-per-second"`
								TxMsgsRate       float64 `xml:"current-egress-rate-per-second"`
								RxBytesRate      float64 `xml:"current-ingress-byte-rate-per-second"`
								TxBytesRate      float64 `xml:"current-egress-byte-rate-per-second"`
								IngressDiscards  struct {
									DiscardedRxMsgCount float64 `xml:"total-ingress-discards"`
								} `xml:"ingress-discards"`
								EgressDiscards struct {
									DiscardedTxMsgCount float64 `xml:"total-egress-discards"`
								} `xml:"egress-discards"`
							} `xml:"stats"`
						} `xml:"global"`
					} `xml:"client"`
				} `xml:"stats"`
			} `xml:"show"`
		} `xml:"rpc"`
		types.Reply
	}

	command := "<rpc><show><stats><client/></stats></show></rpc>"
	var target Data
	if up, err := semp.fetchSemp1(command, "GlobalStatsSemp1", 1, &target); err != nil {
		return up, err
	}

	ch <- semp.NewMetric(MetricDesc["GlobalStats"]["system_total_clients_connected"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.ClientsConnected)
	ch <- semp.NewMetric(MetricDesc["GlobalStats"]["system_rx_msgs_total"], prometheus.CounterValue, target.RPC.Show.Stats.Client.Global.Stats.DataRxMsgCount)
	ch <- semp.NewMetric(MetricDesc["GlobalStats"]["system_tx_msgs_total"], prometheus.CounterValue, target.RPC.Show.Stats.Client.Global.Stats.DataTxMsgCount)
	ch <- semp.NewMetric(MetricDesc["GlobalStats"]["system_rx_bytes_total"], prometheus.CounterValue, target.RPC.Show.Stats.Client.Global.Stats.DataRxByteCount)
	ch <- semp.NewMetric(MetricDesc["GlobalStats"]["system_tx_bytes_total"], prometheus.CounterValue, target.RPC.Show.Stats.Client.Global.Stats.DataTxByteCount)
	ch <- semp.NewMetric(MetricDesc["GlobalStats"]["system_total_rx_discards"], prometheus.CounterValue, target.RPC.Show.Stats.Client.Global.Stats.IngressDiscards.DiscardedRxMsgCount)
	ch <- semp.NewMetric(MetricDesc["GlobalStats"]["system_total_tx_discards"], prometheus.CounterValue, target.RPC.Show.Stats.Client.Global.Stats.EgressDiscards.DiscardedTxMsgCount)

	return 1, nil
}
