// Copyright 2018 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	namespace = "solace" // For Prometheus metrics.
)

type metrics map[string]*prometheus.Desc

var (
	globalMutex             = &sync.Mutex{}
	globalResetExecuted     = false
	variableLabelsVpn       = []string{"vpn_name"}
	variableLabelsVpnClient = []string{"vpn_name", "client_name", "client_username"}
	variableLabelsVpnQueue  = []string{"vpn_name", "queue_name"}
	solaceUp                = prometheus.NewDesc(namespace+"_"+"up", "Was the last scrape of solace successful.", nil, nil)
)

var metricsStd = metrics{
	"system_redundancy_up":            prometheus.NewDesc(namespace+"_"+"system_redundancy_up", "Is redundancy up? (0=down, 1=up).", nil, nil),
	"system_redundancy_enabled":       prometheus.NewDesc(namespace+"_"+"system_redundancy_enabled", "Is redundancy enabled in config? (0=disabled, 1=enabled).", nil, nil),
	"system_redundancy_role":          prometheus.NewDesc(namespace+"_"+"system_redundancy_role", "Redundancy role (0=backup, 1=primary, 2=monitor).", nil, nil),
	"system_redundancy_local_active":  prometheus.NewDesc(namespace+"_"+"system_redundancy_local_active", "Is local node the active messaging node? (0=not active, 1=active).", nil, nil),
	"system_spool_quota_mb":           prometheus.NewDesc(namespace+"_"+"system_spool_quota_mb", "Spool configured max disk usage MB.", nil, nil),
	"system_spool_quota_msg":          prometheus.NewDesc(namespace+"_"+"system_spool_quota_msg", "Spool configured max number of messages.", nil, nil),
	"system_spool_usage_mb":           prometheus.NewDesc(namespace+"_"+"system_spool_usage_mb", "Spool total persisted MB usage.", nil, nil),
	"system_spool_msg_count":          prometheus.NewDesc(namespace+"_"+"system_spool_msg_count", "Spool total number of persisted messages.", nil, nil),
	"system_disk_latency_min_us":      prometheus.NewDesc(namespace+"_"+"system_disk_latency_min_us", "Minimum disk latency.", nil, nil),
	"system_disk_latency_max_us":      prometheus.NewDesc(namespace+"_"+"system_disk_latency_max_us", "Maximum disk latency.", nil, nil),
	"system_disk_latency_avg_us":      prometheus.NewDesc(namespace+"_"+"system_disk_latency_avg_us", "Average disk latency.", nil, nil),
	"system_disk_latency_cur_us":      prometheus.NewDesc(namespace+"_"+"system_disk_latency_cur_us", "Current disk latency.", nil, nil),
	"system_compute_latency_min_us":   prometheus.NewDesc(namespace+"_"+"system_compute_latency_min_us", "Minimum compute latency.", nil, nil),
	"system_compute_latency_max_us":   prometheus.NewDesc(namespace+"_"+"system_compute_latency_max_us", "Maximum compute latency.", nil, nil),
	"system_compute_latency_avg_us":   prometheus.NewDesc(namespace+"_"+"system_compute_latency_avg_us", "Average compute latency.", nil, nil),
	"system_compute_latency_cur_us":   prometheus.NewDesc(namespace+"_"+"system_compute_latency_cur_us", "Current compute latency.", nil, nil),
	"system_mate_link_latency_min_us": prometheus.NewDesc(namespace+"_"+"system_mate_link_latency_min_us", "Minimum mate link latency.", nil, nil),
	"system_mate_link_latency_max_us": prometheus.NewDesc(namespace+"_"+"system_mate_link_latency_max_us", "Maximum mate link latency.", nil, nil),
	"system_mate_link_latency_avg_us": prometheus.NewDesc(namespace+"_"+"system_mate_link_latency_avg_us", "Average mate link latency.", nil, nil),
	"system_mate_link_latency_cur_us": prometheus.NewDesc(namespace+"_"+"system_mate_link_latency_cur_us", "Current mate link latency.", nil, nil),

	"vpn_local_status":         prometheus.NewDesc(namespace+"_"+"vpn_local_status", "Local status (0=Down, 1=Up).", variableLabelsVpn, nil),
	"vpn_connection_count":     prometheus.NewDesc(namespace+"_"+"vpn_connection_count", "Number of connections.", variableLabelsVpn, nil),
	"vpn_rx_msg_count":         prometheus.NewDesc(namespace+"_"+"vpn_rx_msg_count", "Number of received messages.", variableLabelsVpn, nil),
	"vpn_tx_msg_count":         prometheus.NewDesc(namespace+"_"+"vpn_tx_msg_count", "Number of transmitted messages.", variableLabelsVpn, nil),
	"vpn_rx_byte_count":        prometheus.NewDesc(namespace+"_"+"vpn_rx_byte_count", "Number of received bytes.", variableLabelsVpn, nil),
	"vpn_tx_byte_count":        prometheus.NewDesc(namespace+"_"+"vpn_tx_byte_count", "Number of transmitted bytes.", variableLabelsVpn, nil),
	"vpn_rx_discard_msg_count": prometheus.NewDesc(namespace+"_"+"vpn_rx_discard_msg_count", "Number of discarded received messages.", variableLabelsVpn, nil),
	"vpn_tx_discard_msg_count": prometheus.NewDesc(namespace+"_"+"vpn_tx_discard_msg_count", "Number of discarded transmitted messages.", variableLabelsVpn, nil),
	"vpn_rx_msg_rate":          prometheus.NewDesc(namespace+"_"+"vpn_rx_msg_rate", "Rate of received messages.", variableLabelsVpn, nil),
	"vpn_tx_msg_rate":          prometheus.NewDesc(namespace+"_"+"vpn_tx_msg_rate", "Rate of transmitted messages.", variableLabelsVpn, nil),
	"vpn_rx_byte_rate":         prometheus.NewDesc(namespace+"_"+"vpn_rx_byte_rate", "Rate of received bytes.", variableLabelsVpn, nil),
	"vpn_tx_byte_rate":         prometheus.NewDesc(namespace+"_"+"vpn_tx_byte_rate", "Rate of transmitted bytes.", variableLabelsVpn, nil),
	"vpn_rx_msg_rate_avg":      prometheus.NewDesc(namespace+"_"+"vpn_rx_msg_rate_avg", "Averate rate of received messages.", variableLabelsVpn, nil),
	"vpn_tx_msg_rate_avg":      prometheus.NewDesc(namespace+"_"+"vpn_tx_msg_rate_avg", "Averate rate of transmitted messages.", variableLabelsVpn, nil),
	"vpn_rx_byte_rate_avg":     prometheus.NewDesc(namespace+"_"+"vpn_rx_byte_rate_avg", "Averate rate of received bytes.", variableLabelsVpn, nil),
	"vpn_tx_byte_rate_avg":     prometheus.NewDesc(namespace+"_"+"vpn_tx_byte_rate_avg", "Averate rate of transmitted bytes.", variableLabelsVpn, nil),
}

