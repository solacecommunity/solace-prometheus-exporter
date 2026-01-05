package semp

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"solace_exporter/internal/semp/types"
	"strconv"
	"strings"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

// GetVersionSemp1 Get version of broker
func (semp *Semp) GetVersionSemp1(ch chan<- PrometheusMetric) (float64, error) {
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
		ExecuteResult types.ExecuteResult `xml:"execute-result"`
	}

	command := "<rpc><show><version/></show></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "VersionSemp1", 1)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't scrape getVersionSemp1", "err", err, "broker", semp.brokerURI)
		return -1, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't decode Xml getVersionSemp1", "err", err, "broker", semp.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(semp.logger).Log("msg", "Unexpected result for getVersionSemp1", "command", command, "result", target.ExecuteResult.Result, "reason", target.ExecuteResult.Reason, "broker", semp.brokerURI)
		return 0, errors.New("unexpected result: " + target.ExecuteResult.Reason + ". see log for further details")
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

	ch <- semp.NewMetric(MetricDesc["Version"]["system_version_currentload"], prometheus.GaugeValue, vmrVersionNr)
	ch <- semp.NewMetric(MetricDesc["Version"]["system_version_uptime_totalsecs"], prometheus.GaugeValue, target.RPC.Show.Version.Uptime.TotalSecs)

	return 1, nil
}
