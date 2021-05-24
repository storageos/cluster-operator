package storageos

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	storageosapi "github.com/storageos/cluster-operator/internal/pkg/storageos"
	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
)

const (
	// Comma separated list of endpoints on which we will try to connect to the
	// cluster's ETCD instances.
	etcdEndpointsEnvVar = "ETCD_ENDPOINTS"

	// ETCD TLS configuration information. The key/cert/CA need to be PEM encoded
	// DER bytes
	etcdTLSClientKeyEnvVar  = "ETCD_TLS_CLIENT_KEY"
	etcdTLSClientCertEnvVar = "ETCD_TLS_CLIENT_CERT"
	etcdTLSClientCAEnvVar   = "ETCD_TLS_CLIENT_CA"

	// TODO: ETCD authentication information
	// etcdUsernameEnvVar = "ETCD_USERNAME"
	// etcdPasswordEnvVar = "ETCD_PASSWORD"

	// TODO: ETCD namespace in which to operate. All keys in ETCD will be prefixed by
	// this value, allowing for multiple clusters to operate on the same ETCD
	// instance.
	// etcdNamespaceEnvVar = "ETCD_NAMESPACE"

	// Feature flags (enabled by default)
	disableTCMUEnvVar = "DISABLE_TCMU"
	forceTCMUEnvVar   = "FORCE_TCMU"

	// When set to TRUE usage data will not be logged on StorageOS servers.
	disableTelemetryEnvVar = "DISABLE_TELEMETRY"

	// When set to TRUE cluster bugs will not be logged on StorageOS servers.
	disableCrashReportingEnvVar = "DISABLE_CRASH_REPORTING"

	// When set to TRUE checks for available updates will not be carried out
	// against StorageOS servers.
	disableVersionCheckEnvVar = "DISABLE_VERSION_CHECK"

	// Namespace in which storageos operates.
	namespaceEnvVar = "K8S_NAMESPACE"

	// The kubernetes distribution in which storageos is operating.
	k8sDistroEnvVar = "K8S_DISTRO"

	// Enables the API's kubernetes specific scheduler extender endpoints.
	k8sSchedulerExtenderEnvVar = "K8S_ENABLE_SCHEDULER_EXTENDER"

	// TODO: Path to kubernetes config file.
	// kubconfigPathEnvVar = "KUBECONFIG"

	// CSI API listen socket location.  CSI is disabled if not set.
	csiEndpointEnvVar = "CSI_ENDPOINT"

	// CSI version to use, if CSI_ENDPOINT is set.
	csiVersionEnvVar = "CSI_VERSION"

	// Directory in which devices are created.
	deviceDirEnvVar = "DEVICE_DIR"

	// TODO: add to StorageOSCluster CR and optionally create Certificate CR?
	// https://cert-manager.io/docs/usage/certificate/ Since secrets will be
	// used, probably needs to be implemented in DaemonSet.
	// apiTLSCAEnvVar   = "API_TLS_CA"
	// apiTLSKeyEnvVar  = "API_TLS_KEY"
	// apiTLSCertEnvVar = "API_TLS_CERT"

	// TODO: add to StorageOSCluster CR
	// Health checking duration values
	//
	// A duration string is a possibly signed sequence of decimal numbers, each
	// with optional fraction and a unit suffix, such as "300ms", "-1.5h" or
	// "2h45m". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	// healthProbeIntervalEnvVar = "HEALTH_PROBE_INTERVAL"
	// healthProbeTimeoutEnvVar  = "HEALTH_PROBE_TIMEOUT"
	// healthGracePeriodEnvVar   = "HEALTH_GRACE_PERIOD"

	// Node capacity update interval.
	// nodeCapacityUpdateIntervalEnvVar = "NODE_CAPACITY_INTERVAL"

	// TODO: General dial timeout settings (RPC, etcd...)
	//
	// A duration string is a possibly signed sequence of decimal numbers, each
	// with optional fraction and a unit suffix, such as "300ms", "-1.5h" or
	// "2h45m". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	//
	// defaults to 5s.
	// dialTimeoutEnvVar = "DIAL_TIMEOUT"

	// Logging level: debug, info, warn or error.
	logLevelEnvVar = "LOG_LEVEL"

	// Logger format: default or json.
	logFormatEnvVar = "LOG_FORMAT"

	// recommendedPidLimitEnvVar sets the minimum max_pids limit recommended by
	// StorageOS. The init container detects the effective limit and will warn
	// if not met.
	recommendedPidLimitEnvVar = "RECOMMENDED_MAX_PIDS_LIMIT"

	// Tracing configuration.  Intended for internal development use only and
	// should not be documented externally.
	jaegerEndpointEnvVar    = "JAEGER_ENDPOINT"
	jaegerServiceNameEnvVar = "JAEGER_SERVICE_NAME"

	// Etcd TLS cert file names.
	tlsEtcdCA         = "etcd-client-ca.crt"
	tlsEtcdClientCert = "etcd-client.crt"
	tlsEtcdClientKey  = "etcd-client.key"

	// Etcd cert root path.
	tlsEtcdRootPath = "/run/storageos/pki"

	// Etcd certs volume name.
	tlsEtcdCertsVolume = "etcd-certs"
)