var metricsDet = metrics{
	"client_rx_msg_count":         prometheus.NewDesc(namespace+"_"+"client_rx_msg_count", "Number of received messages.", variableLabelsVpnClient, nil),
	"client_tx_msg_count":         prometheus.NewDesc(namespace+"_"+"client_tx_msg_count", "Number of transmitted messages.", variableLabelsVpnClient, nil),
	"client_rx_byte_count":        prometheus.NewDesc(namespace+"_"+"client_rx_byte_count", "Number of received bytes.", variableLabelsVpnClient, nil),
	"client_tx_byte_count":        prometheus.NewDesc(namespace+"_"+"client_tx_byte_count", "Number of transmitted bytes.", variableLabelsVpnClient, nil),
	"client_rx_discard_msg_count": prometheus.NewDesc(namespace+"_"+"client_rx_discard_msg_count", "Number of discarded received messages.", variableLabelsVpnClient, nil),
	"client_tx_discard_msg_count": prometheus.NewDesc(namespace+"_"+"client_tx_discard_msg_count", "Number of discarded transmitted messages.", variableLabelsVpnClient, nil),
	"client_rx_msg_rate":          prometheus.NewDesc(namespace+"_"+"client_rx_msg_rate", "Rate of received messages.", variableLabelsVpnClient, nil),
	"client_tx_msg_rate":          prometheus.NewDesc(namespace+"_"+"client_tx_msg_rate", "Rate of transmitted messages.", variableLabelsVpnClient, nil),
	"client_rx_byte_rate":         prometheus.NewDesc(namespace+"_"+"client_rx_byte_rate", "Rate of received bytes.", variableLabelsVpnClient, nil),
	"client_tx_byte_rate":         prometheus.NewDesc(namespace+"_"+"client_tx_byte_rate", "Rate of transmitted bytes.", variableLabelsVpnClient, nil),
	"client_rx_msg_rate_avg":      prometheus.NewDesc(namespace+"_"+"client_rx_msg_rate_avg", "Averate rate of received messages.", variableLabelsVpnClient, nil),
	"client_tx_msg_rate_avg":      prometheus.NewDesc(namespace+"_"+"client_tx_msg_rate_avg", "Averate rate of transmitted messages.", variableLabelsVpnClient, nil),
	"client_rx_byte_rate_avg":     prometheus.NewDesc(namespace+"_"+"client_rx_byte_rate_avg", "Averate rate of received bytes.", variableLabelsVpnClient, nil),
	"client_tx_byte_rate_avg":     prometheus.NewDesc(namespace+"_"+"client_tx_byte_rate_avg", "Averate rate of transmitted bytes.", variableLabelsVpnClient, nil),
	"client_slow_subscriber":      prometheus.NewDesc(namespace+"_"+"client_slow_subscriber", "Is client a slow subscriber? (0=not slow, 1=slow).", variableLabelsVpnClient, nil),
	"client_uptime_s":             prometheus.NewDesc(namespace+"_"+"client_uptime_s", "Up time of client in seconds.", variableLabelsVpnClient, nil),

	"queue_spool_quota_mb":   prometheus.NewDesc(namespace+"_"+"queue_spool_quota_mb", "Queue spool configured max disk usage MB.", variableLabelsVpnQueue, nil),
	"queue_spool_usage_mb":   prometheus.NewDesc(namespace+"_"+"queue_spool_usage_mb", "Queue spool usage MB.", variableLabelsVpnQueue, nil),
	"queue_spool_msg_count":  prometheus.NewDesc(namespace+"_"+"queue_spool_msg_count", "Queue spooled number of messages.", variableLabelsVpnQueue, nil),
	"queue_bind_count":       prometheus.NewDesc(namespace+"_"+"queue_bind_count", "Number of clients bound to queue.", variableLabelsVpnQueue, nil),
	"queue_rx_msg_rate":      prometheus.NewDesc(namespace+"_"+"queue_rx_msg_rate", "Rate of received messages.", variableLabelsVpnQueue, nil),
	"queue_tx_msg_rate":      prometheus.NewDesc(namespace+"_"+"queue_tx_msg_rate", "Rate of transmitted messages.", variableLabelsVpnQueue, nil),
	"queue_rx_byte_rate":     prometheus.NewDesc(namespace+"_"+"queue_rx_byte_rate", "Rate of received bytes.", variableLabelsVpnQueue, nil),
	"queue_tx_byte_rate":     prometheus.NewDesc(namespace+"_"+"queue_tx_byte_rate", "Rate of transmitted bytes.", variableLabelsVpnQueue, nil),
	"queue_rx_msg_rate_avg":  prometheus.NewDesc(namespace+"_"+"queue_rx_msg_rate_avg", "Averate rate of received messages.", variableLabelsVpnQueue, nil),
	"queue_tx_msg_rate_avg":  prometheus.NewDesc(namespace+"_"+"queue_tx_msg_rate_avg", "Averate rate of transmitted messages.", variableLabelsVpnQueue, nil),
	"queue_rx_byte_rate_avg": prometheus.NewDesc(namespace+"_"+"queue_rx_byte_rate_avg", "Averate rate of received bytes.", variableLabelsVpnQueue, nil),
	"queue_tx_byte_rate_avg": prometheus.NewDesc(namespace+"_"+"queue_tx_byte_rate_avg", "Averate rate of transmitted bytes.", variableLabelsVpnQueue, nil),
}

