package semp

import (
	"testing"
	"time"
)

// sample SEMPv1 openssl-text cert dump; Issuer CN precedes Subject CN on purpose
const leafCertText = `        Version: 3 (0x2)
        Serial Number:
            4a:89:b4:f7:6e:70:a0:54:f0:00:35:7f:5f:61:e6:44:f5:90:75:80
        Signature Algorithm: sha256WithRSAEncryption
        Issuer:
            O=Example Org
            OU=Example OU
            CN=Example Sub CA
        Validity
            Not Before: Jun 11 13:16:49 2026 GMT
            Not After : Sep  9 13:17:19 2026 GMT
        Subject:
            C=DE
            O=Example Org
            OU=Example: bridges
            CN=bridge.example-env.example.com
        Subject Public Key Info:
            Public Key Algorithm: id-ecPublicKey
`

func TestParseCertTextTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		label    string
		wantUnix int64
		wantErr  bool
	}{
		{
			// label "Not After" matches the space-padded "Not After :" line
			name:     "Not After with space-padded day",
			label:    "Not After",
			wantUnix: time.Date(2026, 9, 9, 13, 17, 19, 0, time.UTC).Unix(),
		},
		{
			name:     "Not Before with two-digit day",
			label:    "Not Before",
			wantUnix: time.Date(2026, 6, 11, 13, 16, 49, 0, time.UTC).Unix(),
		},
		{
			name:    "missing label",
			label:   "Not Ever",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseCertTextTime(leafCertText, tt.label)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Unix() != tt.wantUnix {
				t.Errorf("got %d (%s), want %d", got.Unix(), got.UTC(), tt.wantUnix)
			}
		})
	}
}

func TestParseCertTextTimeMalformed(t *testing.T) {
	t.Parallel()

	_, err := parseCertTextTime("            Not After : totally-not-a-date\n", "Not After")
	if err == nil {
		t.Fatalf("expected parse error for malformed date, got nil")
	}
}

// the label match must tolerate openssl's variable spacing around the colon
func TestParseCertTextTimeSpacingTolerant(t *testing.T) {
	t.Parallel()

	want := time.Date(2026, 9, 9, 13, 17, 19, 0, time.UTC).Unix()
	for _, line := range []string{
		"            Not After : Sep  9 13:17:19 2026 GMT",
		"            Not After: Sep  9 13:17:19 2026 GMT",
		"Not After   :   Sep  9 13:17:19 2026 GMT",
	} {
		got, err := parseCertTextTime(line+"\n", "Not After")
		if err != nil {
			t.Fatalf("line %q: unexpected error: %v", line, err)
		}
		if got.Unix() != want {
			t.Errorf("line %q: got %d, want %d", line, got.Unix(), want)
		}
	}
}

func TestParseCertTextCN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "subject CN, not issuer CN",
			in:   leafCertText,
			want: "bridge.example-env.example.com",
		},
		{
			name: "no subject block",
			in:   "        Version: 3 (0x2)\n        Issuer:\n            CN=some-ca\n",
			want: "",
		},
		{
			name: "empty",
			in:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := parseCertTextCN(tt.in); got != tt.want {
				t.Errorf("parseCertTextCN() = %q, want %q", got, tt.want)
			}
		})
	}
}
