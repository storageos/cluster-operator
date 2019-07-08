package k8sutil

import (
	"testing"
)

func TestGetBaseK8SVersion(t *testing.T) {

	testcases := []struct {
		name        string
		version     string
		wantVersion string
	}{
		{
			name:        "simple base version",
			version:     "v1.14.1",
			wantVersion: "1.14.1",
		},
		{
			name:        "version with dash",
			version:     "v1.14.1-gke-qnxo4",
			wantVersion: "1.14.1",
		},
		{
			name:        "version with plus",
			version:     "v1.14.1+nxiel",
			wantVersion: "1.14.1",
		},
		{
			name:        "version with no prefix",
			version:     "1.14.1-aks-foo",
			wantVersion: "1.14.1",
		},
		{
			// https://github.com/blang/semver/issues/55 patch value 0 case.
			name:        "version with patch 0",
			version:     "v1.11.0+d4cacc0",
			wantVersion: "1.11.0",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			k := K8SOps{}
			version, err := k.getBaseK8SVersion(tc.version)
			if err != nil {
				t.Error(err)
			}

			if version != tc.wantVersion {
				t.Errorf("unexpected k8s version:\n\t(WNT) %q\n\t(GOT) %q", tc.wantVersion, version)
			}
		})
	}
}
