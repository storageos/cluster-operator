package controller

import (
	"github.com/storageos/cluster-operator/pkg/controller/storageosupgrade"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, storageosupgrade.Add)
}
