package version

import "testing"

func TestIsSupported(t *testing.T) {
	cases := map[string]struct {
		haveVersion string
		wantVersion string
		expected    bool
	}{
		"Wanted version is not semver": {
			wantVersion: "non-sem-ver",
			expected:    false,
		},
		"Have version is not semver": {
			wantVersion: "1.2.3",
			haveVersion: "non-sem-ver",
			expected:    false,
		},
		"Have patch version is lower": {
			wantVersion: "1.2.3",
			haveVersion: "1.2.2",
			expected:    false,
		},
		"Have minor version is lower": {
			wantVersion: "1.2.3",
			haveVersion: "1.1.3",
			expected:    false,
		},
		"Have major version is lower": {
			wantVersion: "1.2.3",
			haveVersion: "0.2.3",
			expected:    false,
		},
		"Have version is equal": {
			wantVersion: "1.2.3",
			haveVersion: "1.2.3",
			expected:    true,
		},
		"Have version is greater": {
			wantVersion: "1.2.3",
			haveVersion: "1.2.4",
			expected:    true,
		},
	}

	for name, data := range cases {
		data := data
		t.Run(name, func(t *testing.T) {
			actual := IsSupported(data.haveVersion, data.wantVersion)
			if actual != data.expected {
				t.Errorf("unexpected version check result:\n\t(WNT) %t\n\t(GOT) %t", data.expected, actual)
			}
		})
	}
}
