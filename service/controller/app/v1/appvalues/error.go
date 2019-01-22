package appvalues

import "github.com/giantswarm/microerror"

var invalidExecutionError = &microerror.Error{
	Kind: "invalidExecutionError",
}

// IsInvalidExecution asserts invalidExecutionError.
func IsInvalidExecution(err error) bool {
	return microerror.Cause(err) == invalidExecutionError
}
