//go:build k8srequired
// +build k8srequired

package appcatalogentry

import "github.com/giantswarm/microerror"

var testError = &microerror.Error{
	Kind: "testError",
}