// Collection of configs
type config struct {
	scrapeURI   string
	username    string
	password    string
	sslVerify   bool
	timeout     time.Duration
	details     bool
	scrapeRates bool
	resetStats  bool
}

// Exporter collects Solace stats from the given URI and exports them using
// the prometheus metrics package.
type Exporter struct {
	config        config
	serverMetrics map[string]*prometheus.Desc
	logger        log.Logger
}

// NewExporter returns an initialized Exporter.
func NewExporter(serverMetrics map[string]*prometheus.Desc, logger log.Logger, conf config) (*Exporter, error) {

	return &Exporter{
		serverMetrics: serverMetrics,
		logger:        logger,
		config:        conf,
	}, nil
}

// Describe describes all the metrics ever exported by the Solace exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, m := range e.serverMetrics {
		ch <- m
	}
	ch <- solaceUp
}

// Collect fetches the stats from configured Solace location and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	var up float64 = 1

	globalMutex.Lock()
	defer globalMutex.Unlock()

	if e.config.resetStats && !globalResetExecuted {
		// Its time to try to reset the stats
		if e.resetStatsSemp1() {
			level.Info(e.logger).Log("msg", "Statistics successfully reset")
			globalResetExecuted = true
			up = 1
		} else {
			up = 0
		}
	}

	if e.config.details {
		if up > 0 {
			up = e.getClientSemp1(ch)
		}
		if up > 0 {
			up = e.getQueueSemp1(ch)
		}
		if up > 0 && e.config.scrapeRates {
			up = e.getQueueRatesSemp1(ch)
		}
	} else { // Basic
		if up > 0 {
			up = e.getRedundancySemp1(ch)
		}
		if up > 0 {
			up = e.getSpoolSemp1(ch)
		}
		if up > 0 {
			up = e.getHealthSemp1(ch)
		}
		if up > 0 {
			up = e.getVpnSemp1(ch)
		}
	}

	ch <- prometheus.MustNewConstMetric(solaceUp, prometheus.GaugeValue, up)
}

// Encodes string to 0,1,2,... metric
func encodeMetricMulti(item string, refItems []string) float64 {
	uItem := strings.ToUpper(item)
	for i, s := range refItems {
		if uItem == strings.ToUpper(s) {
			return float64(i)
		}
	}
	return -1
}

// Encodes bool into 0,1 metric
func encodeMetricBool(item bool) float64 {
	if item {
		return 1
	}
	return 0
}

// Redirect callback, re-insert basic auth string into header
func (e *Exporter) redirectPolicyFunc(req *http.Request, via []*http.Request) error {
	req.SetBasicAuth(e.config.username, e.config.password)
	return nil
}

