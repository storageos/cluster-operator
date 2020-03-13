package v2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	storageosapiv2 "github.com/storageos/go-api/v2"
	corev1 "k8s.io/api/core/v1"

	"github.com/storageos/cluster-operator/internal/pkg/client/storageos/common"
	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
)

// ErrNoAuthToken is returned when the API client did not get an error
// during authentication but no valid auth token was returned.
var ErrNoAuthToken = errors.New("no token found in auth response")

// NewClient creates a new StorageOS v2 client and returns it.
func NewClient(ip, username, password string) (context.Context, *storageosapiv2.APIClient, error) {
	config := storageosapiv2.NewConfiguration()

	u, err := url.Parse(fmt.Sprintf("%s://%s:%d", common.DefaultScheme, ip, storageosv1.DefaultServiceExternalPort))
	if err != nil {
		return nil, nil, err
	}

	config.Scheme = u.Scheme
	config.Host = u.Host
	config.UserAgent = common.UserAgent

	httpc := &http.Client{
		Timeout:   common.HTTPTimeout,
		Transport: http.DefaultTransport,
	}
	config.HTTPClient = httpc

	// Disable TLS until we are until the operator is able to configure API
	// certs.
	// if u.Scheme == common.TLSScheme {
	// 	tlsConfig := &tls.Config{
	// 		InsecureSkipVerify: true,
	// 	}

	// 	tr := &http.Transport{
	// 		TLSClientConfig: tlsConfig,
	// 	}
	// 	config.HTTPClient = &http.Client{
	// 		Timeout:   common.HTTPTimeout,
	// 		Transport: tr,
	// 	}
	// }

	client := storageosapiv2.NewAPIClient(config)

	ctx, err := authenticate(client, username, password)
	if err != nil {
		return nil, nil, err
	}

	return ctx, client, nil
}

// NewClientFromSecret extracts credentials from a given k8s secret resource and
// uses it to create and return a new StorageOS v2 client.
func NewClientFromSecret(ip string, secret *corev1.Secret) (context.Context, *storageosapiv2.APIClient, error) {
	username := string(secret.Data[common.APIUsernameKey])
	password := string(secret.Data[common.APIPasswordKey])

	return NewClient(ip, username, password)
}

func authenticate(client *storageosapiv2.APIClient, username, password string) (context.Context, error) {
	// Create context just for the login.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// Initial basic auth to retrieve the jwt token.
	_, resp, err := client.DefaultApi.AuthenticateUser(ctx, storageosapiv2.AuthUserData{
		Username: username,
		Password: password,
	})
	if err != nil {
		return nil, GetAPIErrorResponse(err)
	}

	token := respAuthToken(resp)
	if token != "" {
		return context.WithValue(context.Background(), storageosapiv2.ContextAccessToken, token), nil
	}

	return nil, ErrNoAuthToken
}

// respAuthToken is a helper to pull the auth token out of a HTTP Response.
func respAuthToken(resp *http.Response) string {
	if value := resp.Header.Get("Authorization"); value != "" {
		// "Bearer aaaabbbbcccdddeeeff"
		return strings.Split(value, " ")[1]
	}
	return ""
}

// GetAPIErrorResponse returns the actual API response error incl. the response
// Body.
func GetAPIErrorResponse(oerr error) error {
	if n, ok := oerr.(storageosapiv2.GenericOpenAPIError); ok {
		return fmt.Errorf("%s: %s", strings.TrimSuffix(n.Error(), "\n"), n.Body())
	}
	return oerr
}
