package semp

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/prometheus/client_golang/prometheus"
)

type PrometheusMetric struct {
	desc        *Desc
	valueType   prometheus.ValueType
	value       float64
	labelValues []string
	deprecated  bool
}

func (semp *Semp) NewMetric(desc *Desc, valueType prometheus.ValueType, value float64, labelValues ...string) PrometheusMetric {
	err := validateLabelValues(labelValues, len(desc.variableLabels))
	if err != nil {
		panic(err)
	}

	return PrometheusMetric{
		desc:        desc,
		valueType:   valueType,
		value:       value,
		labelValues: labelValues,
		deprecated:  false,
	}
}

var errInconsistentCardinality = errors.New("inconsistent label cardinality")

func validateLabelValues(vals []string, expectedNumberOfValues int) error {
	if len(vals) != expectedNumberOfValues {
		return fmt.Errorf(
			"%w: expected %d label values but got %d in %#v",
			errInconsistentCardinality, expectedNumberOfValues,
			len(vals), vals,
		)
	}

	for _, val := range vals {
		if !utf8.ValidString(val) {
			return fmt.Errorf("label value %q is not valid UTF-8", val)
		}
	}

	return nil
}

func (metric *PrometheusMetric) Name() string {
	if len(metric.desc.variableLabels) < 1 {
		return metric.desc.fqName
	}

	labelStrings := make([]string, len(metric.desc.variableLabels))
	for index, variableLabel := range metric.desc.variableLabels {
		variableLabelValue := metric.labelValues[index]
		labelStrings[index] = variableLabel + "=\"" + variableLabelValue + "\""
	}
	return metric.desc.fqName + "{" + strings.Join(labelStrings, ",") + "}"
}

func (metric *PrometheusMetric) AsPrometheusMetric() prometheus.Metric {
	return prometheus.MustNewConstMetric(metric.desc.AsPrometheusDesc(), metric.valueType, metric.value, metric.labelValues...)
}

func (metric *PrometheusMetric) Deprecate() {
	metric.deprecated = true
}

func (metric *PrometheusMetric) IsDeprecated() bool {
	return metric.deprecated
}
