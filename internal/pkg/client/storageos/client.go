package storageos

import (
	"context"

	storageosapiv1 "github.com/storageos/go-api"
	storageosapiv2 "github.com/storageos/go-api/v2"
)

// Client is a StorageOS API client, consisting of both v1 and v2 client.
type Client struct {
	V1  *storageosapiv1.Client
	V2  *storageosapiv2.APIClient
	Ctx context.Context
}
