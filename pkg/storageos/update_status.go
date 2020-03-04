package storageos

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strings"
	"time"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	storageosapi "github.com/storageos/go-api"
	"github.com/storageos/go-api/types"
	v1 "k8s.io/api/core/v1"
)

var (
	// nodeHealthTimeout specifies how long we should wait for the api to
	// return health results.
	nodeHealthTimeout = time.Second

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

	if s.nodev2 {
		return s.getStorageOSV2Status(nodeIPs)
	}
	return s.getStorageOSV1Status(nodeIPs)
}

// getStorageOSV2Status queries health of all the nodes in the cluster and
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

// getStorageOSV1Status queries health of all the nodes in the join token and
// returns the cluster status.
func (s *Deployment) getStorageOSV1Status(nodeIPs []string) (*storageosv1.StorageOSClusterStatus, error) {

	var readyNodes int

	totalNodes := len(nodeIPs)
	healthStatus := make(map[string]storageosv1.NodeHealth)
	memberStatus := new(storageosv1.MembersStatus)

	for _, node := range nodeIPs {
		if status, err := getNodeHealth(node, nodeHealthTimeout); err == nil {
			healthStatus[node] = *status
			if isHealthy(status) {
				readyNodes++
				memberStatus.Ready = append(memberStatus.Ready, node)
			} else {
				memberStatus.Unready = append(memberStatus.Unready, node)
			}
		}
	}

	phase := storageosv1.ClusterPhaseCreating
	if readyNodes == totalNodes {
		phase = storageosv1.ClusterPhaseRunning
	}

	return &storageosv1.StorageOSClusterStatus{
		Phase:            phase,
		Nodes:            nodeIPs,
		NodeHealthStatus: healthStatus,
		Ready:            fmt.Sprintf("%d/%d", readyNodes, totalNodes),
		Members:          *memberStatus,
	}, nil
}

func isHealthy(health *storageosv1.NodeHealth) bool {
	if health.DirectfsInitiator+health.Director+health.KV+health.KVWrite+
		health.Nats+health.Presentation+health.Rdb == strings.Repeat("alive", 7) {
		return true
	}
	return false
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

func getNodeHealth(address string, timeout time.Duration) (*storageosv1.NodeHealth, error) {
	healthEndpointFormat := "http://%s:%s/v1/" + storageosapi.HealthAPIPrefix

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := &http.Client{}

	var healthStatus types.HealthStatus
	cpURL := fmt.Sprintf(healthEndpointFormat, address, storageosapi.DefaultPort)
	cpReq, err := http.NewRequest("GET", cpURL, nil)
	if err != nil {
		return nil, err
	}

	cpResp, err := client.Do(cpReq.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	if err := json.NewDecoder(cpResp.Body).Decode(&healthStatus); err != nil {
		return nil, err
	}

	return &storageosv1.NodeHealth{
		DirectfsInitiator: healthStatus.Submodules.DirectFSClient.Status,
		Director:          healthStatus.Submodules.Director.Status,
		KV:                healthStatus.Submodules.KV.Status,
		KVWrite:           healthStatus.Submodules.KVWrite.Status,
		Nats:              healthStatus.Submodules.NATS.Status,
		Presentation:      healthStatus.Submodules.FS.Status,
		Rdb:               healthStatus.Submodules.FSDriver.Status,
	}, nil
}
