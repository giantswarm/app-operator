package chartcrd

import (
	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var notEstablishedError = &microerror.Error{
	Kind: "notEstablishedError",
}

// IsNotEstablished asserts notEstablishedError.
func IsNotEstablished(err error) bool {
	return microerror.Cause(err) == notEstablishedError
}
