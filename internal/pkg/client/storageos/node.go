package storageos

import (
	"context"
	"net/http"

	storageostypes "github.com/storageos/go-api/types"
	storageosapiv2 "github.com/storageos/go-api/v2"

	"github.com/storageos/cluster-operator/internal/pkg/client/storageos/common"
	v2 "github.com/storageos/cluster-operator/internal/pkg/client/storageos/v2"
)

// GetNodeV1 returns StorageOS v1 node.
func (c Client) GetNodeV1(name string) (*storageostypes.Node, error) {
	return c.V1.Node(name)
}

// GetNodeV2 returns StorageOS v2 node.
func (c Client) GetNodeV2(name string) (*storageosapiv2.Node, error) {
	// Get a list of all the nodes.
	nodes, rsp, err := c.V2.DefaultApi.ListNodes(c.Ctx)
	if err != nil {
		return nil, statusCodeBasedError(rsp.StatusCode, err)
	}

	var node storageosapiv2.Node

	found := false
	for _, n := range nodes {
		if n.Name == name {
			node = n
			found = true
			break
		}
	}

	if !found {
		return nil, common.ErrResourceNotFound
	}

	return &node, nil
}

// UpdateNodeV1 updates a StorageOS v1 node with the given node attributes.
func (c Client) UpdateNodeV1(node *storageostypes.Node) error {
	// Create a new context for v1 client.
	ctx, cancel := context.WithTimeout(context.TODO(), common.HTTPTimeout)
	defer cancel()

	opts := storageostypes.NodeUpdateOptions{
		ID:          node.ID,
		Name:        node.Name,
		Description: node.Description,
		Labels:      node.Labels,
		Cordon:      node.Cordon,
		Drain:       node.Drain,
		Context:     ctx,
	}

	_, err := c.V1.NodeUpdate(opts)
	return err
}

// UpdateNodeV2 updates a StorageOS v2 node with the given node attributes.
func (c Client) UpdateNodeV2(node *storageosapiv2.Node) error {
	nodeData := storageosapiv2.UpdateNodeData{
		Labels:  node.Labels,
		Version: node.Version,
	}

	_, rsp, err := c.V2.DefaultApi.UpdateNode(c.Ctx, node.Id, nodeData, nil)
	if err != nil {
		return statusCodeBasedError(rsp.StatusCode, err)
	}

	return nil
}

// statusCodeBasedError returns known errors based on the status code or
// a generic error.
func statusCodeBasedError(statusCode int, err error) error {
	if statusCode == http.StatusUnauthorized {
		return common.ErrUnauthorized
	}

	// Return generic error.
	return v2.GetAPIErrorResponse(err)
}
