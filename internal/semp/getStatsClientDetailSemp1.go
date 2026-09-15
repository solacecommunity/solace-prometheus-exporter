package semp

import (
	"encoding/xml"
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetStatsClientDetailSemp1 Get stats client information
func (semp *Semp) GetStatsClientDetailSemp1(ch chan<- PrometheusMetric) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Stats struct {
					Client struct {
						Global struct {
							Stats struct {
								// Msg-Rate current (per-second)
								CurrentIngressRatePerSecond              float64 `xml:"current-ingress-rate-per-second"`
								CurrentEgressRatePerSecond               float64 `xml:"current-egress-rate-per-second"`
								CurrentIngressPersistentRatePerSecond    float64 `xml:"current-ingress-persistent-rate-per-second"`
								CurrentEgressPersistentRatePerSecond     float64 `xml:"current-egress-persistent-rate-per-second"`
								CurrentIngressNonpersistentRatePerSecond float64 `xml:"current-ingress-nonpersistent-rate-per-second"`
								CurrentEgressNonpersistentRatePerSecond  float64 `xml:"current-egress-nonpersistent-rate-per-second"`
								CurrentIngressDirectRatePerSecond        float64 `xml:"current-ingress-direct-rate-per-second"`
								CurrentEgressDirectRatePerSecond         float64 `xml:"current-egress-direct-rate-per-second"`

								// Msg-Rate average (per-minute)
								AverageIngressRatePerMinute              float64 `xml:"average-ingress-rate-per-minute"`
								AverageEgressRatePerMinute               float64 `xml:"average-egress-rate-per-minute"`
								AverageIngressPersistentRatePerMinute    float64 `xml:"average-ingress-persistent-rate-per-minute"`
								AverageEgressPersistentRatePerMinute     float64 `xml:"average-egress-persistent-rate-per-minute"`
								AverageIngressNonpersistentRatePerMinute float64 `xml:"average-ingress-nonpersistent-rate-per-minute"`
								AverageEgressNonpersistentRatePerMinute  float64 `xml:"average-egress-nonpersistent-rate-per-minute"`
								AverageIngressDirectRatePerMinute        float64 `xml:"average-ingress-direct-rate-per-minute"`
								AverageEgressDirectRatePerMinute         float64 `xml:"average-egress-direct-rate-per-minute"`

								// Byte-Rate current (per-second)
								CurrentIngressByteRatePerSecond              float64 `xml:"current-ingress-byte-rate-per-second"`
								CurrentEgressByteRatePerSecond               float64 `xml:"current-egress-byte-rate-per-second"`
								CurrentIngressPersistentByteRatePerSecond    float64 `xml:"current-ingress-persistent-byte-rate-per-second"`
								CurrentEgressPersistentByteRatePerSecond     float64 `xml:"current-egress-persistent-byte-rate-per-second"`
								CurrentIngressNonpersistentByteRatePerSecond float64 `xml:"current-ingress-nonpersistent-byte-rate-per-second"`
								CurrentEgressNonpersistentByteRatePerSecond  float64 `xml:"current-egress-nonpersistent-byte-rate-per-second"`
								CurrentIngressDirectByteRatePerSecond        float64 `xml:"current-ingress-direct-byte-rate-per-second"`
								CurrentEgressDirectByteRatePerSecond         float64 `xml:"current-egress-direct-byte-rate-per-second"`

								// Byte-Rate average (per-minute)
								AverageIngressByteRatePerMinute              float64 `xml:"average-ingress-byte-rate-per-minute"`
								AverageEgressByteRatePerMinute               float64 `xml:"average-egress-byte-rate-per-minute"`
								AverageIngressPersistentByteRatePerMinute    float64 `xml:"average-ingress-persistent-byte-rate-per-minute"`
								AverageEgressPersistentByteRatePerMinute     float64 `xml:"average-egress-persistent-byte-rate-per-minute"`
								AverageIngressNonpersistentByteRatePerMinute float64 `xml:"average-ingress-nonpersistent-byte-rate-per-minute"`
								AverageEgressNonpersistentByteRatePerMinute  float64 `xml:"average-egress-nonpersistent-byte-rate-per-minute"`
								AverageIngressDirectByteRatePerMinute        float64 `xml:"average-ingress-direct-byte-rate-per-minute"`
								AverageEgressDirectByteRatePerMinute         float64 `xml:"average-egress-direct-byte-rate-per-minute"`
							} `xml:"stats"`
						} `xml:"global"`
					} `xml:"client"`
				} `xml:"stats"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	command := "<rpc><show><stats><client><detail/></client></stats></show></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "GetStatsClientDetailSemp1", 1)
	if err != nil {
		semp.logger.Error("Can't scrape GetStatsClientDetailSemp1", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer func() { _ = body.Close() }()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		semp.logger.Error("Can't decode Xml GetStatsClientDetailSemp1", "err", err, "broker", semp.brokerURI)
		return 0, err
	}
	if err := target.ExecuteResult.OK(); err != nil {
		semp.logger.Error(
			"unexpected result",
			"command", command,
			"result", target.ExecuteResult.Result,
			"reason", target.ExecuteResult.Reason,
			"broker", semp.brokerURI,
		)
		return 0, err
	}

	// Msg-Rate current (per-second)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["current_ingress_rate_per_second"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.CurrentIngressRatePerSecond)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["current_egress_rate_per_second"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.CurrentEgressRatePerSecond)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["current_ingress_persistent_rate_per_second"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.CurrentIngressPersistentRatePerSecond)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["current_egress_persistent_rate_per_second"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.CurrentEgressPersistentRatePerSecond)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["current_ingress_nonpersistent_rate_per_second"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.CurrentIngressNonpersistentRatePerSecond)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["current_egress_nonpersistent_rate_per_second"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.CurrentEgressNonpersistentRatePerSecond)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["current_ingress_direct_rate_per_second"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.CurrentIngressDirectRatePerSecond)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["current_egress_direct_rate_per_second"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.CurrentEgressDirectRatePerSecond)

	// Msg-Rate average (per-minute)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["average_ingress_rate_per_minute"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.AverageIngressRatePerMinute)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["average_egress_rate_per_minute"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.AverageEgressRatePerMinute)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["average_ingress_persistent_rate_per_minute"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.AverageIngressPersistentRatePerMinute)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["average_egress_persistent_rate_per_minute"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.AverageEgressPersistentRatePerMinute)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["average_ingress_nonpersistent_rate_per_minute"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.AverageIngressNonpersistentRatePerMinute)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["average_egress_nonpersistent_rate_per_minute"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.AverageEgressNonpersistentRatePerMinute)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["average_ingress_direct_rate_per_minute"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.AverageIngressDirectRatePerMinute)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["average_egress_direct_rate_per_minute"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.AverageEgressDirectRatePerMinute)

	// Byte-Rate current (per-second)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["current_ingress_byte_rate_per_second"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.CurrentIngressByteRatePerSecond)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["current_egress_byte_rate_per_second"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.CurrentEgressByteRatePerSecond)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["current_ingress_persistent_byte_rate_per_second"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.CurrentIngressPersistentByteRatePerSecond)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["current_egress_persistent_byte_rate_per_second"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.CurrentEgressPersistentByteRatePerSecond)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["current_ingress_nonpersistent_byte_rate_per_second"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.CurrentIngressNonpersistentByteRatePerSecond)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["current_egress_nonpersistent_byte_rate_per_second"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.CurrentEgressNonpersistentByteRatePerSecond)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["current_ingress_direct_byte_rate_per_second"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.CurrentIngressDirectByteRatePerSecond)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["current_egress_direct_byte_rate_per_second"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.CurrentEgressDirectByteRatePerSecond)

	// Byte-Rate average (per-minute)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["average_ingress_byte_rate_per_minute"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.AverageIngressByteRatePerMinute)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["average_egress_byte_rate_per_minute"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.AverageEgressByteRatePerMinute)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["average_ingress_persistent_byte_rate_per_minute"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.AverageIngressPersistentByteRatePerMinute)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["average_egress_persistent_byte_rate_per_minute"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.AverageEgressPersistentByteRatePerMinute)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["average_ingress_nonpersistent_byte_rate_per_minute"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.AverageIngressNonpersistentByteRatePerMinute)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["average_egress_nonpersistent_byte_rate_per_minute"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.AverageEgressNonpersistentByteRatePerMinute)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["average_ingress_direct_byte_rate_per_minute"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.AverageIngressDirectByteRatePerMinute)
	ch <- semp.NewMetric(MetricDesc["StatsClientDetail"]["average_egress_direct_byte_rate_per_minute"], prometheus.GaugeValue, target.RPC.Show.Stats.Client.Global.Stats.AverageEgressDirectByteRatePerMinute)
	return 1, nil
}
