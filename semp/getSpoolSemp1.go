package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"math"
	"strconv"
)

// Get system-wide spool information
func (e *Semp) GetSpoolSemp1(ch chan<- prometheus.Metric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Spool struct {
					Info struct {
						QuotaDiskUsage           float64 `xml:"max-disk-usage"`
						QuotaMsgCount            string  `xml:"max-message-count"`
						PersistUsage             float64 `xml:"current-persist-usage"`
						PersistMsgCount          float64 `xml:"total-messages-currently-spooled"`
						ActiveDiskPartitionUsage string  `xml:"active-disk-partition-usage"` // May be "-"
					} `xml:"message-spool-info"`
				} `xml:"message-spool"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><message-spool></message-spool></show ></rpc>"
	body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", command)
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

	ch <- prometheus.MustNewConstMetric(MetricDesc["Spool"]["system_spool_quota_bytes"], prometheus.GaugeValue, math.Round(target.RPC.Show.Spool.Info.QuotaDiskUsage*1048576.0))
	// MaxMsgCount is in the form "100M"
	s1 := target.RPC.Show.Spool.Info.QuotaMsgCount[:len(target.RPC.Show.Spool.Info.QuotaMsgCount)-1]
	f1, err3 := strconv.ParseFloat(s1, 64)
	if err3 == nil {
		ch <- prometheus.MustNewConstMetric(MetricDesc["Spool"]["system_spool_quota_msgs"], prometheus.GaugeValue, f1*1000000)
	}
	ActiveDiskPartitionUsage, err4 := strconv.ParseFloat(target.RPC.Show.Spool.Info.ActiveDiskPartitionUsage, 64)
	if err4 == nil {
		ch <- prometheus.MustNewConstMetric(MetricDesc["Spool"]["system_spool_disk_partition_usage_active_percent"], prometheus.GaugeValue, math.Round(ActiveDiskPartitionUsage))
	}

	ch <- prometheus.MustNewConstMetric(MetricDesc["Spool"]["system_spool_usage_bytes"], prometheus.GaugeValue, math.Round(target.RPC.Show.Spool.Info.PersistUsage*1048576.0))
	ch <- prometheus.MustNewConstMetric(MetricDesc["Spool"]["system_spool_usage_msgs"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.PersistMsgCount)

	return 1, nil
}
