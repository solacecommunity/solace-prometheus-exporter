package semp

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"strings"
)

// Get version of broker
func (e *Semp) GetVersionSemp1(ch chan<- PrometheusMetric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Version struct {
					Description string `xml:"description"`
					CurrentLoad string `xml:"current-load"`
					Uptime      struct {
						Days      float64 `xml:"days"`
						Hours     float64 `xml:"hours"`
						Mins      float64 `xml:"mins"`
						Secs      float64 `xml:"secs"`
						TotalSecs float64 `xml:"total-secs"`
					} `xml:"uptime"`
				} `xml:"version"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><version/></show></rpc>"
	body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", command, "VersionSemp1", 1)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape getVersionSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml getVersionSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "Unexpected result for getVersionSemp1", "command", command, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
		return 0, errors.New("unexpected result: see log")
	}

	// remember this for the label
	vmrVersion := strings.TrimPrefix(target.RPC.Show.Version.CurrentLoad, "soltr_")
	// compute a version number so it can be measured by Prometheus
	var vmrVersionStrBuffer bytes.Buffer
	for _, s := range strings.Split(vmrVersion, ".") {
		vmrVersionStrBuffer.WriteString(fmt.Sprintf("%03v", s))
	}
	var vmrVersionNr float64
	vmrVersionNr, _ = strconv.ParseFloat(vmrVersionStrBuffer.String(), 64)

	ch <- e.NewMetric(MetricDesc["Version"]["system_version_currentload"], prometheus.GaugeValue, vmrVersionNr)
	ch <- e.NewMetric(MetricDesc["Version"]["system_version_uptime_totalsecs"], prometheus.GaugeValue, target.RPC.Show.Version.Uptime.TotalSecs)
	ch <- e.NewMetric(MetricDesc["Version"]["exporter_version_current"], prometheus.GaugeValue, e.exporterVersion)

	return 1, nil
}
