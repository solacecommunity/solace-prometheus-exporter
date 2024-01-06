package semp

import (
	"github.com/prometheus/client_golang/prometheus"
)

const NoSempV2Ready = "NOT_SEMP_V2_READY"

type Desc struct {
	fqName         string
	sempV2field    string
	help           string
	variableLabels []string
	constLabels    prometheus.Labels
}

func NewSemDesc(fqName string, sempV2field string, help string, variableLabels []string) *Desc {
	return &Desc{
		fqName:         namespace + "_" + fqName,
		sempV2field:    sempV2field,
		help:           help,
		variableLabels: variableLabels,
		constLabels:    nil,
	}
}

func (v2Desc *Desc) AsPrometheusDesc() *prometheus.Desc {
	return prometheus.NewDesc(v2Desc.fqName, v2Desc.help, v2Desc.variableLabels, v2Desc.constLabels)
}
func (v2Desc *Desc) isSelected(selectedFields []string) bool {
	if len(selectedFields) < 1 {
		return true
	}

	return sliceContains(selectedFields, v2Desc.sempV2field)
}

func getSempV2FieldMapList(descriptions Descriptions) map[string]string {
	mapList := make(map[string]string, len(descriptions))

	for _, desc := range descriptions {
		mapList[desc.fqName] = desc.sempV2field
	}
	return mapList
}

func getSempV2FieldsToSelect(metricFilter []string, mandatoryFields []string, descriptions Descriptions) ([]string, error) {
	var fields, err = mapItems(metricFilter, getSempV2FieldMapList(descriptions))
	if err != nil {
		return []string{}, err
	}

	for _, mandatoryField := range mandatoryFields {
		if !sliceContains(fields, mandatoryField) {
			fields = append(fields, mandatoryField)
		}
	}

	return fields, nil
}

type Descriptions map[string]*Desc

type SempV2Result struct {
	v2Desc *Desc
	value  float64
}
