package controller

import (
	"github.com/k8s-kanary/kanary/pkg/controller/kanarystatefulset"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, kanarystatefulset.Add)
}
