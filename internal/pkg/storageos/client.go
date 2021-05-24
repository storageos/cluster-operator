package storageos

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	api "github.com/storageos/go-api/v2"
)

//go:generate mockgen -build_flags=--mod=vendor -destination=mocks/mock_control_plane.go -package=mocks github.com/storageos/cluster-operator/internal/pkg/storageos ControlPlane

// ControlPlane is the subset of the StorageOS control plane ControlPlane that
// the operator requires.  New methods should be added here as needed, then the
// mocks regenerated.
type ControlPlane interface {
	RefreshJwt(ctx context.Context) (api.UserSession, *http.Response, error)
	AuthenticateUser(ctx context.Context, authUserData api.AuthUserData) (api.UserSession, *http.Response, error)
	GetCluster(ctx context.Context) (api.Cluster, *http.Response, error)
	UpdateCluster(ctx context.Context, updateClusterData api.UpdateClusterData, localVarOptionals *api.UpdateClusterOpts) (api.Cluster, *http.Response, error)
}

// Client provides access to the StorageOS API.
type Client struct {
	api ControlPlane
	ctx context.Context
}

const (
	// DefaultPort is the default api port.
	DefaultPort = 5705

	// DefaultScheme is used for api endpoint.
	DefaultScheme = "http"
)

var (
	// ErrNoAuthToken is returned when the API client did not get an error
	// during authentication but no valid auth token was returned.
	ErrNoAuthToken = errors.New("no token found in auth response")

	// HTTPTimeout is the time limit for requests made by the API Client. The
	// timeout includes connection time, any redirects, and reading the response
	// body. The timer remains running after Get, Head, Post, or Do return and
	// will interrupt reading of the Response.Body.
	HTTPTimeout = 10 * time.Second

	// AuthenticationTimeout is the time limit for authentication requests to
	// complete.  It should be longer than the HTTPTimeout.
	AuthenticationTimeout = 20 * time.Second
)

// Mocked returns a client that uses the provided ControlPlane api client.
// Intended for tests that use a mocked StorageOS api.  This avoids having to
// publically expose the api on the Client struct.
func Mocked(api ControlPlane) *Client {
	return &Client{
		api: api,
		ctx: context.TODO(),
	}
}

// New returns an unauthenticated client for the StorageOS API.  Authenticate()
// must be called before using the client.
func New(endpoint string) *Client {
	config := api.NewConfiguration()

	if !strings.Contains(endpoint, "://") {
		endpoint = fmt.Sprintf("%s://%s", DefaultScheme, endpoint)
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		// This should never happen as we control the endpoint that is passed
		// in.  It allows us to create a client in places that are unable to
		// handle an error gracefully.
		panic(err)
	}

	config.Scheme = u.Scheme
	config.Host = u.Host
	if !strings.Contains(u.Host, ":") {
		config.Host = fmt.Sprintf("%s:%d", u.Host, DefaultPort)
	}

	config.HTTPClient = &http.Client{
		Timeout:   HTTPTimeout,
		Transport: http.DefaultTransport,
	}

	// Get a wrappered API client.
	client := api.NewAPIClient(config)

	return &Client{api: client.DefaultApi, ctx: context.TODO()}
}

// Authenticate against the API and set the authentication token in the client
// to be used for subsequent API requests.  The token must be refreshed
// periodically using AuthenticateRefresh(), or Authenticate() called again.
func (c *Client) Authenticate(username, password string) error {
	// Create context just for the login.
	ctx, cancel := context.WithTimeout(context.Background(), AuthenticationTimeout)
	defer cancel()

	// Initial basic auth to retrieve the jwt token.
	_, resp, err := c.api.AuthenticateUser(ctx, api.AuthUserData{
		Username: username,
		Password: password,
	})
	if err != nil {
		return api.MapAPIError(err, resp)
	}
	defer resp.Body.Close()

	// Set auth token in a new context for re-use.
	token := respAuthToken(resp)
	if token == "" {
		return ErrNoAuthToken
	}

	// Update the client with the new token.
	c.ctx = context.WithValue(context.Background(), api.ContextAccessToken, token)

	return nil
}

// AddToken adds the current authentication token to a given context.
func (c *Client) AddToken(ctx context.Context) context.Context {
	return context.WithValue(ctx, api.ContextAccessToken, c.ctx.Value(api.ContextAccessToken))
}

// respAuthToken is a helper to pull the auth token out of a HTTP Response.
func respAuthToken(resp *http.Response) string {
	if value := resp.Header.Get("Authorization"); value != "" {
		// "Bearer aaaabbbbcccdddeeeff"
		return strings.Split(value, " ")[1]
	}
	return ""
}
