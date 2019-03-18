package storageoscluster

// Deployment is an interface for deployment of a cluster.
type Deployment interface {
	// Deploy deploys a cluster.
	Deploy() error
	// Delete deletes a deployed cluster.
	Delete() error
}
