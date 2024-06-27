//go:build k8srequired
// +build k8srequired

package basic

import "github.com/giantswarm/microerror"

var testError = &microerror.Error{
	Kind: "testError",
}
