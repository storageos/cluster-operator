package nfs

import (
	"strings"
	"testing"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testConfig struct {
	Foo string
	Bar string
}

func TestRenderConfig(t *testing.T) {
	someTemplate := `
foo = {{.Foo}}
bar = {{.Bar}}
`
	wantRender := `
foo = foo1
bar = bar1
`

	someData := testConfig{
		Foo: "foo1",
		Bar: "bar1",
	}

	render, err := renderConfig("somefoo", someTemplate, someData)
	if err != nil {
		t.Error("failed to render config", err)
	}

	if render != wantRender {
		t.Errorf("unexpected template renders:\n\t(WNT) %v\n\t(GOT) %v", wantRender, render)
	}
}

func TestCreateConfig(t *testing.T) {
	testcases := []struct {
		name          string
		nfsServerSpec storageosv1.NFSServerSpec
		wantConfig    string
		wantErr       bool
	}{
		{
			name:          "default nfs server spec",
			nfsServerSpec: storageosv1.NFSServerSpec{},
			wantConfig: `
NFSv4 {
	Graceless = true;
}
NFS_Core_Param {
	fsid_device = true;
}

LOG {
	default_log_level = DEBUG;
	Components {
		ALL = DEBUG;
	}
}

EXPORT {
	Export_Id = 57;
	Path = /export/test-nfs;
	Pseudo = /test-nfs;
	Protocols = 4;
	Transports = TCP;
	Sectype = sys;
	Access_Type = RW;
	Squash = none;
	FSAL {
		Name = VFS;
	}
}`,
		},
		{
			name: "nfs server spec with exports",
			nfsServerSpec: storageosv1.NFSServerSpec{
				Exports: []storageosv1.ExportsSpec{
					{
						Name: "export1",
						Server: storageosv1.ServerSpec{
							AccessMode: "readonly",
							Squash:     "test-squash",
						},
						PersistentVolumeClaim: corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "test-claim",
							ReadOnly:  false,
						},
					},
					{
						Name:   "export2",
						Server: storageosv1.ServerSpec{},
						PersistentVolumeClaim: corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: "test-claim-2",
							ReadOnly:  false,
						},
					},
				},
			},
			wantConfig: `
NFSv4 {
	Graceless = true;
}
NFS_Core_Param {
	fsid_device = true;
}

LOG {
	default_log_level = DEBUG;
	Components {
		ALL = DEBUG;
	}
}

EXPORT {
	Export_Id = 57;
	Path = /export/test-claim;
	Pseudo = /test-claim;
	Protocols = 4;
	Transports = TCP;
	Sectype = sys;
	Access_Type = RO;
	Squash = test-squash;
	FSAL {
		Name = VFS;
	}
}

EXPORT {
	Export_Id = 58;
	Path = /export/test-claim-2;
	Pseudo = /test-claim-2;
	Protocols = 4;
	Transports = TCP;
	Sectype = sys;
	Access_Type = RW;
	Squash = none;
	FSAL {
		Name = VFS;
	}
}`,
		},
	}

	for _, tc := range testcases {
		nfsServer := &storageosv1.NFSServer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-nfs",
				Namespace: "default",
			},
			Spec: tc.nfsServerSpec,
		}

		gotConfig, err := createConfig(nfsServer)
		if err != nil {
			t.Fatal("failed to create config", err)
		}

		if strings.TrimSpace(tc.wantConfig) != strings.TrimSpace(gotConfig) {
			t.Errorf("unexpected nfs config:\n\t(WNT) %s\n\t(GOT) %s", tc.wantConfig, gotConfig)
		}
	}
}
