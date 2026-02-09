package semp

import (
	"encoding/xml"
	"math"
	"solace_exporter/internal/semp/types"
	"strconv"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// GetSpoolSemp1 Get system-wide spool information
func (semp *Semp) GetSpoolSemp1(ch chan<- PrometheusMetric) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Spool struct {
					Info struct {
						MessageCountUtilPercentage      string  `xml:"message-count-utilization-percentage"`
						QuotaDiskUsage                  float64 `xml:"max-disk-usage"`
						QuotaMsgCount                   string  `xml:"max-message-count"`
						PersistUsage                    float64 `xml:"current-persist-usage"`
						TotalMsgCount                   float64 `xml:"total-messages-currently-spooled"`
						ActiveDiskPartitionUsage        string  `xml:"active-disk-partition-usage"`        // May be "-"
						MateDiskPartitionUsage          string  `xml:"mate-disk-partition-usage"`          // May be "-"
						SpoolFilesUtilizationPercentage string  `xml:"spool-files-utilization-percentage"` // May be "-"
						SpoolSyncStatus                 string  `xml:"synchronization-status"`
						TransactedSessionUtilisation    string  `xml:"transacted-session-count-utilization-percentage"`

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
						CurrentDiskUsage               float64 `xml:"current-disk-usage"`
						MessagesCurrentlySpooledADB    float64 `xml:"rfad-messages-currently-spooled"`
						MessagesCurrentlySpooledDisk   float64 `xml:"disk-messages-currently-spooled"`

						QueueTopicSubscriptionsQuota float64 `xml:"max-queue-topic-subscriptions"`
						QueueTopicSubscriptionsUsed  float64 `xml:"queue-topic-subscriptions-used"`

                        DefragScheduleEnabled           bool    `xml:"defrag-schedule-enabled"`
                        DefragThresholdEnabled          bool    `xml:"defrag-threshold-enabled"`
                        DefragThresholdFragPercentage   float64 `xml:"defrag-threshold-spool-fragmentation-percentage"`
                        DefragThresholdUsagePercentage  float64 `xml:"defrag-threshold-spool-usage-percentage"`
                        DefragEstimatedFragPercentage   float64 `xml:"defrag-est-fragmentation-percentage"`
                        DefragEstimatedRecoverableSpace float64 `xml:"defrag-est-recoverable-space"`

                        DiskInfos                       struct {
                            DiskInfo                    []struct{
                                Partition               string  `xml:"partition"`
                                PartitionBlocks         float64 `xml:"blocks"`
                                PartitionUsed           float64 `xml:"used"`
                                PartitionAvailable      float64 `xml:"available"`
                                PartitionUsePercentage  string  `xml:"use"`
                                PartitionMountedOn      string  `xml:"mounted-on"`
                            } `xml:"disk-info"`
                        } `xml:"disk-infos"`
					} `xml:"message-spool-info"`
				} `xml:"message-spool"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	command := "<rpc><show><message-spool><detail/></message-spool></show ></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "SpoolSemp1", 1)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't scrape Solace", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't decode Xml", "err", err, "broker", semp.brokerURI)
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

	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_quota_bytes"], prometheus.GaugeValue, math.Round(target.RPC.Show.Spool.Info.QuotaDiskUsage*1048576.0))
	// MaxMsgCount is in the form "100M"
	s1 := target.RPC.Show.Spool.Info.QuotaMsgCount[:len(target.RPC.Show.Spool.Info.QuotaMsgCount)-1]
	if value, err := strconv.ParseFloat(s1, 64); err == nil {
		ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_quota_msgs"], prometheus.GaugeValue, value*1000000)
	}
	if value, err := strconv.ParseFloat(target.RPC.Show.Spool.Info.ActiveDiskPartitionUsage, 64); err == nil {
		ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_disk_partition_usage_active_percent"], prometheus.GaugeValue, math.Round(value))
	}
	if value, err := strconv.ParseFloat(target.RPC.Show.Spool.Info.MateDiskPartitionUsage, 64); err == nil {
		ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_disk_partition_usage_mate_percent"], prometheus.GaugeValue, math.Round(value))
	}
	if value, err := strconv.ParseFloat(target.RPC.Show.Spool.Info.SpoolFilesUtilizationPercentage, 64); err == nil {
		ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_files_utilization_percent"], prometheus.GaugeValue, math.Round(value))
	}

	if value, err := strconv.ParseFloat(target.RPC.Show.Spool.Info.MessageCountUtilPercentage, 64); err == nil {
		ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_message_count_utilization_percent"], prometheus.GaugeValue, math.Round(value))
	}

	if value, err := strconv.ParseFloat(target.RPC.Show.Spool.Info.TransactedSessionUtilisation, 64); err == nil {
		ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_transacted_session_utilisation_pct"], prometheus.GaugeValue, math.Round(value))
	}

	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_usage_bytes"], prometheus.GaugeValue, math.Round(target.RPC.Show.Spool.Info.PersistUsage*1048576.0))
	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_usage_adb_bytes"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.CurrentPersistentStoreUsageADB*1048576.0)
	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_usage_msgs"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.TotalMsgCount)
	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_ingress_flows_quota"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.IngressFlowsQuota)
	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_ingress_flows_count"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.IngressFlowsCount)

	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_egress_flows_quota"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.EgressFlowsQuota)
	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_egress_flows_count"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.EgressFlowsActive+target.RPC.Show.Spool.Info.EgressFlowsInactive+target.RPC.Show.Spool.Info.EgressFlowsBrowser)
	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_egress_flows_active"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.EgressFlowsActive)
	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_egress_flows_inactive"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.EgressFlowsInactive)
	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_egress_flows_browser"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.EgressFlowsBrowser)

	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_endpoints_quota"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.EntitiesByQendptQuota)
	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_endpoints_queue"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.EntitiesByQendptQueue)
	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_endpoints_dte"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.EntitiesByQendptDte)

	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_transacted_sessions_quota"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.TransactedSessionsQuota)
	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_transacted_sessions_used"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.TransactedSessionsUsed)
	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_queue_topic_subscriptions_quota"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.QueueTopicSubscriptionsQuota)
	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_queue_topic_subscriptions_used"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.QueueTopicSubscriptionsUsed)

	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_transactions_quota"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.TransactionsQuota)
	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_transactions_used"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.TransactionsUsed)

	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_messages_currently_spooled_adb"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.MessagesCurrentlySpooledADB)
	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_messages_currently_spooled_disk"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.MessagesCurrentlySpooledDisk)

	// this is probably more useful for appliances where ADB storage is independent of disk utilisation
	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_messages_total_disk_usage_bytes"], prometheus.GaugeValue, math.Round(target.RPC.Show.Spool.Info.CurrentDiskUsage*1048576.0))
	// I have been unable to ascertain what the error values for this metric are
	ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_sync_status"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Spool.Info.SpoolSyncStatus, []string{"Synced"}))

    ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_defrag_schedule_enabled"], prometheus.GaugeValue, encodeMetricBool(target.RPC.Show.Spool.Info.DefragScheduleEnabled))
    ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_defrag_threshold_enabled"], prometheus.GaugeValue, encodeMetricBool(target.RPC.Show.Spool.Info.DefragThresholdEnabled))
    ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_defrag_threshold_frag_percent"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.DefragThresholdFragPercentage)
    ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_defrag_threshold_usage_percent"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.DefragThresholdUsagePercentage)
    ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_defrag_estimated_frag_percent"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.DefragEstimatedFragPercentage)
    ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_defrag_estimated_recoverable_space"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.DefragEstimatedRecoverableSpace)

    for _, diskInfo := range target.RPC.Show.Spool.Info.DiskInfos.DiskInfo {
        partition := diskInfo.Partition
        ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_disk_partition_blocks"], prometheus.GaugeValue, diskInfo.PartitionBlocks, partition)
        ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_disk_partition_used"], prometheus.GaugeValue, diskInfo.PartitionUsed, partition)
        ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_disk_partition_available"], prometheus.GaugeValue, diskInfo.PartitionAvailable, partition)
        if value, err := strconv.ParseFloat(diskInfo.PartitionUsePercentage, 64); err == nil {
            ch <- semp.NewMetric(MetricDesc["Spool"]["system_spool_disk_partition_use_percent"], prometheus.GaugeValue, math.Round(value), partition)
        }
    }

	return 1, nil
}
