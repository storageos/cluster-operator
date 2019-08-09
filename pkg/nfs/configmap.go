package nfs

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// NFS server configuration constants.
const (
	DefaultAccessType = "readwrite"
	DefaultSquash     = "none"
	DefaultLogLevel   = "DEBUG"
)

func createConfig(instance *storageosv1.NFSServer) (string, error) {

	// id needs to be unique for each export on the server node.
	id := 57

	var exports []string
	// If no export list given, use defaults
	if len(instance.Spec.Exports) == 0 {
		exportCfg, err := exportConfig(id, instance.Name, DefaultAccessType, DefaultSquash)
		if err != nil {
			return "", err
		}
		exports = append(exports, exportCfg)
	}

	// Otherwise use export list
	for _, export := range instance.Spec.Exports {
		exportCfg, err := exportConfig(id, export.PersistentVolumeClaim.ClaimName, export.Server.AccessMode, export.Server.Squash)
		if err != nil {
			return "", err
		}
		exports = append(exports, exportCfg)
		id++
	}

	globalCfg, err := globalConfig(true, true)
	if err != nil {
		return "", err
	}

	logCfg, err := logConfig(DefaultLogLevel)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s\n%s\n%s", globalCfg, logCfg, strings.Join(exports, "\n")), nil
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

// TODO, use defualt "EVENT" level.
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

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            d.nfsServer.Name,
			Namespace:       d.nfsServer.Namespace,
			OwnerReferences: d.nfsServer.ObjectMeta.OwnerReferences,
			// Labels:          createAppLabels(nfsServer),
		},
		Data: map[string]string{
			d.nfsServer.Name: nfsConfig,
		},
	}

	return d.createOrUpdateObject(configMap)
}

func (d *Deployment) getConfigMap(name string, namespace string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}

	namespacedConfigMap := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	if err := d.client.Get(context.TODO(), namespacedConfigMap, configMap); err != nil {
		return nil, err
	}
	return configMap, nil
}

func (d *Deployment) deleteNFSConfigMap() error {
	configMap, err := d.getConfigMap(d.nfsServer.Name, d.nfsServer.Namespace)
	if err != nil {
		return err
	}
	return d.deleteObject(configMap)
}
