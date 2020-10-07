package storageos

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strings"
	"time"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	storageosapi "github.com/storageos/go-api"
	v1 "k8s.io/api/core/v1"
)

var (
	// nodeLivenessTimeout specifies how long we should wait for a connection to
	// the node's api port.
	nodeLivenessTimeout = time.Second
)

func (s *Deployment) updateStorageOSStatus(status *storageosv1.StorageOSClusterStatus) error {
	if reflect.DeepEqual(s.stos.Status, *status) {
		return nil
	}

	// When there's a difference in node ready count, broadcast the health change event.
	if s.stos.Status.Ready != status.Ready {
		// Ready contains the node count in the format 3/3.
		ready := strings.Split(status.Ready, "/")

		// If the ready/total counts are equal and not zero, the cluster is
		// healthy. Else, not ready. 0/0 is an unready cluster.
		if ready[0] == ready[1] && ready[0] != "0" {
			if s.recorder != nil {
				s.recorder.Event(s.stos, v1.EventTypeNormal, "ChangedStatus", fmt.Sprintf("%s StorageOS nodes are functional. Cluster healthy", status.Ready))
			}
		} else {
			if s.recorder != nil {
				s.recorder.Event(s.stos, v1.EventTypeWarning, "ChangedStatus", fmt.Sprintf("%s StorageOS nodes are functional", status.Ready))
			}
		}
	}

	// Update subresource status.
	s.stos.Status = *status
	return s.client.Status().Update(context.Background(), s.stos)
}

func (s *Deployment) getStorageOSStatus() (*storageosv1.StorageOSClusterStatus, error) {
	// Create an empty array because it's used to create cluster status. An
	// uninitialized array results in error at cluster status validation.
	// error: status.nodes in body must be of type array: "null"
	nodeIPs := []string{}

	// Everything is empty if join token is empty.
	if len(s.stos.Spec.Join) > 0 {
		nodeIPs = strings.Split(s.stos.Spec.Join, ",")
	}

	return s.getStorageOSV2Status(nodeIPs)
}

// getStorageOSStatus queries health of all the nodes in the cluster and
// returns the cluster status.
//
// NodeHealthStatus is deprecated and not set for V2.
func (s *Deployment) getStorageOSV2Status(nodeIPs []string) (*storageosv1.StorageOSClusterStatus, error) {
	var readyNodes int

	totalNodes := len(nodeIPs)
	memberStatus := new(storageosv1.MembersStatus)

	for _, ip := range nodeIPs {
		if isListening(ip, storageosapi.DefaultPort, nodeLivenessTimeout) {
			readyNodes++
			memberStatus.Ready = append(memberStatus.Ready, ip)
		} else {
			memberStatus.Unready = append(memberStatus.Unready, ip)
		}
	}

	phase := storageosv1.ClusterPhaseCreating
	if readyNodes == totalNodes {
		phase = storageosv1.ClusterPhaseRunning
	}

	return &storageosv1.StorageOSClusterStatus{
		Phase:            phase,
		Nodes:            nodeIPs,
		NodeHealthStatus: make(map[string]storageosv1.NodeHealth),
		Ready:            fmt.Sprintf("%d/%d", readyNodes, totalNodes),
		Members:          *memberStatus,
	}, nil
}

func isListening(host string, port string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
	if err != nil {
		return false
	}
	if conn != nil {
		defer conn.Close()
	}
	return true
}
