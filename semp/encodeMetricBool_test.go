package semp

import (
	"testing"
)

func TestEncodeMetricBool(t *testing.T) {
	tests := []struct {
		item bool
		want float64
	}{
		{true, 1},
		{false, 0},
	}

	for _, test := range tests {
		result := encodeMetricBool(test.item)
		if result != test.want {
			t.Errorf("encodeMetricBool(%v) = %v; want %v", test.item, result, test.want)
		}
	}
}
