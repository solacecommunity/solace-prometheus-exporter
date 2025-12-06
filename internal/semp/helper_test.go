package semp

import (
	"net/url"
	"testing"
)

func equalUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aMap := make(map[string]int)
	bMap := make(map[string]int)

	for _, v := range a {
		aMap[v]++
	}
	for _, v := range b {
		bMap[v]++
	}

	for k, v := range aMap {
		if bMap[k] != v {
			return false
		}
	}

	return true
}

func TestMapItems(t *testing.T) {
	t.Parallel()

	translateMap := map[string]string{
		"metric1": "field1",
		"metric2": "field2",
	}

	tests := []struct {
		name    string
		items   []string
		want    []string
		wantErr string
	}{
		{
			name:    "Valid items - translate keys",
			items:   []string{"metric1", "metric2"},
			want:    []string{"field1", "field2"},
			wantErr: "",
		},
		{
			name:    "Valid items - raw fields",
			items:   []string{"field1"},
			want:    []string{"field1"},
			wantErr: "",
		},
		{
			name:    "Mixed valid keys and fields",
			items:   []string{"metric1", "field2"},
			want:    []string{"field1", "field2"},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := mapItems(tt.items, translateMap)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error but got nil")
				}
				if err.Error() != tt.wantErr {
					t.Errorf("unexpected error: got %v, want %v", err.Error(), tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !equalUnordered(tt.want, result) {
					t.Errorf("unexpected result: got %v, want %v", result, tt.want)
				}
			}
		})
	}
}

func TestQueryEscape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Empty string",
			input: "",
			want:  "",
		},
		{
			name:  "Already encoded string",
			input: "field%201",
			want:  "field%201",
		},
		{
			name:  "String with spaces",
			input: "field 1",
			want:  "field+1",
		},
		{
			name:  "String with special characters",
			input: "field@#&",
			want:  url.QueryEscape("field@#&"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := queryEscape(tt.input)
			if result != tt.want {
				t.Errorf("unwant result: got %v, want %v", result, tt.want)
			}
		})
	}
}
