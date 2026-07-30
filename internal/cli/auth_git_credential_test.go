package cli

import (
	"strings"
	"testing"
)

func TestReadCredentialInput(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		expectedAttributes map[string]string
	}{
		{
			name:               "typical get request",
			input:              "protocol=https\nhost=knot1.tangled.sh\n\n",
			expectedAttributes: map[string]string{"protocol": "https", "host": "knot1.tangled.sh"},
		},
		{
			name:               "no blank terminator at EOF",
			input:              "host=knot1.tangled.sh",
			expectedAttributes: map[string]string{"host": "knot1.tangled.sh"},
		},
		{
			name:               "empty input",
			input:              "",
			expectedAttributes: map[string]string{},
		},
		{
			name:               "ignores lines without equals",
			input:              "garbage\nhost=knot1.tangled.sh\n\n",
			expectedAttributes: map[string]string{"host": "knot1.tangled.sh"},
		},
		{
			name:               "preserves path with slashes",
			input:              "protocol=https\nhost=knot1.tangled.sh\npath=aly.codes/morsels.git\n\n",
			expectedAttributes: map[string]string{"protocol": "https", "host": "knot1.tangled.sh", "path": "aly.codes/morsels.git"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			credentialAttributes, err := readCredentialInput(strings.NewReader(testCase.input))
			if err != nil {
				t.Fatalf("readCredentialInput() error = %v", err)
			}
			if len(credentialAttributes) != len(testCase.expectedAttributes) {
				t.Fatalf("attribute count = %d, want %d (%v)", len(credentialAttributes), len(testCase.expectedAttributes), credentialAttributes)
			}
			for key, expectedValue := range testCase.expectedAttributes {
				if credentialAttributes[key] != expectedValue {
					t.Errorf("%s = %q, want %q", key, credentialAttributes[key], expectedValue)
				}
			}
		})
	}
}

func TestIsHTTPSCredentialRequest(t *testing.T) {
	tests := []struct {
		name  string
		attrs map[string]string
		want  bool
	}{
		{name: "HTTPS host", attrs: map[string]string{"protocol": "https", "host": "knot.example"}, want: true},
		{name: "case insensitive protocol", attrs: map[string]string{"protocol": "HTTPS", "host": "knot.example"}, want: true},
		{name: "HTTP host", attrs: map[string]string{"protocol": "http", "host": "knot.example"}, want: false},
		{name: "missing protocol", attrs: map[string]string{"host": "knot.example"}, want: false},
		{name: "missing host", attrs: map[string]string{"protocol": "https"}, want: false},
		{name: "blank host", attrs: map[string]string{"protocol": "https", "host": " "}, want: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if got := isHTTPSCredentialRequest(testCase.attrs); got != testCase.want {
				t.Errorf("isHTTPSCredentialRequest(%v) = %v, want %v", testCase.attrs, got, testCase.want)
			}
		})
	}
}
