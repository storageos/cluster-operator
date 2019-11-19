package k8s

// k8s recommended labels from https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/ .
const (
	AppName      = "app.kubernetes.io/name"
	AppInstance  = "app.kubernetes.io/instance"
	AppVersion   = "app.kubernetes.io/version"
	AppComponent = "app.kubernetes.io/component"
	AppPartOf    = "app.kubernetes.io/part-of"
	AppManagedBy = "app.kubernetes.io/managed-by"
)

// GetDefaultAppLabels returns the default k8s app labels for resources created
// by the operator. appInstanceName should be the name of the StorageOSCluster
// object.
func GetDefaultAppLabels(appInstanceName string) map[string]string {
	return map[string]string{
		AppName:      "storageos",
		AppInstance:  appInstanceName,
		AppComponent: "cluster",
		AppPartOf:    "storageos",
		AppManagedBy: "storageos-operator",
		// NOTE: StorageOSCluster CR isn't aware of StorageOS node version. Add
		// version only when the StorageOSCluster becomes node version aware.
		// AppVersion: "",
	}
}

// AddDefaultAppLabels adds the default app labels to given labels.
func AddDefaultAppLabels(appInstanceName string, labels map[string]string) map[string]string {
	if labels == nil {
		labels = map[string]string{}
	}
	// Get the default labels and apply the existing labels over it. Passed
	// labels should override the default labels.
	outLabels := GetDefaultAppLabels(appInstanceName)
	for k, v := range labels {
		outLabels[k] = v
	}
	return outLabels
}