// Call http post for the supplied uri and body
func (e *Exporter) postHTTP(uri string, contentType string, body string) (io.ReadCloser, error) {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: !e.config.sslVerify}}
	client := http.Client{
		Timeout:       e.config.timeout,
		Transport:     tr,
		CheckRedirect: e.redirectPolicyFunc,
	}

	req, err := http.NewRequest("GET", uri, strings.NewReader(body))
	req.SetBasicAuth(e.config.username, e.config.password)
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

// Reset a stats item via SEMP1
func (e *Exporter) resetStatsItemSemp1(bodyString string) (ok bool) {

	type Data struct {
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", bodyString)
	if err != nil {
		level.Error(e.logger).Log("command", bodyString, "err", err)
		return false
	}

	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		level.Error(e.logger).Log("command", bodyString, "err", err)
		return false
	}

	if target.ExecuteResult.Result != "ok" {
		level.Error(e.logger).Log("command", bodyString)
		return false
	}

	return true
}

// Reset some stats items via SEMP1
func (e *Exporter) resetStatsSemp1() (ok bool) {

	if e.resetStatsItemSemp1("<rpc><clear><system><health><stats/></health></system></clear></rpc>") == false {
		return false
	}
	if e.resetStatsItemSemp1("<rpc><clear><message-spool><stats/></message-spool></clear></rpc>") == false {
		return false
	}
	if e.resetStatsItemSemp1("<rpc><clear><message-vpn><vpn-name>*</vpn-name><stats/></message-vpn></clear></rpc>") == false {
		return false
	}
	if e.resetStatsItemSemp1("<rpc><clear><stats><client/></stats></clear></rpc>") == false {
		return false
	}
	if e.resetStatsItemSemp1("<rpc><clear><client><name>*</name><stats/></client></clear></rpc>") == false {
		return false
	}
	if e.resetStatsItemSemp1("<rpc><clear><queue><name>*</name><stats/></queue></clear></rpc>") == false {
		return false
	}

	return true
}

