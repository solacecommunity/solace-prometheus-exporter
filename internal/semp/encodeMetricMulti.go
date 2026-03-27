package semp

import "strings"

// Encodes string to 0,1,2,... metric
func encodeMetricMulti(item string, refItems []string) float64 {
	for i, s := range refItems {
		if strings.EqualFold(item, s) {
			return float64(i)
		}
	}
	return -1
}
