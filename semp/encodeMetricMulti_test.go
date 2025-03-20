package semp

import (
	"testing"
)

func TestEncodeMetricMulti(t *testing.T) {
	tests := []struct {
		item     string
		refItems []string
		want     float64
	}{
		{"apple", []string{"apple", "banana", "cherry"}, 0},
		{"banana", []string{"apple", "banana", "cherry"}, 1},
		{"cherry", []string{"apple", "banana", "cherry"}, 2},
		{"APPLE", []string{"apple", "banana", "cherry"}, 0},  // Case-insensitive match
		{"Banana", []string{"apple", "banana", "cherry"}, 1}, // Case-insensitive match
		{"grape", []string{"apple", "banana", "cherry"}, -1}, // Item not in list
		{"", []string{"apple", "banana", "cherry"}, -1},      // Empty string
	}

	for _, test := range tests {
		result := encodeMetricMulti(test.item, test.refItems)
		if result != test.want {
			t.Errorf("encodeMetricMulti(%q, %v) = %v; want %v", test.item, test.refItems, result, test.want)
		}
	}
}
