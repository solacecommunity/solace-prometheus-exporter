package semp

import (
	"bytes"
	"fmt"
	"solace_exporter/internal/semp/types"
	"strconv"
	"strings"

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
		types.Reply
	}

	command := "<rpc><show><version/></show></rpc>"
	var target Data
	if up, err := semp.fetchSemp1(command, "VersionSemp1", 1, &target); err != nil {
		return up, withLegacyResultErrorText(err)
	}

	// remember this for the label
	vmrVersion := strings.TrimPrefix(target.RPC.Show.Version.CurrentLoad, "soltr_")
	// compute a version number so it can be measured by Prometheus
	var vmrVersionStrBuffer bytes.Buffer
	for _, s := range strings.Split(vmrVersion, ".") {
		fmt.Fprintf(&vmrVersionStrBuffer, "%03v", s)
	}
	var vmrVersionNr float64
	vmrVersionNr, _ = strconv.ParseFloat(vmrVersionStrBuffer.String(), 64)

	ch <- semp.NewMetric(MetricDesc["Version"]["system_version_currentload"], prometheus.GaugeValue, vmrVersionNr)
	ch <- semp.NewMetric(MetricDesc["Version"]["system_version_uptime_totalsecs"], prometheus.GaugeValue, target.RPC.Show.Version.Uptime.TotalSecs)

	return 1, nil
}