// Get system-wide basic redundancy information for HA triples
func (e *Exporter) getRedundancySemp1(ch chan<- prometheus.Metric) (ok float64) {
	var f float64

	type Data struct {
		RPC struct {
			Show struct {
				Red struct {
					ConfigStatus      string `xml:"config-status"`
					RedundancyStatus  string `xml:"redundancy-status"`
					OperatingMode     string `xml:"operating-mode"`
					ActiveStandbyRole string `xml:"active-standby-role"`
					VirtualRouters    struct {
						Primary struct {
							Status struct {
								Activity string `xml:"activity"`
							} `xml:"status"`
						} `xml:"primary"`
						Backup struct {
							Status struct {
								Activity string `xml:"activity"`
							} `xml:"status"`
						} `xml:"backup"`
					} `xml:"virtual-routers"`
				} `xml:"redundancy"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><redundancy/></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't scrape RedundancySemp1", "err", err)
		return 0
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't decode Xml RedundancySemp1", "err", err)
		return 0
	}
	if target.ExecuteResult.Result != "ok" {
		level.Error(e.logger).Log("command", command)
		return 0
	}

	ch <- prometheus.MustNewConstMetric(metricsStd["system_redundancy_enabled"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Red.ConfigStatus, []string{"Disabled", "Enabled"}))
	ch <- prometheus.MustNewConstMetric(metricsStd["system_redundancy_up"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Red.RedundancyStatus, []string{"Down", "Up"}))
	ch <- prometheus.MustNewConstMetric(metricsStd["system_redundancy_role"], prometheus.GaugeValue, encodeMetricMulti(target.RPC.Show.Red.ActiveStandbyRole, []string{"Backup", "Primary", "Undefined"}))

	if target.RPC.Show.Red.ActiveStandbyRole == "Primary" && target.RPC.Show.Red.VirtualRouters.Primary.Status.Activity == "Local Active" ||
		target.RPC.Show.Red.ActiveStandbyRole == "Backup" && target.RPC.Show.Red.VirtualRouters.Backup.Status.Activity == "Local Active" {
		f = 1
	} else {
		f = 0
	}
	ch <- prometheus.MustNewConstMetric(metricsStd["system_redundancy_local_active"], prometheus.GaugeValue, f)

	return 1
}

// Get system-wide spool information
func (e *Exporter) getSpoolSemp1(ch chan<- prometheus.Metric) (ok float64) {

	type Data struct {
		RPC struct {
			Show struct {
				Spool struct {
					Info struct {
						QuotaDiskUsage  float64 `xml:"max-disk-usage"`
						QuotaMsgCount   string  `xml:"max-message-count"`
						PersistUsage    float64 `xml:"current-persist-usage"`
						PersistMsgCount float64 `xml:"total-messages-currently-spooled"`
					} `xml:"message-spool-info"`
				} `xml:"message-spool"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><message-spool></message-spool></show ></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't scrape Solace", "err", err)
		return 0
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't decode Xml", "err", err)
		return 0
	}
	if target.ExecuteResult.Result != "ok" {
		level.Error(e.logger).Log("command", command)
		return 0
	}

	ch <- prometheus.MustNewConstMetric(metricsStd["system_spool_quota_mb"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.QuotaDiskUsage)
	// MaxMsgCount is in the form "100M"
	s1 := target.RPC.Show.Spool.Info.QuotaMsgCount[:len(target.RPC.Show.Spool.Info.QuotaMsgCount)-1]
	f1, err3 := strconv.ParseFloat(s1, 64)
	if err3 == nil {
		ch <- prometheus.MustNewConstMetric(metricsStd["system_spool_quota_msg"], prometheus.GaugeValue, f1*1000000)
	}
	ch <- prometheus.MustNewConstMetric(metricsStd["system_spool_usage_mb"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.PersistUsage)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_spool_msg_count"], prometheus.GaugeValue, target.RPC.Show.Spool.Info.PersistMsgCount)

	return 1
}

// Get system health information
func (e *Exporter) getHealthSemp1(ch chan<- prometheus.Metric) (ok float64) {

	type Data struct {
		RPC struct {
			Show struct {
				System struct {
					Health struct {
						DiskLatencyMinimumValue     float64 `xml:"disk-latency-minimum-value"`
						DiskLatencyMaximumValue     float64 `xml:"disk-latency-maximum-value"`
						DiskLatencyAverageValue     float64 `xml:"disk-latency-average-value"`
						DiskLatencyCurrentValue     float64 `xml:"disk-latency-current-value"`
						ComputeLatencyMinimumValue  float64 `xml:"compute-latency-minimum-value"`
						ComputeLatencyMaximumValue  float64 `xml:"compute-latency-maximum-value"`
						ComputeLatencyAverageValue  float64 `xml:"compute-latency-average-value"`
						ComputeLatencyCurrentValue  float64 `xml:"compute-latency-current-value"`
						MateLinkLatencyMinimumValue float64 `xml:"mate-link-latency-minimum-value"`
						MateLinkLatencyMaximumValue float64 `xml:"mate-link-latency-maximum-value"`
						MateLinkLatencyAverageValue float64 `xml:"mate-link-latency-average-value"`
						MateLinkLatencyCurrentValue float64 `xml:"mate-link-latency-current-value"`
					} `xml:"health"`
				} `xml:"system"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><system><health/></system></show ></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't scrape HealthSemp1", "err", err)
		return 0
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't decode Xml HealthSemp1", "err", err)
		return 0
	}
	if target.ExecuteResult.Result != "ok" {
		level.Error(e.logger).Log("command", command)
		return 0
	}

	ch <- prometheus.MustNewConstMetric(metricsStd["system_disk_latency_min_us"], prometheus.GaugeValue, target.RPC.Show.System.Health.DiskLatencyMinimumValue)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_disk_latency_max_us"], prometheus.GaugeValue, target.RPC.Show.System.Health.DiskLatencyMaximumValue)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_disk_latency_avg_us"], prometheus.GaugeValue, target.RPC.Show.System.Health.DiskLatencyAverageValue)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_disk_latency_cur_us"], prometheus.GaugeValue, target.RPC.Show.System.Health.DiskLatencyCurrentValue)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_compute_latency_min_us"], prometheus.GaugeValue, target.RPC.Show.System.Health.ComputeLatencyMinimumValue)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_compute_latency_max_us"], prometheus.GaugeValue, target.RPC.Show.System.Health.ComputeLatencyMaximumValue)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_compute_latency_avg_us"], prometheus.GaugeValue, target.RPC.Show.System.Health.ComputeLatencyAverageValue)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_compute_latency_cur_us"], prometheus.GaugeValue, target.RPC.Show.System.Health.ComputeLatencyCurrentValue)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_mate_link_latency_min_us"], prometheus.GaugeValue, target.RPC.Show.System.Health.MateLinkLatencyMinimumValue)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_mate_link_latency_max_us"], prometheus.GaugeValue, target.RPC.Show.System.Health.MateLinkLatencyMaximumValue)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_mate_link_latency_avg_us"], prometheus.GaugeValue, target.RPC.Show.System.Health.MateLinkLatencyAverageValue)
	ch <- prometheus.MustNewConstMetric(metricsStd["system_mate_link_latency_cur_us"], prometheus.GaugeValue, target.RPC.Show.System.Health.MateLinkLatencyCurrentValue)

	return 1
}

// Get status and number of connected clients of all vpn's, and some data stats and rates
func (e *Exporter) getVpnSemp1(ch chan<- prometheus.Metric) (ok float64) {

	type Data struct {
		RPC struct {
			Show struct {
				MessageVpn struct {
					Vpn []struct {
						Name        string  `xml:"name"`
						LocalStatus string  `xml:"local-status"`
						Connections float64 `xml:"connections"`
						Stats       struct {
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
					} `xml:"vpn"`
				} `xml:"message-vpn"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><message-vpn><vpn-name>*</vpn-name><stats/></message-vpn></show></rpc>"
	body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", command)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't scrape VpnSemp1", "err", err)
		return 0
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		level.Error(e.logger).Log("msg", "Can't decode Xml VpnSemp1", "err", err)
		return 0
	}
	if target.ExecuteResult.Result != "ok" {
		level.Error(e.logger).Log("command", command)
		return 0
	}

	for _, vpn := range target.RPC.Show.MessageVpn.Vpn {
		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_connection_count"], prometheus.GaugeValue, vpn.Connections, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_local_status"], prometheus.GaugeValue, encodeMetricMulti(vpn.LocalStatus, []string{"Down", "Up"}), vpn.Name)

		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_rx_msg_count"], prometheus.GaugeValue, vpn.Stats.DataRxMsgCount, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_tx_msg_count"], prometheus.GaugeValue, vpn.Stats.DataTxMsgCount, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_rx_byte_count"], prometheus.GaugeValue, vpn.Stats.DataRxByteCount, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_tx_byte_count"], prometheus.GaugeValue, vpn.Stats.DataTxByteCount, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_rx_discard_msg_count"], prometheus.GaugeValue, vpn.Stats.IngressDiscards.DiscardedRxMsgCount, vpn.Name)
		ch <- prometheus.MustNewConstMetric(metricsStd["vpn_tx_discard_msg_count"], prometheus.GaugeValue, vpn.Stats.EgressDiscards.DiscardedTxMsgCount, vpn.Name)

		if e.config.scrapeRates {
			ch <- prometheus.MustNewConstMetric(metricsStd["vpn_rx_msg_rate"], prometheus.GaugeValue, vpn.Stats.RxMsgRate, vpn.Name)
			ch <- prometheus.MustNewConstMetric(metricsStd["vpn_tx_msg_rate"], prometheus.GaugeValue, vpn.Stats.TxMsgRate, vpn.Name)
			ch <- prometheus.MustNewConstMetric(metricsStd["vpn_rx_byte_rate"], prometheus.GaugeValue, vpn.Stats.RxByteRate, vpn.Name)
			ch <- prometheus.MustNewConstMetric(metricsStd["vpn_tx_byte_rate"], prometheus.GaugeValue, vpn.Stats.TxByteRate, vpn.Name)
			ch <- prometheus.MustNewConstMetric(metricsStd["vpn_rx_msg_rate_avg"], prometheus.GaugeValue, vpn.Stats.AverageRxMsgRate, vpn.Name)
			ch <- prometheus.MustNewConstMetric(metricsStd["vpn_tx_msg_rate_avg"], prometheus.GaugeValue, vpn.Stats.AverageTxMsgRate, vpn.Name)
			ch <- prometheus.MustNewConstMetric(metricsStd["vpn_rx_byte_rate_avg"], prometheus.GaugeValue, vpn.Stats.AverageRxByteRate, vpn.Name)
			ch <- prometheus.MustNewConstMetric(metricsStd["vpn_tx_byte_rate_avg"], prometheus.GaugeValue, vpn.Stats.AverageTxByteRate, vpn.Name)
		}
	}

	return 1
}

// Get some statistics for each indivitual client of all vpn's
// This can result in heavy system load for lots of clients
func (e *Exporter) getClientSemp1(ch chan<- prometheus.Metric) (ok float64) {

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
					} `xml:"primary-virtual-router"`
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

	for nextRequest := "<rpc><show><client><name>*</name><stats/><count/><num-elements>100</num-elements></client></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			level.Error(e.logger).Log("msg", "Can't scrape ClientSemp1", "err", err)
			return 0
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			level.Error(e.logger).Log("msg", "Can't decode ClientSemp1", "err", err)
			return 0
		}
		if target.ExecuteResult.Result != "ok" {
			level.Error(e.logger).Log("command", "Show client stats")
			return 0
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, client := range target.RPC.Show.Client.PrimaryVirtualRouter.Client {
			ch <- prometheus.MustNewConstMetric(metricsDet["client_rx_msg_count"], prometheus.GaugeValue, client.Stats.DataRxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(metricsDet["client_tx_msg_count"], prometheus.GaugeValue, client.Stats.DataTxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(metricsDet["client_rx_byte_count"], prometheus.GaugeValue, client.Stats.DataRxByteCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(metricsDet["client_tx_byte_count"], prometheus.GaugeValue, client.Stats.DataTxByteCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(metricsDet["client_rx_discard_msg_count"], prometheus.GaugeValue, client.Stats.IngressDiscards.DiscardedRxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)
			ch <- prometheus.MustNewConstMetric(metricsDet["client_tx_discard_msg_count"], prometheus.GaugeValue, client.Stats.EgressDiscards.DiscardedTxMsgCount, client.MsgVpnName, client.ClientName, client.ClientUsername)

			if e.config.scrapeRates {
				ch <- prometheus.MustNewConstMetric(metricsDet["client_rx_msg_rate"], prometheus.GaugeValue, client.Stats.RxMsgRate, client.MsgVpnName, client.ClientName, client.ClientUsername)
				ch <- prometheus.MustNewConstMetric(metricsDet["client_tx_msg_rate"], prometheus.GaugeValue, client.Stats.TxMsgRate, client.MsgVpnName, client.ClientName, client.ClientUsername)
				ch <- prometheus.MustNewConstMetric(metricsDet["client_rx_byte_rate"], prometheus.GaugeValue, client.Stats.RxByteRate, client.MsgVpnName, client.ClientName, client.ClientUsername)
				ch <- prometheus.MustNewConstMetric(metricsDet["client_tx_byte_rate"], prometheus.GaugeValue, client.Stats.TxByteRate, client.MsgVpnName, client.ClientName, client.ClientUsername)
				ch <- prometheus.MustNewConstMetric(metricsDet["client_rx_msg_rate_avg"], prometheus.GaugeValue, client.Stats.AverageRxMsgRate, client.MsgVpnName, client.ClientName, client.ClientUsername)
				ch <- prometheus.MustNewConstMetric(metricsDet["client_tx_msg_rate_avg"], prometheus.GaugeValue, client.Stats.AverageTxMsgRate, client.MsgVpnName, client.ClientName, client.ClientUsername)
				ch <- prometheus.MustNewConstMetric(metricsDet["client_rx_byte_rate_avg"], prometheus.GaugeValue, client.Stats.AverageRxByteRate, client.MsgVpnName, client.ClientName, client.ClientUsername)
				ch <- prometheus.MustNewConstMetric(metricsDet["client_tx_byte_rate_avg"], prometheus.GaugeValue, client.Stats.AverageTxByteRate, client.MsgVpnName, client.ClientName, client.ClientUsername)
			}

			ch <- prometheus.MustNewConstMetric(metricsDet["client_slow_subscriber"], prometheus.GaugeValue, encodeMetricBool(client.SlowSubscriber), client.MsgVpnName, client.ClientName, client.ClientUsername)
			//ch <- prometheus.MustNewConstMetric(metricsDet["client_uptime_s"], prometheus.GaugeValue, 0, client.MsgVpnName, client.ClientName, client.ClientUsername)
		}
		body.Close()
	}

	return 1
}

// Get some statistics for each indivitual queue of all vpn's
// This can result in heavy system load for lots of queues
func (e *Exporter) getQueueSemp1(ch chan<- prometheus.Metric) (ok float64) {

	type Data struct {
		RPC struct {
			Show struct {
				Queue struct {
					Queues struct {
						Queue []struct {
							QueueName string `xml:"name"`
							Info      struct {
								MsgVpnName      string  `xml:"message-vpn"`
								Quota           float64 `xml:"quota"`
								Usage           float64 `xml:"current-spool-usage-in-mb"`
								SpooledMsgCount float64 `xml:"num-messages-spooled"`
								BindCount       float64 `xml:"bind-count"`
							} `xml:"info"`
						} `xml:"queue"`
					} `xml:"queues"`
				} `xml:"queue"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	for nextRequest := "<rpc><show><queue><name>*</name><detail/><count/><num-elements>100</num-elements></queue></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			level.Error(e.logger).Log("msg", "Can't scrape QueueSemp1", "err", err)
			return 0
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			level.Error(e.logger).Log("msg", "Can't decode QueueSemp1", "err", err)
			return 0
		}
		if target.ExecuteResult.Result != "ok" {
			level.Error(e.logger).Log("command", "Show queue details")
			return 0
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, queue := range target.RPC.Show.Queue.Queues.Queue {
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_spool_quota_mb"], prometheus.GaugeValue, queue.Info.Quota, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_spool_usage_mb"], prometheus.GaugeValue, queue.Info.Usage, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_spool_msg_count"], prometheus.GaugeValue, queue.Info.SpooledMsgCount, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_bind_count"], prometheus.GaugeValue, queue.Info.BindCount, queue.Info.MsgVpnName, queue.QueueName)
		}
		body.Close()
	}

	return 1
}

// Get rates for each indivitual queue of all vpn's
// This can result in heavy system load for lots of queues
func (e *Exporter) getQueueRatesSemp1(ch chan<- prometheus.Metric) (ok float64) {

	type Data struct {
		RPC struct {
			Show struct {
				Queue struct {
					Queues struct {
						Queue []struct {
							QueueName string `xml:"name"`
							Info      struct {
								MsgVpnName string `xml:"message-vpn"`
							} `xml:"info"`
							Rates struct {
								Qendpt struct {
									AverageRxByteRate float64 `xml:"average-ingress-byte-rate-per-minute"`
									AverageRxMsgRate  float64 `xml:"average-ingress-rate-per-minute"`
									AverageTxByteRate float64 `xml:"average-egress-byte-rate-per-minute"`
									AverageTxMsgRate  float64 `xml:"average-egress-rate-per-minute"`
									RxByteRate        float64 `xml:"current-ingress-byte-rate-per-second"`
									RxMsgRate         float64 `xml:"current-ingress-rate-per-second"`
									TxByteRate        float64 `xml:"current-egress-byte-rate-per-second"`
									TxMsgRate         float64 `xml:"current-egress-rate-per-second"`
								} `xml:"qendpt-data-rates"`
							} `xml:"rates"`
						} `xml:"queue"`
					} `xml:"queues"`
				} `xml:"queue"`
			} `xml:"show"`
		} `xml:"rpc"`
		MoreCookie struct {
			RPC string `xml:",innerxml"`
		} `xml:"more-cookie"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	for nextRequest := "<rpc><show><queue><name>*</name><rates/><count/><num-elements>100</num-elements></queue></show></rpc>"; nextRequest != ""; {
		body, err := e.postHTTP(e.config.scrapeURI+"/SEMP", "application/xml", nextRequest)
		if err != nil {
			level.Error(e.logger).Log("msg", "Can't scrape QueueRatesSemp1", "err", err)
			return 0
		}
		defer body.Close()
		decoder := xml.NewDecoder(body)
		var target Data
		err = decoder.Decode(&target)
		if err != nil {
			level.Error(e.logger).Log("msg", "Can't decode QueueRatesSemp1", "err", err)
			return 0
		}
		if target.ExecuteResult.Result != "ok" {
			level.Error(e.logger).Log("command", "Show queue rates")
			return 0
		}

		//fmt.Printf("Next request: %v\n", target.MoreCookie.RPC)
		nextRequest = target.MoreCookie.RPC

		for _, queue := range target.RPC.Show.Queue.Queues.Queue {
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_rx_msg_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.RxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_tx_msg_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.TxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_rx_byte_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.RxByteRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_tx_byte_rate"], prometheus.GaugeValue, queue.Rates.Qendpt.TxByteRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_rx_msg_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageRxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_tx_msg_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageTxMsgRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_rx_byte_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageRxByteRate, queue.Info.MsgVpnName, queue.QueueName)
			ch <- prometheus.MustNewConstMetric(metricsDet["queue_tx_byte_rate_avg"], prometheus.GaugeValue, queue.Rates.Qendpt.AverageTxByteRate, queue.Info.MsgVpnName, queue.QueueName)
		}
		body.Close()
	}

	return 1
}

func main() {
	listenAddress := kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9101").Envar("SOLACE_WEB_LISTEN_ADDRESS").String()

	var conf config
	kingpin.Flag("sol.uri", "Base URI on which to scrape Solace.").Default("http://localhost:8080").Envar("SOLACE_SCRAPE_URI").StringVar(&conf.scrapeURI)
	kingpin.Flag("sol.user", "Username for http requests to Solace broker.").Default("admin").Envar("SOLACE_USER").StringVar(&conf.username)
	kingpin.Flag("sol.pass", "Password for http requests to Solace broker.").Default("admin").Envar("SOLACE_PASSWORD").StringVar(&conf.password)
	kingpin.Flag("sol.timeout", "Timeout for trying to get stats from Solace.").Default("5s").Envar("SOLACE_SCRAPE_TIMEOUT").DurationVar(&conf.timeout)
	kingpin.Flag("sol.sslv", "Flag that enables SSL certificate verification for the scrape URI").Default("False").Envar("SOLACE_SSL_VERIFY").BoolVar(&conf.sslVerify)
	kingpin.Flag("sol.reset", "Flag that enables resetting system/vpn/client/queue stats in Solace").Default("False").Envar("SOLACE_RESET_STATS").BoolVar(&conf.resetStats)
	kingpin.Flag("sol.rates", "Flag that enables scrape of rate metrics").Default("False").Envar("SOLACE_INCLUDE_RATES").BoolVar(&conf.scrapeRates)

	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger := promlog.New(promlogConfig)

	level.Info(logger).Log("msg", "Starting solace_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "context", version.BuildContext())

	conf.details = false
	exporterStd, err := NewExporter(metricsStd, logger, conf)
	if err != nil {
		level.Error(logger).Log("msg", "Error creating an exporter", "err", err)
		os.Exit(1)
	}
	registryStd := prometheus.NewRegistry()
	registryStd.MustRegister(exporterStd)
	registryStd.MustRegister(version.NewCollector("solace_standard"))
	handlerStd := promhttp.HandlerFor(registryStd, promhttp.HandlerOpts{})

	conf.details = true
	exporterDet, err := NewExporter(metricsDet, logger, conf)
	if err != nil {
		level.Error(logger).Log("msg", "Error creating an exporter", "err", err)
		os.Exit(1)
	}
	registryDet := prometheus.NewRegistry()
	registryDet.MustRegister(exporterDet)
	registryDet.MustRegister(version.NewCollector("solace_detailed"))
	handlerDet := promhttp.HandlerFor(registryDet, promhttp.HandlerOpts{})

	level.Info(logger).Log("msg", "Listening on address", "address", *listenAddress)
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/solace-std", handlerStd)
	http.Handle("/solace-det", handlerDet)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Solace Exporter</title></head>
             <body>
             <h1>Solace Exporter</h1>
             <p><a href='` + "/metrics" + `'>Default Metrics</a></p>
             <p><a href='` + "/solace-std" + `'>Solace Standard Metrics (System and VPN)</a></p>
             <p><a href='` + "/solace-det" + `'>Solace Detailed Metrics (Client and Queue)</a></p>
             </body>
             </html>`))
	})
	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
		os.Exit(1)
	}
}