// ensureConfigMap creates or updates a ConfigMap to store the node container
// configuration.
func (s *Deployment) ensureConfigMap() error {
	config := configFromSpec(s.stos.Spec, CSIV1Supported(s.k8sVersion))

	labels := make(map[string]string)

	existing, err := s.k8sResourceManager.ConfigMap(configmapName, s.stos.Spec.GetResourceNS(), labels, config).Get()
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	// Create the ConfigMap or update if its data doesn't match the new config.
	if existing == nil {
		if err := s.k8sResourceManager.ConfigMap(configmapName, s.stos.Spec.GetResourceNS(), labels, config).Create(); err != nil {
			return fmt.Errorf("failed to create ConfigMap: %v", err)
		}
	} else if !reflect.DeepEqual(existing.Data, config) {
		if err := s.k8sResourceManager.ConfigMap(configmapName, s.stos.Spec.GetResourceNS(), labels, config).Update(); err != nil {
			return fmt.Errorf("failed to update ConfigMap: %v", err)
		}
	}

	// Apply a subset of the configuration directly to the StorageOS API.
	//
	// Don't attempt to update via the API if the cluster isn't ready.

	// This is likely on first bootstrap, where we need to write the ConfigMap
	// for the cluster to start, but it isn't used again.  In this case, don't
	// return an error as there is no need to re-apply after the bootstrap.
	//
	// To ensure changes to an existing CR made during an upgrade are applied,
	// we need to return an error so that the API update will get retried when
	// the cluster is back online.
	status, err := s.getStorageOSStatus()
	if err != nil {
		return fmt.Errorf("failed to get storageos status: %v", err)
	}
	if status.Phase != storageosv1.ClusterPhaseRunning {
		// No need to re-queue if the cluster isn't ready.  It will get
		// re-evaluated when the status changes.
		return nil
	}

	// Apply cluster configuration.  Don't progress on error, re-queue instead.
	if err := s.applyClusterConfig(); err != nil {
		return fmt.Errorf("failed to apply cluster config: %v", err)
	}

	return nil
}

// returns true if cluster config was updated.
func (s *Deployment) applyClusterConfig() error {
	// Load api admin credentials from secret.
	username, password, err := s.getAdminCreds()
	if err != nil {
		return err
	}

	if err := s.apiClient.Authenticate(string(username), string(password)); err != nil {
		return err
	}

	current, err := s.apiClient.GetCluster(context.Background())
	if err != nil {
		return err
	}

	want := &storageosapi.Cluster{
		DisableTelemetry:      s.stos.Spec.DisableTelemetry,
		DisableCrashReporting: s.stos.Spec.DisableTelemetry,
		DisableVersionCheck:   s.stos.Spec.DisableTelemetry,
		LogLevel:              "info", // default.
		LogFormat:             "json", // not configurable.
		Version:               current.Version,
	}
	if s.stos.Spec.Debug {
		want.LogLevel = debugVal
	}

	if current.IsEqual(want) {
		return nil
	}

	return s.apiClient.UpdateCluster(context.Background(), want)
}

