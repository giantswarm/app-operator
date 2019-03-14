package kubeconfig

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfigError asserts invalidConfigError.
func IsInvalidConfigError(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var notFoundError = &microerror.Error{
	Kind: "missingKubeConfigError",
}

// IsNotFoundError asserts notFoundError.
func IsNotFoundError(err error) bool {
	return microerror.Cause(err) == notFoundError
}

var executionFailedError = &microerror.Error{
	Kind: "executionError",
}

// IsExecutionError asserts executionError.
func IsExecutionError(err error) bool {
	return microerror.Cause(err) == executionFailedError
}
