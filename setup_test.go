package policy

import (
	"testing"

	"github.com/coredns/caddy"
)

func TestSetup(t *testing.T) {
	tests := []struct {
		input     string
		shouldErr bool
	}{
		{"policy", false},
		{"policy /some/file {\n base64 \n}", false},
		{"policy /some/file {\n base64\n period 24h\n cache_dir /tmp/policy \n}", false},
	}

	for i, test := range tests {
		_, err := parseStanza(caddy.NewTestController("dns", test.input))

		if test.shouldErr && err == nil {
			t.Errorf("Test %d: Expected error but found none for input %s", i, test.input)
		}
		if err != nil && !test.shouldErr {
			t.Errorf("Test %d: Expected no error but found one for input %s. Error was: %v", i, test.input, err)
		}
		if test.shouldErr {
			continue
		}
	}
}