// configFromSpec generates config entries.
//
//     Config set in DaemonSet env vars:
//       - HOSTNAME (reads from spec.nodeName)
//       - ADVERTISE_IP (reads from status.podIP)
//       - BOOTSTRAP_USERNAME, BOOTSTRAP_PASSWORD (reads from secret)
func configFromSpec(spec storageosv1.StorageOSClusterSpec, csiv1 bool) map[string]string {
	config := make(map[string]string)

	// ETCD_ENDPOINTS must be set to a comma separated list of endpoints.
	config[etcdEndpointsEnvVar] = spec.KVBackend.Address

	// Append Etcd TLS config, if given.  Volumes are created in Podspec.
	if spec.TLSEtcdSecretRefName != "" && spec.TLSEtcdSecretRefNamespace != "" {
		config[etcdTLSClientCAEnvVar] = filepath.Join(tlsEtcdRootPath, tlsEtcdCA)
		config[etcdTLSClientKeyEnvVar] = filepath.Join(tlsEtcdRootPath, tlsEtcdClientKey)
		config[etcdTLSClientCertEnvVar] = filepath.Join(tlsEtcdRootPath, tlsEtcdClientCert)
	}

	// Always show telemetry and feature options to ensure they're visble.
	config[disableTelemetryEnvVar] = strconv.FormatBool(spec.DisableTelemetry)

	// TODO: separate CR items for version check and crash reports.  Use
	// Telemetry to enable/disable everything for now.
	config[disableVersionCheckEnvVar] = strconv.FormatBool(spec.DisableTelemetry)
	config[disableCrashReportingEnvVar] = strconv.FormatBool(spec.DisableTelemetry)

	// DISABLE_TCMU and FORCE_TCMU should not be set unless under advice from
	// support.  Only show the vars if set.
	if spec.DisableTCMU {
		config[disableTCMUEnvVar] = strconv.FormatBool(spec.DisableTCMU)
	}
	if spec.ForceTCMU {
		config[forceTCMUEnvVar] = strconv.FormatBool(spec.ForceTCMU)
	}

	config[namespaceEnvVar] = spec.GetResourceNS()

	if spec.K8sDistro != "" {
		config[k8sDistroEnvVar] = spec.K8sDistro
	}

	// CSI is always enabled.
	config[csiEndpointEnvVar] = spec.GetCSIEndpoint(true)
	config[csiVersionEnvVar] = spec.GetCSIVersion(true)

	// Since we're running in k8s, always listen on the the scheduler extender
	// api endpoints.  The feature can be disabled with the operator.  This
	// allows users to toggle the feature without restarting the cluster.
	config[k8sSchedulerExtenderEnvVar] = "true"

	// If kubelet is running in a container, sharedDir should be set.
	if spec.SharedDir != "" {
		config[deviceDirEnvVar] = fmt.Sprintf("%s/devices", spec.SharedDir)
	}

	config[logFormatEnvVar] = "json"
	config[logLevelEnvVar] = "info"
	if spec.Debug {
		config[logLevelEnvVar] = debugVal
	}

	// Always set max_pids recommendation.
	config[recommendedPidLimitEnvVar] = fmt.Sprint(recommendedPidLimit)

	// Set Jaeger configuration, only if set as an env var in the operator. We
	// do this because we don't want to publish configuration options in the CRD
	// that are intended for developer use only.
	if val := os.Getenv(jaegerEndpointEnvVar); val != "" {
		config[jaegerEndpointEnvVar] = val
	}
	if val := os.Getenv(jaegerServiceNameEnvVar); val != "" {
		config[jaegerServiceNameEnvVar] = val
	}

	return config
}
