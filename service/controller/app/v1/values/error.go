package values

import "github.com/giantswarm/microerror"

var failedExecutionError = &microerror.Error{
	Kind: "failedExecutionError",
}

// IsFailedExecution asserts failedExecutionError.
func IsFailedExecution(err error) bool {
	return microerror.Cause(err) == failedExecutionError
}
