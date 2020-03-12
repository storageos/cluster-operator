package v1

import (
	"fmt"

	storageosapi "github.com/storageos/go-api"
	corev1 "k8s.io/api/core/v1"

	"github.com/storageos/cluster-operator/internal/pkg/client/storageos/common"
)

// NewClient creats a new StorageOS v1 client and returns it.
func NewClient(ip string, username, password string) (*storageosapi.Client, error) {
	client, err := storageosapi.NewVersionedClient(fmt.Sprintf("%s:%s", ip, storageosapi.DefaultPort), storageosapi.DefaultVersionStr)
	if err != nil {
		return client, err
	}

	client.SetUserAgent(common.UserAgent)
	client.SetAuth(username, password)
	return client, nil
}

// NewClientFromSecret extracts credentials from a given k8s secret resource and
// uses it to create and return a new StorageOS v1 client.
func NewClientFromSecret(ip string, secret *corev1.Secret) (*storageosapi.Client, error) {
	username := string(secret.Data[common.APIUsernameKey])
	password := string(secret.Data[common.APIPasswordKey])

	return NewClient(ip, username, password)
}
