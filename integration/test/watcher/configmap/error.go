// +build k8srequired

package configmap

import "github.com/giantswarm/microerror"

var notFoundError = &microerror.Error{
	Kind: "notFoundError",
}

var testError = &microerror.Error{
	Kind: "testError",
}
