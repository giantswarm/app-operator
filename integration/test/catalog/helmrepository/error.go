//go:build k8srequired
// +build k8srequired

package helmrepository

import "github.com/giantswarm/microerror"

var testError = &microerror.Error{
	Kind: "testError",
}
