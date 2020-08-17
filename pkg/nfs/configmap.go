package nfs

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	corev1 "k8s.io/api/core/v1"
)

// NFS server configuration constants.
const (
	DefaultExportName        = "NFSExport"
	DefaultExportPVCReadOnly = false
	DefaultAccessType        = "readwrite"
	DefaultSquash            = "none"
	DefaultLogLevel          = "DEBUG"
	DefaultGraceless         = true
	DefaultFsidDevice        = false
)

func createConfig(instance *storageosv1.NFSServer) (string, error) {

	// id needs to be unique for each export on the server node.
	id := 57

	var exportCfg string

	// Get export spec and obtain export config.
	exportSpec := getExportSpec(instance)
	export, err := exportConfig(id, exportSpec.PersistentVolumeClaim.ClaimName, exportSpec.Server.AccessMode, exportSpec.Server.Squash)
	if err != nil {
		return "", err
	}
	exportCfg = export

	globalCfg, err := globalConfig(DefaultGraceless, DefaultFsidDevice)
	if err != nil {
		return "", err
	}

	logCfg, err := logConfig(DefaultLogLevel)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s\n%s\n%s", globalCfg, logCfg, exportCfg), nil
}

// getExportSpec returns a NFS ExportSpec based on the NFSServer instance.
func getExportSpec(instance *storageosv1.NFSServer) storageosv1.ExportSpec {
	// Default export spec using the instance name as the PVC name.
	// ExportSpec.Name is not used anywhere, use a default name.
	exportSpec := storageosv1.ExportSpec{
		Name: DefaultExportName,
		PersistentVolumeClaim: corev1.PersistentVolumeClaimVolumeSource{
			ClaimName: instance.Name,
			ReadOnly:  DefaultExportPVCReadOnly,
		},
		Server: storageosv1.ServerSpec{
			AccessMode: DefaultAccessType,
			Squash:     DefaultSquash,
		},
	}

	// If PVC is specified in the spec, use the specified PVC.
	if instance.Spec.PersistentVolumeClaim.ClaimName != "" {
		exportSpec.PersistentVolumeClaim = instance.Spec.PersistentVolumeClaim
	}

	// If Export is specified in the spec, use the specified export.
	if instance.Spec.Export.Name != "" {
		exportSpec = instance.Spec.Export
	}

	return exportSpec
}

// nfsExportConfig is the NFS server export configuration.
type nfsExportConfig struct {
	ID         int
	Name       string
	AccessType string
	Squash     string
}

func exportConfig(id int, ref string, access string, squash string) (string, error) {
	exportConfigTemplate := `
EXPORT {
	Export_Id = {{.ID}};
	Path = /export/{{.Name}};
	Pseudo = /{{.Name}};
	Protocols = 4;
	Transports = TCP;
	Sectype = sys;
	Access_Type = {{.AccessType}};
	Squash = {{.Squash}};
	FSAL {
		Name = VFS;
		fsid_type = None;
	}
}`
	exportConfigData := nfsExportConfig{
		ID:         id,
		Name:       ref,
		AccessType: getAccessMode(access),
		Squash:     getSquash(squash),
	}
	return renderConfig("exportConfig", exportConfigTemplate, exportConfigData)
}

type nfsGlobalConfig struct {
	Graceless  bool
	FSIDDevice bool
}

func globalConfig(graceless, fsidDevice bool) (string, error) {
	globalConfigTemplate := `
NFSv4 {
	Graceless = {{.Graceless}};
}
NFS_Core_Param {
	fsid_device = {{.FSIDDevice}};
}`
	globalConfigData := nfsGlobalConfig{
		Graceless:  graceless,
		FSIDDevice: fsidDevice,
	}
	return renderConfig("globalConfig", globalConfigTemplate, globalConfigData)
}

type nfsLogConfig struct {
	LogLevel string
}

// TODO, use default "EVENT" level.
func logConfig(logLevel string) (string, error) {
	logConfigTemplate := `
LOG {
	default_log_level = {{.LogLevel}};
	Components {
		ALL = {{.LogLevel}};
	}
}`
	logConfigData := nfsLogConfig{
		LogLevel: logLevel,
	}
	return renderConfig("logConfig", logConfigTemplate, logConfigData)
}

// renderConfig takes template name, template of a configuration and config data
// and returns a rendered configuration.
func renderConfig(templateName, configTemplate string, config interface{}) (string, error) {
	var configuration bytes.Buffer
	tmpl, err := template.New(templateName).Parse(configTemplate)
	if err != nil {
		return "", err
	}

	if err := tmpl.Execute(&configuration, config); err != nil {
		return "", err
	}

	return configuration.String(), nil
}

// getAccessMode converts the access mode in NFSServer config to nfs-ganesha
// access modes.
func getAccessMode(mode string) string {
	switch strings.ToLower(mode) {
	case "none":
		return "None"
	case "readonly":
		return "RO"
	default:
		return "RW"
	}
}

func getSquash(squash string) string {
	if squash != "" {
		return strings.ToLower(squash)
	}
	return "none"
}

func (d *Deployment) createNFSConfigMap() error {
	nfsConfig, err := createConfig(d.nfsServer)
	if err != nil {
		return err
	}

	data := map[string]string{
		d.nfsServer.Name: nfsConfig,
	}

	return d.k8sResourceManager.ConfigMap(d.nfsServer.Name, d.nfsServer.Namespace, nil, data).Create()
}
