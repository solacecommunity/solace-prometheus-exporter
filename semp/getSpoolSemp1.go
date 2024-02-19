package semp

import (
	"encoding/xml"
	"errors"
	"math"
	"strconv"

	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// Get system-wide spool information
func (e *Semp) GetSpoolSemp1(ch chan<- PrometheusMetric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Spool struct {
					Info struct {
						MessageCountUtilPercentage      string  `xml:"message-count-utilization-percentage"`
						QuotaDiskUsage                  float64 `xml:"max-disk-usage"`
						QuotaMsgCount                   string  `xml:"max-message-count"`
						PersistUsage                    float64 `xml:"current-persist-usage"`
						PersistMsgCount                 float64 `xml:"total-messages-currently-spooled"`
						ActiveDiskPartitionUsage        string  `xml:"active-disk-partition-usage"`        // May be "-"
						MateDiskPartitionUsage          string  `xml:"mate-disk-partition-usage"`          // May be "-"
						SpoolFilesUtilizationPercentage string  `xml:"spool-files-utilization-percentage"` // May be "-"
						SpoolSyncStatus                 string  `xml:"spool-sync-status"`

						IngressFlowsQuota   float64 `xml:"ingress-flows-allowed"`
						IngressFlowsCount   float64 `xml:"ingress-flow-count"`
						EgressFlowsQuota    float64 `xml:"flows-allowed"`
						EgressFlowsActive   float64 `xml:"active-flow-count"`
						EgressFlowsInactive float64 `xml:"inactive-flow-count"`
						EgressFlowsBrowser  float64 `xml:"browser-flow-count"`

						EntitiesByQendptQuota float64 `xml:"message-spool-entities-allowed-by-qendpt"`
						EntitiesByQendptQueue float64 `xml:"message-spool-entities-used-by-queue"`
						EntitiesByQendptDte   float64 `xml:"message-spool-entities-used-by-dte"`

						TransactedSessionsQuota float64 `xml:"max-transacted-sessions"`
						TransactedSessionsUsed  float64 `xml:"transacted-sessions-used"`

						TransactionsQuota float64 `xml:"max-transactions"`
						TransactionsUsed  float64 `xml:"transactions-used"`

						CurrentPersistentStoreUsageADB float64 `xml:"current-rfad-usage"`
						MessagesCurrentlySpooledADB    float64 `xml:"rfad-messages-currently-spooled"`
						MessagesCurrentlySpooledDisk   float64 `xml:"disk-messages-currently-spooled"`
					} `xml:"message-spool-info"`
				} `xml:"message-spool"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><message-spool><detail/></message-spool></show ></rpc>"
	body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", command, "SpoolSemp1", 1)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape Solace", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
		return 0, errors.New("unexpected result: see log")
	}

	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_quota_bytes"], prometheus.GaugeValue, math.Round(target.RPC.Show.Spool.Info.QuotaDiskUsage*1048576.0))
	// MaxMsgCount is in the form "100M"
	s1 := target.RPC.Show.Spool.Info.QuotaMsgCount[:len(target.RPC.Show.Spool.Info.QuotaMsgCount)-1]
	if value, err := strconv.ParseFloat(s1, 64); err == nil {
		ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_quota_msgs"], prometheus.GaugeValue, value*1000000)
	}
	if value, err := strconv.ParseFloat(target.RPC.Show.Spool.Info.ActiveDiskPartitionUsage, 64); err == nil {
		ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_disk_partition_usage_active_percent"], prometheus.GaugeValue, math.Round(value))
	}
	if value, err := strconv.ParseFloat(target.RPC.Show.Spool.Info.MateDiskPartitionUsage, 64); err == nil {
		ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_disk_partition_usage_mate_percent"], prometheus.GaugeValue, math.Round(value))
	}
	if value, err := strconv.ParseFloat(target.RPC.Show.Spool.Info.SpoolFilesUtilizationPercentage, 64); err == nil {
		ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_files_utilization_percent"], prometheus.GaugeValue, math.Round(value))
	}

	if value, err := strconv.ParseFloat(target.RPC.Show.Spool.Info.MessageCountUtilPercentage, 64); err == nil {
		ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_message_count_utilization_percent"], prometheus.GaugeValue, math.Round(value))
	}

	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_usage_bytes"], prometheus.GaugeValue, math.Round(target.RPC.Show.Spool.Info.PersistUsage*1048576.0))
	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_usage_adb_bytes"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.CurrentPersistentStoreUsageADB*1048576.0)
	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_usage_msgs"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.PersistMsgCount)

	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_ingress_flows_quota"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.IngressFlowsQuota)
	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_ingress_flows_count"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.IngressFlowsCount)

	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_egress_flows_quota"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.EgressFlowsQuota)
	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_egress_flows_count"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.EgressFlowsActive+target.RPC.Show.Spool.Info.EgressFlowsInactive+target.RPC.Show.Spool.Info.EgressFlowsBrowser)
	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_egress_flows_active"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.EgressFlowsActive)
	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_egress_flows_inactive"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.EgressFlowsInactive)
	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_egress_flows_browser"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.EgressFlowsBrowser)

	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_endpoints_quota"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.EntitiesByQendptQuota)
	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_endpoints_queue"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.EntitiesByQendptQueue)
	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_endpoints_dte"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.EntitiesByQendptDte)

	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_transacted_sessions_quota"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.TransactedSessionsQuota)
	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_transacted_sessions_used"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.TransactedSessionsUsed)
	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_transactions_quota"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.TransactionsQuota)
	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_transactions_used"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.TransactionsUsed)

	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_messages_currently_spooled_adb"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.MessagesCurrentlySpooledADB)
	ch <- e.NewMetric(MetricDesc["Spool"]["system_spool_messages_currently_spooled_disk"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.MessagesCurrentlySpooledDisk)

	return 1, nil
}
