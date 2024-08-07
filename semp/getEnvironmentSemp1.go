package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"math"
	"strconv"
	"strings"
)

// Get system Alarm information
func (e *Semp) GetEnvironmentSemp1(ch chan<- PrometheusMetric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Environment struct {
					Mainboard struct {
						Sensors struct {
							Sensor []struct {
								Type   string `xml:"type"`
								Name   string `xml:"name"`
								Value  string `xml:"value"`
								Unit   string `xml:"unit"`
								Status string `xml:"status"`
							} `xml:"sensor"`
						} `xml:"sensors"`
					} `xml:"mainboard"`
					Slots struct {
						Slot []struct {
							SlotNumber string `xml:"slot-number"`
							CardType   string `xml:"card-type"`
							Sensors    struct {
								Sensor []struct {
									Type   string `xml:"type"`
									Name   string `xml:"name"`
									Value  string `xml:"value"`
									Unit   string `xml:"unit"`
									Status string `xml:"status"`
								} `xml:"sensor"`
							} `xml:"sensors"`
						} `xml:"slot"`
					} `xml:"slots"`
				} `xml:"environment"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><environment/></show></rpc>"
	body, err := e.postHTTP(e.brokerURI+"/SEMP", "application/xml", command, "EnvironmentSemp1", 1)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't scrape EnvironmentSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(e.logger).Log("msg", "Can't decode Xml EnvironmentSemp1", "err", err, "broker", e.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(e.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", e.brokerURI)
		return 0, errors.New("unexpected result: see log")
	}

	for _, sensor := range target.RPC.Show.Environment.Mainboard.Sensors.Sensor {
		if sensor.Type == "Fan speed" && strings.Contains(sensor.Name, "Chassis") {
			if value, err := strconv.ParseFloat(sensor.Value, 64); err == nil {
				ch <- e.NewMetric(MetricDesc["Environment"]["system_chassis_fan_speed"], prometheus.GaugeValue, math.Round(value), sensor.Name)
			}
		} else if sensor.Type == "Temperature" && strings.Contains(sensor.Name, "Therm Margin") {
			if value, err := strconv.ParseFloat(sensor.Value, 64); err == nil {
				ch <- e.NewMetric(MetricDesc["Environment"]["system_cpu_thermal_margin"], prometheus.GaugeValue, math.Round(value), sensor.Name)
			}
		}
	}
	for _, slot := range target.RPC.Show.Environment.Slots.Slot {
		if slot.CardType == "Network Acceleration Blade" {
			for _, sensor := range slot.Sensors.Sensor {
				if sensor.Type == "Temperature" && sensor.Name == "NPU Core Temp" {
					if value, err := strconv.ParseFloat(sensor.Value, 64); err == nil {
						ch <- e.NewMetric(MetricDesc["Environment"]["system_nab_core_temperature"], prometheus.GaugeValue, math.Round(value), sensor.Name)
					}
				}
			}
		}
	}

	return 1, nil
}
