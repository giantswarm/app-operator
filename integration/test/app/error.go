// +build k8srequired

package app

import "github.com/giantswarm/microerror"

var testError = &microerror.Error{
	Kind: "testError",
}
