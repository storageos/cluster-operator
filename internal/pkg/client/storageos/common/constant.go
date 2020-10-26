package common

import "time"

const (
	// APIUsernameKey is the name of the k8s secret key that stores StorageOS
	// username.
	APIUsernameKey = "apiUsername"
	// APIPasswordKey is the name of the k8s secret key that stores StorageOS
	// password.
	APIPasswordKey = "apiPassword"
	// UserAgent is the user-agent name of the StorageOS client.
	UserAgent = "cluster-operator/v2.3.0"
	// DefaultScheme is the default scheme of the StorageOS API endpoint.
	DefaultScheme = "http"
	// TLSScheme is the TLS scheme of the StorageOS API endpoint.
	TLSScheme = "https"
	// HTTPTimeout is the http request timeout for StorageOS API clients.
	HTTPTimeout = 10 * time.Second
)
