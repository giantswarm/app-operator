package chartoperator

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var notReadyError = &microerror.Error{
	Kind: "notReadyError",
}

// IsNotReady asserts notReadyError.
func IsNotReady(err error) bool {
	return microerror.Cause(err) == notReadyError
}
