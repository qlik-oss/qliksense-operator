package controller

import (
	"github.com/qlik-oss/qliksense-operator/pkg/controller/qliksense"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, qliksense.Add)
}
