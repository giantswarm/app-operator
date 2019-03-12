// +build k8srequired

package ensure

import "github.com/giantswarm/microerror"

var testError = &microerror.Error{
	Kind: "testError",
}
