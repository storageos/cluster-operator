package util

import (
	"github.com/blang/semver"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("storageos.util")

// VersionSupported takes two versions, current version (haveVersion) and a
// minimum requirement version (wantVersion) and checks if the current version
// is supported by comparing it with the minimum requirement.
func VersionSupported(haveVersion, wantVersion string) bool {
	supportedVersion, err := semver.Parse(wantVersion)
	if err != nil {
		log.Info("Failed to parse version", "error", err, "want", wantVersion)
		return false
	}

	currentVersion, err := semver.Parse(haveVersion)
	if err != nil {
		log.Info("Failed to parse version", "error", err, "have", haveVersion)
		return false
	}

	return currentVersion.Compare(supportedVersion) >= 0
}
