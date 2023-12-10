package semp

import (
	"github.com/prometheus/client_golang/prometheus"
)

type SempV2Desc struct {
	fqName         string
	sempV2field    string
	help           string
	variableLabels []string
	constLabels    prometheus.Labels
}

func NewSempV2Desc(fqName string, sempV2field string, help string, variableLabels []string) *SempV2Desc {
	return &SempV2Desc{
		fqName:         fqName,
		sempV2field:    sempV2field,
		help:           help,
		variableLabels: variableLabels,
		constLabels:    nil,
	}
}

func (v2Desc *SempV2Desc) NewPrometheusDesc() *prometheus.Desc {
	return prometheus.NewDesc(v2Desc.fqName, v2Desc.help, v2Desc.variableLabels, v2Desc.constLabels)
}
func (v2Desc *SempV2Desc) isSelected(selectedFields []string) bool {
	if len(selectedFields) < 1 {
		return true
	}

	return sliceContains(selectedFields, v2Desc.sempV2field)
}

func getSempV2FieldMapList(descs SempV2Descs) map[string]string {
	mapList := make(map[string]string, len(descs))

	for _, desc := range descs {
		mapList[desc.fqName] = desc.sempV2field
	}
	return mapList
}

func getSempV2FieldsToSelect(metricFilter []string, mandatoryFields []string, descs SempV2Descs) ([]string, error) {
	var fields, err = mapItems(metricFilter, getSempV2FieldMapList(descs))
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

type SempV2Descs map[string]*SempV2Desc

type SempV2Result struct {
	v2Desc *SempV2Desc
	value  float64
}
