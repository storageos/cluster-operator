package image

import (
	"os"
	"testing"
)

func TestGetDefaultImage(t *testing.T) {
	fakeNFSDefaultImage := "nfs/foo:1"
	fakeStorageOSNodeImage := "stos/foo:1"

	testcases := []struct {
		name          string
		envVars       map[string]string
		defaultImages map[string]string
		wantImages    map[string]string
	}{
		{
			name: "images from env vars",
			envVars: map[string]string{
				NFSImageEnvVar:           fakeNFSDefaultImage,
				StorageOSNodeImageEnvVar: fakeStorageOSNodeImage,
			},
			defaultImages: map[string]string{
				NFSImageEnvVar:           DefaultNFSContainerImage,
				StorageOSNodeImageEnvVar: DefaultNodeContainerImage,
			},
			wantImages: map[string]string{
				NFSImageEnvVar:           fakeNFSDefaultImage,
				StorageOSNodeImageEnvVar: fakeStorageOSNodeImage,
			},
		},
		{
			name:    "images not in env var",
			envVars: map[string]string{},
			defaultImages: map[string]string{
				NFSImageEnvVar:           DefaultNFSContainerImage,
				StorageOSNodeImageEnvVar: DefaultNodeContainerImage,
			},
			wantImages: map[string]string{
				NFSImageEnvVar:           DefaultNFSContainerImage,
				StorageOSNodeImageEnvVar: DefaultNodeContainerImage,
			},
		},
		{
			name: "some images in env var and some defaults",
			envVars: map[string]string{
				NFSImageEnvVar:           fakeNFSDefaultImage,
				StorageOSNodeImageEnvVar: fakeStorageOSNodeImage,
			},
			defaultImages: map[string]string{
				NFSImageEnvVar:                DefaultNFSContainerImage,
				StorageOSNodeImageEnvVar:      DefaultNodeContainerImage,
				CSIv1LivenessProbeImageEnvVar: CSIv1LivenessProbeContainerImage,
			},
			wantImages: map[string]string{
				NFSImageEnvVar:                fakeNFSDefaultImage,
				StorageOSNodeImageEnvVar:      fakeStorageOSNodeImage,
				CSIv1LivenessProbeImageEnvVar: CSIv1LivenessProbeContainerImage,
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Set the env vars.
			for k, v := range tc.envVars {
				os.Setenv(k, v)
			}

			// Unset the env vars at the end.
			defer func() {
				for k := range tc.envVars {
					os.Unsetenv(k)
				}
			}()

			// Get the default images and check.
			for k, v := range tc.wantImages {
				got := GetDefaultImage(k, tc.defaultImages[k])
				if v != got {
					t.Errorf("unexpected default images for %s:\n\t(WNT) %s\n\t(GOT) %s", k, v, got)
				}
			}
		})
	}
}
