package semp

import (
	"encoding/xml"
	"errors"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
)

// GetHardwareSemp1 Get system Alarm information
func (semp *Semp) GetHardwareSemp1(ch chan<- PrometheusMetric) (ok float64, err error) {
	type Data struct {
		RPC struct {
			Show struct {
				Hardware struct {
					PowerRedundancy struct {
						OperationalPowerSupplies float64 `xml:"operational-power-supplies"`
					} `xml:"power-redundancy"`
					Fabric struct {
						Slot []struct {
							SlotNumber       string `xml:"slot-number" optional:"yes"`
							CardType         string `xml:"card-type" optional:"yes"`
							OperationalState bool   `xml:"operational-state-up" optional:"yes"`
							FlashCardState   string `xml:"flash-card-state" optional:"yes"`
							PowerModuleState string `xml:"power-module-state" optional:"yes"`
							MateLink1State   string `xml:"mate-link-1-state" optional:"yes"`
							MateLink2State   string `xml:"mate-link-2-state" optional:"yes"`
							FibreChannel     []struct {
								Number           string `xml:"number"`
								OperationalState string `xml:"operational-state"`
								State            string `xml:"state"`
							} `xml:"fibre-channel" optional:"yes"`
							ExternalDiskLun []struct {
								Number string `xml:"number"`
								State  string `xml:"state"`
							} `xml:"external-disk-lun" optional:"yes"`
						} `xml:"slot"`
					} `xml:"fabric"`
				} `xml:"hardware"`
			} `xml:"show"`
		} `xml:"rpc"`
		ExecuteResult struct {
			Result string `xml:"code,attr"`
		} `xml:"execute-result"`
	}

	command := "<rpc><show><hardware><details/></hardware></show></rpc>"
	body, err := semp.postHTTP(semp.brokerURI+"/SEMP", "application/xml", command, "HardwareSemp1", 1)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't scrape HardwareSemp1", "err", err, "broker", semp.brokerURI)
		return 0, err
	}
	defer body.Close()
	decoder := xml.NewDecoder(body)
	var target Data
	err = decoder.Decode(&target)
	if err != nil {
		_ = level.Error(semp.logger).Log("msg", "Can't decode Xml HardwareSemp1", "err", err, "broker", semp.brokerURI)
		return 0, err
	}
	if target.ExecuteResult.Result != "ok" {
		_ = level.Error(semp.logger).Log("msg", "unexpected result", "command", command, "result", target.ExecuteResult.Result, "broker", semp.brokerURI)
		return 0, errors.New("unexpected result: see log")
	}

	ch <- semp.NewMetric(MetricDesc["Hardware"]["operational_power_supplies"], prometheus.GaugeValue, target.RPC.Show.Hardware.PowerRedundancy.OperationalPowerSupplies)

	for _, slot := range target.RPC.Show.Hardware.Fabric.Slot {
		if slot.CardType == "Host Bus Adapter Blade" {
			for _, FC := range slot.FibreChannel {
				ch <- semp.NewMetric(MetricDesc["Hardware"]["fibre_channel_operational_state"], prometheus.GaugeValue, encodeMetricMulti(FC.OperationalState, []string{"Linkdown", "Online"}), FC.Number)
				ch <- semp.NewMetric(MetricDesc["Hardware"]["fibre_channel_state"], prometheus.GaugeValue, encodeMetricMulti(FC.State, []string{"Link Down", "Link Up - F_Port (fabric via point-to-point)", "Link Up - Loop (private loop)"}), FC.Number)
			}
			for _, LUN := range slot.ExternalDiskLun {
				State := "Ready"
				if !strings.Contains(LUN.State, "Ready") {
					State = "Offline"
				}
				ch <- semp.NewMetric(MetricDesc["Hardware"]["external_disk_lun_state"], prometheus.GaugeValue, encodeMetricMulti(State, []string{"Offline", "Ready"}), LUN.Number)
			}
		} else if slot.CardType == "Assured Delivery Blade" {
			ch <- semp.NewMetric(MetricDesc["Hardware"]["adb_operational_state"], prometheus.GaugeValue, encodeMetricBool(slot.OperationalState))
			ch <- semp.NewMetric(MetricDesc["Hardware"]["adb_flash_card_state"], prometheus.GaugeValue, encodeMetricMulti(slot.FlashCardState, []string{"Link Down", "Ready"}))
			ch <- semp.NewMetric(MetricDesc["Hardware"]["adb_power_module_state"], prometheus.GaugeValue, encodeMetricMulti(slot.PowerModuleState, []string{"", "Ok"}))
			ch <- semp.NewMetric(MetricDesc["Hardware"]["adb_mate_link_port1_state"], prometheus.GaugeValue, encodeMetricMulti(slot.MateLink1State, []string{"LOS", "Ok", "No SFP Module", "No Data"}))
			ch <- semp.NewMetric(MetricDesc["Hardware"]["adb_mate_link_port2_state"], prometheus.GaugeValue, encodeMetricMulti(slot.MateLink2State, []string{"LOS", "Ok", "No SFP Module", "No Data"}))
		}
	}

	return 1, nil
}
