package storageos

import (
	"context"
	"net/http"

	api "github.com/storageos/go-api/v2"
)

type fakeReadCloser struct{}

func (f fakeReadCloser) Read(p []byte) (n int, err error) { return 0, nil }
func (f fakeReadCloser) Close() error                     { return nil }

// Fake returns a client that uses a fake ControlPlane api client.
func Fake() *Client {
	return &Client{
		api: fakeControlPlane{},
		ctx: context.TODO(),
	}
}

type fakeControlPlane struct {
}

func (f fakeControlPlane) RefreshJwt(ctx context.Context) (api.UserSession, *http.Response, error) {
	return api.UserSession{}, &http.Response{
		Header: http.Header{
			"Authorization": []string{"Bearer aaaabbbbcccdddeeeff"},
		},
		Body: fakeReadCloser{},
	}, nil
}

func (f fakeControlPlane) AuthenticateUser(ctx context.Context, authUserData api.AuthUserData) (api.UserSession, *http.Response, error) {
	return api.UserSession{}, &http.Response{
		Header: http.Header{
			"Authorization": []string{"Bearer aaaabbbbcccdddeeeff"},
		},
		Body: fakeReadCloser{},
	}, nil
}

func (f fakeControlPlane) GetCluster(ctx context.Context) (api.Cluster, *http.Response, error) {
	return api.Cluster{}, &http.Response{
		Header: http.Header{
			"Authorization": []string{"Bearer aaaabbbbcccdddeeeff"},
		},
		Body: fakeReadCloser{},
	}, nil
}

func (f fakeControlPlane) UpdateCluster(ctx context.Context, updateClusterData api.UpdateClusterData, localVarOptionals *api.UpdateClusterOpts) (api.Cluster, *http.Response, error) {
	return api.Cluster{}, &http.Response{
		Header: http.Header{
			"Authorization": []string{"Bearer aaaabbbbcccdddeeeff"},
		},
		Body: fakeReadCloser{},
	}, nil
}
