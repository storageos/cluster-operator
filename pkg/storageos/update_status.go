package storageos

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	api "github.com/storageos/cluster-operator/pkg/apis/cluster/v1alpha1"
	storageosapi "github.com/storageos/go-api"
	"github.com/storageos/go-api/types"
	"k8s.io/api/core/v1"
)

func (s *Deployment) updateStorageOSStatus(status *api.StorageOSStatus) error {
	if reflect.DeepEqual(s.stos.Status, *status) {
		return nil
	}

	// When there's a difference in node ready count, broadcast the health change event.
	if s.stos.Status.Ready != status.Ready {
		// Ready contains the node count in the format 3/3.
		ready := strings.Split(status.Ready, "/")
		if ready[0] == ready[1] {
			if s.recorder != nil {
				s.recorder.Event(s.stos, v1.EventTypeNormal, "ChangedStatus", fmt.Sprintf("%s StorageOS nodes are functional. Cluster healthy", status.Ready))
			}
		} else {
			if s.recorder != nil {
				s.recorder.Event(s.stos, v1.EventTypeWarning, "ChangedStatus", fmt.Sprintf("%s StorageOS nodes are functional", status.Ready))
			}
		}
	}

	s.stos.Status = *status
	return s.client.Update(context.Background(), s.stos)
}

// getStorageOSStatus queries health of all the nodes in the join token and
// returns the cluster status.
func (s *Deployment) getStorageOSStatus() (*api.StorageOSStatus, error) {
	nodeIPs := strings.Split(s.stos.Spec.Join, ",")

	totalNodes := len(nodeIPs)
	readyNodes := 0

	healthStatus := make(map[string]api.NodeHealth)

	for _, node := range nodeIPs {
		if status, err := getNodeHealth(node, 1); err == nil {
			healthStatus[node] = *status
			if isHealthy(status) {
				readyNodes++
			}
		} else {
			log.Printf("failed to get health of node %s: %v", node, err)
		}
	}

	phase := api.ClusterPhaseInitial
	if readyNodes == totalNodes {
		phase = api.ClusterPhaseRunning
	}

	return &api.StorageOSStatus{
		Phase:            phase,
		Nodes:            nodeIPs,
		NodeHealthStatus: healthStatus,
		Ready:            fmt.Sprintf("%d/%d", readyNodes, totalNodes),
	}, nil
}

func isHealthy(health *api.NodeHealth) bool {
	if health.DirectfsInitiator+health.Director+health.KV+health.KVWrite+
		health.Nats+health.Presentation+health.Rdb == strings.Repeat("alive", 7) {
		return true
	}
	return false
}

func getNodeHealth(address string, timeout int) (*api.NodeHealth, error) {
	healthEndpointFormat := "http://%s:%s/v1/" + storageosapi.HealthAPIPrefix

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))
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

	return &api.NodeHealth{
		DirectfsInitiator: healthStatus.Submodules.DirectFSClient.Status,
		Director:          healthStatus.Submodules.Director.Status,
		KV:                healthStatus.Submodules.KV.Status,
		KVWrite:           healthStatus.Submodules.KVWrite.Status,
		Nats:              healthStatus.Submodules.NATS.Status,
		Presentation:      healthStatus.Submodules.FS.Status,
		Rdb:               healthStatus.Submodules.FSDriver.Status,
	}, nil
}
