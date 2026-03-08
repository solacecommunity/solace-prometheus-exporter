package semp

import (
	"encoding/xml"
	"solace_exporter/internal/semp/types"

	"github.com/prometheus/client_golang/prometheus"
)

// GetSpoolStatsSemp1 Get system-wide spool statistics
func (semp *Semp) GetSpoolStatsSemp1(ch chan<- PrometheusMetric) (float64, error) {
	type Data struct {
		RPC struct {
			Show struct {
				Spool struct {
					Stats struct {
						VpnName                                  string  `xml:"vpn-name"`
						DiscardSpoolingNotReady                  float64 `xml:"discard-spooling-not-ready"`
						DiscardOoo                               float64 `xml:"discard-ooo"`
						DiscardDuplicate                         float64 `xml:"discard-duplicate"`
						DiscardNodest                            float64 `xml:"discard-nodest"`
						DiscardSpoolOverQuota                    float64 `xml:"discard-spool-over-quota"`
						DiscardQendptOverQuota                   float64 `xml:"discard-qendpt-over-quota"`
						DiscardReplayLogOverQuota                float64 `xml:"discard-replay-log-over-quota"`
						DiscardMaxMsgUsageExceeded               float64 `xml:"discard-max-msg-usage-exceeded"`
						DiscardMaxMsgSizeExceeded                float64 `xml:"discard-max-msg-size-exceeded"`
						DiscardRemoteRouterSpoolingNotSupported  float64 `xml:"discard-remote-router-spooling-not-supported"`
						DiscardSpoolToAdbFail                    float64 `xml:"discard-spool-to-adb-fail"`
						DiscardSpoolToDiskFail                   float64 `xml:"discard-spool-to-disk-fail"`
						DiscardSpoolFileLimitExceeded            float64 `xml:"discard-spool-file-limit-exceeded"`
						DiscardErroredMessage                    float64 `xml:"discard-errored-message"`
						DiscardQueueNotFound                     float64 `xml:"discard-queue-not-found"`
						SpoolShutdownDiscard                     float64 `xml:"spool-shutdown-discard"`
						UserProfileDenyGuaranteed                float64 `xml:"user-profile-deny-guaranteed"`
						DiscardPublisherNotFound                 float64 `xml:"discard-publisher-not-found"`
						NoLocalDeliveryDiscard                   float64 `xml:"no-local-delivery-discard"`
						SmfTtlExceeded                           float64 `xml:"smf-ttl-exceeded"`
						PublishAclDenied                         float64 `xml:"publish-acl-denied"`
						DestinationGroupError                    float64 `xml:"destination-group-error"`
						NotCompatibleWithForwardingMode          float64 `xml:"not-compatible-with-forwarding-mode"`
						LowPriorityMsgCongestionDiscard          float64 `xml:"low-priority-msg-congestion-discard"`
						ReplicationIsStandbyDiscard              float64 `xml:"replication-is-standby-discard"`
						SyncReplicationIneligibleDiscard         float64 `xml:"sync-replication-ineligible-discard"`
						XaTransactionNotSupported                float64 `xml:"xa-transaction-not-supported"`
						DiscardOther                             float64 `xml:"discard-other"`
						TotalDeletedMessages                     float64 `xml:"total-deleted-messages"`
						TotalTtlExpiredDiscardMessages           float64 `xml:"total-ttl-expired-discard-messages"`
						TotalTtlExpiredToDmqMessages             float64 `xml:"total-ttl-expired-to-dmq-messages"`
						TotalTtlExpiredToDmqFailures             float64 `xml:"total-ttl-expired-to-dmq-failures"`
						MaxRedeliveryExceededDiscardMessages     float64 `xml:"max-redelivery-exceeded-discard-messages"`
						MaxRedeliveryExceededToDmqMessages       float64 `xml:"max-redelivery-exceeded-to-dmq-messages"`
						MaxRedeliveryExceededToDmqFailures       float64 `xml:"max-redelivery-exceeded-to-dmq-failures"`
						TotalTtlExceededDiscardMessages          float64 `xml:"total-ttl-exceeded-discard-messages"`
						IngressMessages                          float64 `xml:"ingress-messages"`
						IngressMessagesPromoted                  float64 `xml:"ingress-messages-promoted"`
						IngressMessagesDemoted                   float64 `xml:"ingress-messages-demoted"`
						PromotedMessagesReplicated               float64 `xml:"promoted-messages-replicated"`
						IngressMessagesAsyncReplicated           float64 `xml:"ingress-messages-async-replicated"`
						IngressMessagesSyncReplicated            float64 `xml:"ingress-messages-sync-replicated"`
						IngressMessagesFromReplicationMate       float64 `xml:"ingress-messages-from-replication-mate"`
						IngressMessagesCopiedToReplayLog         float64 `xml:"ingress-messages-copied-to-replay-log"`
						SequencedTopicMatches                    float64 `xml:"sequenced-topic-matches"`
						SeqNumAlreadyAssigned                    float64 `xml:"seq-num-already-assigned"`
						SeqNumRollover                           float64 `xml:"seq-num-rollover"`
						SeqNumMessagesDiscarded                  float64 `xml:"seq-num-messages-discarded"`
						TransactedMessagesNotSequenced           float64 `xml:"transacted-messages-not-sequenced"`
						TotalDiscardedMessages                   float64 `xml:"total-discarded-messages"`
						SpooledToAdb                             float64 `xml:"spooled-to-adb"`
						SpooledToDisk                            float64 `xml:"spooled-to-disk"`
						RetrieveFromAdb                          float64 `xml:"retrieve-from-adb"`
						RetrieveFromDisk                         float64 `xml:"retrieve-from-disk"`
						TotalGuaranteedMessageCacheMisses        float64 `xml:"total-guaranteed-message-cache-misses"`
						TotalIngressSelectorMatchMessages        float64 `xml:"total-ingress-selector-match-messages"`
						TotalIngressSelectorMismatchMessages     float64 `xml:"total-ingress-selector-mismatch-messages"`
						TotalEgressSelectorMatchMessages         float64 `xml:"total-egress-selector-match-messages"`
						TotalEgressSelectorMismatchMessages      float64 `xml:"total-egress-selector-mismatch-messages"`
						TotalDiscardedEgressMessages             float64 `xml:"total-discarded-egress-messages"`
						EgressMessages                           float64 `xml:"egress-messages"`
						EgressMessagesRedelivered                float64 `xml:"egress-messages-redelivered"`
						EgressMessagesTransportRetransmit        float64 `xml:"egress-messages-transport-retransmit"`
						ConfirmedDelivered                       float64 `xml:"confirmed-delivered"`
						ConfirmedDeliveredStoreAndForward        float64 `xml:"confirmed-delivered-store-and-forward"`
						ConfirmedDeliveredCutThrough             float64 `xml:"confirmed-delivered-cut-through"`
						ConfirmedDeliveredFromReplicationMate    float64 `xml:"confirmed-delivered-from-replication-mate"`
						RequestForRedelivery                     float64 `xml:"request-for-redelivery"`
						OpenSession                              float64 `xml:"open-session"`
						OpenSessionSuccess                       float64 `xml:"open-session-success"`
						OpenSessionMaxSessionsExceeded           float64 `xml:"open-session-max-sessions-exceeded"`
						OpenSessionOtherFailures                 float64 `xml:"open-session-other-failures"`
						Transactions                             float64 `xml:"transactions"`
						TransactionsSuccess                      float64 `xml:"transactions-success"`
						TransactionsCommit                       float64 `xml:"transactions-commit"`
						TransactionsRollback                     float64 `xml:"transactions-rollback"`
						TransactionsFail                         float64 `xml:"transactions-fail"`
						TransactionsMsgsSpooledToAdb             float64 `xml:"transactions-msgs-spooled-to-adb"`
						TransactionsMsgsRetrievedFromAdbOrDisk   float64 `xml:"transactions-msgs-retrieved-from-adb-or-disk"`
						TransactionsMsgsPublished                float64 `xml:"transactions-msgs-published"`
						TransactionsMsgsConsumed                 float64 `xml:"transactions-msgs-consumed"`
						MaxTransactionsExceeded                  float64 `xml:"max-transactions-exceeded"`
						MaxTransactionResourcesExceeded          float64 `xml:"max-transaction-resources-exceeded"`
						XaOpenSession                            float64 `xml:"xa-open-session"`
						XaOpenSessionSuccess                     float64 `xml:"xa-open-session-success"`
						XaOpenSessionMaxSessionsExceeded         float64 `xml:"xa-open-session-max-sessions-exceeded"`
						XaOpenSessionOtherFailures               float64 `xml:"xa-open-session-other-failures"`
						XaTransactions                           float64 `xml:"xa-transactions"`
						XaTransactionsSuccess                    float64 `xml:"xa-transactions-success"`
						XaTransactionsFail                       float64 `xml:"xa-transactions-fail"`
						XaTransactionsMsgsSpooledToAdb           float64 `xml:"xa-transactions-msgs-spooled-to-adb"`
						XaTransactionsMsgsRetrievedFromAdbOrDisk float64 `xml:"xa-transactions-msgs-retrieved-from-adb-or-disk"`
						XaTransactionsMsgsPublished              float64 `xml:"xa-transactions-msgs-published"`
						XaTransactionsMsgsConsumed               float64 `xml:"xa-transactions-msgs-consumed"`
						XaMaxTransactionsExceeded                float64 `xml:"xa-max-transactions-exceeded"`
						XaMaxTransactionResourcesExceeded        float64 `xml:"xa-max-transaction-resources-exceeded"`
						ReplaysInitiated                         float64 `xml:"replays-initiated"`
						ReplaysSucceeded                         float64 `xml:"replays-succeeded"`
						ReplaysFailed                            float64 `xml:"replays-failed"`
						ReplayedMessagesSent                     float64 `xml:"replayed-messages-sent"`
						ReplayedMessagesAcked                    float64 `xml:"replayed-messages-acked"`
						CurrentBindRatePerSecond                 float64 `xml:"current-bind-rate-per-second"`
						AverageBindRatePerMinute                 float64 `xml:"average-bind-rate-per-minute"`
					} `xml:"message-spool-stats"`
				} `xml:"message-spool"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	command := "<rpc><show><message-spool><stats/></message-spool></show></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "SpoolStatsSemp1", 1)
	if err != nil {
		semp.logger.Error("Can't scrape Solace", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		semp.logger.Error("Can't decode Xml", "err", err, "broker", semp.brokerURI)
		return 0, err
	}
	if err := target.ExecuteResult.OK(); err != nil {
		semp.logger.Error("unexpected result",
			"command", command,
			"result", target.ExecuteResult.Result,
			"reason", target.ExecuteResult.Reason,
			"broker", semp.brokerURI,
		)
		return 0, err
	}

	stats := target.RPC.Show.Spool.Stats

	// Discard counters
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_discard_spooling_not_ready"], prometheus.CounterValue, stats.DiscardSpoolingNotReady)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_discard_out_of_order"], prometheus.CounterValue, stats.DiscardOoo)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_discard_duplicate"], prometheus.CounterValue, stats.DiscardDuplicate)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_discard_no_destination"], prometheus.CounterValue, stats.DiscardNodest)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_discard_spool_over_quota"], prometheus.CounterValue, stats.DiscardSpoolOverQuota)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_discard_queue_endpoint_over_quota"], prometheus.CounterValue, stats.DiscardQendptOverQuota)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_discard_replay_log_over_quota"], prometheus.CounterValue, stats.DiscardReplayLogOverQuota)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_discard_max_msg_usage_exceeded"], prometheus.CounterValue, stats.DiscardMaxMsgUsageExceeded)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_discard_max_msg_size_exceeded"], prometheus.CounterValue, stats.DiscardMaxMsgSizeExceeded)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_discard_remote_router_spooling_not_supported"], prometheus.CounterValue, stats.DiscardRemoteRouterSpoolingNotSupported)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_discard_spool_to_adb_fail"], prometheus.CounterValue, stats.DiscardSpoolToAdbFail)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_discard_spool_to_disk_fail"], prometheus.CounterValue, stats.DiscardSpoolToDiskFail)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_discard_spool_file_limit_exceeded"], prometheus.CounterValue, stats.DiscardSpoolFileLimitExceeded)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_discard_errored_message"], prometheus.CounterValue, stats.DiscardErroredMessage)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_discard_queue_not_found"], prometheus.CounterValue, stats.DiscardQueueNotFound)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_spool_shutdown_discard"], prometheus.CounterValue, stats.SpoolShutdownDiscard)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_user_profile_deny_guaranteed"], prometheus.CounterValue, stats.UserProfileDenyGuaranteed)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_discard_publisher_not_found"], prometheus.CounterValue, stats.DiscardPublisherNotFound)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_no_local_delivery_discard"], prometheus.CounterValue, stats.NoLocalDeliveryDiscard)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_smf_ttl_exceeded"], prometheus.CounterValue, stats.SmfTtlExceeded)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_publish_acl_denied"], prometheus.CounterValue, stats.PublishAclDenied)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_destination_group_error"], prometheus.CounterValue, stats.DestinationGroupError)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_not_compatible_with_forwarding_mode"], prometheus.CounterValue, stats.NotCompatibleWithForwardingMode)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_low_priority_msg_congestion_discard"], prometheus.CounterValue, stats.LowPriorityMsgCongestionDiscard)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_replication_is_standby_discard"], prometheus.CounterValue, stats.ReplicationIsStandbyDiscard)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_sync_replication_ineligible_discard"], prometheus.CounterValue, stats.SyncReplicationIneligibleDiscard)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_xa_transaction_not_supported"], prometheus.CounterValue, stats.XaTransactionNotSupported)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_discard_other"], prometheus.CounterValue, stats.DiscardOther)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_total_discarded_messages"], prometheus.CounterValue, stats.TotalDiscardedMessages)

	// TTL and redelivery counters
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_total_deleted_messages"], prometheus.CounterValue, stats.TotalDeletedMessages)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_total_ttl_expired_discard_messages"], prometheus.CounterValue, stats.TotalTtlExpiredDiscardMessages)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_total_ttl_expired_to_dmq_messages"], prometheus.CounterValue, stats.TotalTtlExpiredToDmqMessages)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_total_ttl_expired_to_dmq_failures"], prometheus.CounterValue, stats.TotalTtlExpiredToDmqFailures)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_max_redelivery_exceeded_discard_messages"], prometheus.CounterValue, stats.MaxRedeliveryExceededDiscardMessages)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_max_redelivery_exceeded_to_dmq_messages"], prometheus.CounterValue, stats.MaxRedeliveryExceededToDmqMessages)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_max_redelivery_exceeded_to_dmq_failures"], prometheus.CounterValue, stats.MaxRedeliveryExceededToDmqFailures)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_total_ttl_exceeded_discard_messages"], prometheus.CounterValue, stats.TotalTtlExceededDiscardMessages)

	// Ingress counters
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_ingress_messages"], prometheus.CounterValue, stats.IngressMessages)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_ingress_messages_promoted"], prometheus.CounterValue, stats.IngressMessagesPromoted)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_ingress_messages_demoted"], prometheus.CounterValue, stats.IngressMessagesDemoted)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_promoted_messages_replicated"], prometheus.CounterValue, stats.PromotedMessagesReplicated)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_ingress_messages_async_replicated"], prometheus.CounterValue, stats.IngressMessagesAsyncReplicated)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_ingress_messages_sync_replicated"], prometheus.CounterValue, stats.IngressMessagesSyncReplicated)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_ingress_messages_from_replication_mate"], prometheus.CounterValue, stats.IngressMessagesFromReplicationMate)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_ingress_messages_copied_to_replay_log"], prometheus.CounterValue, stats.IngressMessagesCopiedToReplayLog)

	// Sequence number stats
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_sequenced_topic_matches"], prometheus.CounterValue, stats.SequencedTopicMatches)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_seq_num_already_assigned"], prometheus.CounterValue, stats.SeqNumAlreadyAssigned)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_seq_num_rollover"], prometheus.CounterValue, stats.SeqNumRollover)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_seq_num_messages_discarded"], prometheus.CounterValue, stats.SeqNumMessagesDiscarded)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_transacted_messages_not_sequenced"], prometheus.CounterValue, stats.TransactedMessagesNotSequenced)

	// ADB/Disk stats
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_spooled_to_adb"], prometheus.CounterValue, stats.SpooledToAdb)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_spooled_to_disk"], prometheus.CounterValue, stats.SpooledToDisk)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_retrieve_from_adb"], prometheus.CounterValue, stats.RetrieveFromAdb)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_retrieve_from_disk"], prometheus.CounterValue, stats.RetrieveFromDisk)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_total_guaranteed_message_cache_misses"], prometheus.CounterValue, stats.TotalGuaranteedMessageCacheMisses)

	// Selector stats
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_total_ingress_selector_match_messages"], prometheus.CounterValue, stats.TotalIngressSelectorMatchMessages)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_total_ingress_selector_mismatch_messages"], prometheus.CounterValue, stats.TotalIngressSelectorMismatchMessages)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_total_egress_selector_match_messages"], prometheus.CounterValue, stats.TotalEgressSelectorMatchMessages)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_total_egress_selector_mismatch_messages"], prometheus.CounterValue, stats.TotalEgressSelectorMismatchMessages)

	// Egress stats
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_total_discarded_egress_messages"], prometheus.CounterValue, stats.TotalDiscardedEgressMessages)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_egress_messages"], prometheus.CounterValue, stats.EgressMessages)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_egress_messages_redelivered"], prometheus.CounterValue, stats.EgressMessagesRedelivered)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_egress_messages_transport_retransmit"], prometheus.CounterValue, stats.EgressMessagesTransportRetransmit)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_confirmed_delivered"], prometheus.CounterValue, stats.ConfirmedDelivered)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_confirmed_delivered_store_and_forward"], prometheus.CounterValue, stats.ConfirmedDeliveredStoreAndForward)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_confirmed_delivered_cut_through"], prometheus.CounterValue, stats.ConfirmedDeliveredCutThrough)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_confirmed_delivered_from_replication_mate"], prometheus.CounterValue, stats.ConfirmedDeliveredFromReplicationMate)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_request_for_redelivery"], prometheus.CounterValue, stats.RequestForRedelivery)

	// Session stats
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_open_session"], prometheus.CounterValue, stats.OpenSession)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_open_session_success"], prometheus.CounterValue, stats.OpenSessionSuccess)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_open_session_max_sessions_exceeded"], prometheus.CounterValue, stats.OpenSessionMaxSessionsExceeded)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_open_session_other_failures"], prometheus.CounterValue, stats.OpenSessionOtherFailures)

	// Transaction stats
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_transactions"], prometheus.CounterValue, stats.Transactions)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_transactions_success"], prometheus.CounterValue, stats.TransactionsSuccess)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_transactions_commit"], prometheus.CounterValue, stats.TransactionsCommit)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_transactions_rollback"], prometheus.CounterValue, stats.TransactionsRollback)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_transactions_fail"], prometheus.CounterValue, stats.TransactionsFail)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_transactions_msgs_spooled_to_adb"], prometheus.CounterValue, stats.TransactionsMsgsSpooledToAdb)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_transactions_msgs_retrieved_from_adb_or_disk"], prometheus.CounterValue, stats.TransactionsMsgsRetrievedFromAdbOrDisk)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_transactions_msgs_published"], prometheus.CounterValue, stats.TransactionsMsgsPublished)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_transactions_msgs_consumed"], prometheus.CounterValue, stats.TransactionsMsgsConsumed)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_max_transactions_exceeded"], prometheus.CounterValue, stats.MaxTransactionsExceeded)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_max_transaction_resources_exceeded"], prometheus.CounterValue, stats.MaxTransactionResourcesExceeded)

	// XA session stats
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_xa_open_session"], prometheus.CounterValue, stats.XaOpenSession)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_xa_open_session_success"], prometheus.CounterValue, stats.XaOpenSessionSuccess)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_xa_open_session_max_sessions_exceeded"], prometheus.CounterValue, stats.XaOpenSessionMaxSessionsExceeded)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_xa_open_session_other_failures"], prometheus.CounterValue, stats.XaOpenSessionOtherFailures)

	// XA transaction stats
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_xa_transactions"], prometheus.CounterValue, stats.XaTransactions)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_xa_transactions_success"], prometheus.CounterValue, stats.XaTransactionsSuccess)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_xa_transactions_fail"], prometheus.CounterValue, stats.XaTransactionsFail)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_xa_transactions_msgs_spooled_to_adb"], prometheus.CounterValue, stats.XaTransactionsMsgsSpooledToAdb)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_xa_transactions_msgs_retrieved_from_adb_or_disk"], prometheus.CounterValue, stats.XaTransactionsMsgsRetrievedFromAdbOrDisk)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_xa_transactions_msgs_published"], prometheus.CounterValue, stats.XaTransactionsMsgsPublished)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_xa_transactions_msgs_consumed"], prometheus.CounterValue, stats.XaTransactionsMsgsConsumed)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_xa_max_transactions_exceeded"], prometheus.CounterValue, stats.XaMaxTransactionsExceeded)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_xa_max_transaction_resources_exceeded"], prometheus.CounterValue, stats.XaMaxTransactionResourcesExceeded)

	// Replay stats
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_replays_initiated"], prometheus.CounterValue, stats.ReplaysInitiated)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_replays_succeeded"], prometheus.CounterValue, stats.ReplaysSucceeded)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_replays_failed"], prometheus.CounterValue, stats.ReplaysFailed)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_replayed_messages_sent"], prometheus.CounterValue, stats.ReplayedMessagesSent)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_replayed_messages_acked"], prometheus.CounterValue, stats.ReplayedMessagesAcked)

	// Rate stats (these are gauges, not counters)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_current_bind_rate_per_second"], prometheus.GaugeValue, stats.CurrentBindRatePerSecond)
	ch <- semp.NewMetric(MetricDesc["SpoolStats"]["system_spool_stats_average_bind_rate_per_minute"], prometheus.GaugeValue, stats.AverageBindRatePerMinute)

	return 1, nil
}
