// +build k8srequired

package app

import "github.com/giantswarm/microerror"

var notMatching = &microerror.Error{
	Kind: "notMatching",
}

// IsNotMatching asserts notMatching.
func IsNotMatching(err error) bool {
	return microerror.Cause(err) == notMatching
}

var notDeleted = &microerror.Error{
	Kind: "notDeleted",
}

// IsNotDeleted asserts notMatching.
func IsNotDeleted(err error) bool {
	return microerror.Cause(err) == notDeleted
}
