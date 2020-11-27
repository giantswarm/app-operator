package chart

import "github.com/giantswarm/microerror"

var appDependencyNotReadyError = &microerror.Error{
	Kind: "appDependencyNotReadyError",
}

// IsAppDependencyNotReady asserts appDependencyNotReadyError.
func IsAppDependencyNotReady(err error) bool {
	return microerror.Cause(err) == appDependencyNotReadyError
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var notFoundError = &microerror.Error{
	Kind: "notFoundError",
}

// IsNotFound asserts notFoundError.
func IsNotFound(err error) bool {
	return microerror.Cause(err) == notFoundError
}

var wrongTypeError = &microerror.Error{
	Kind: "wrongTypeError",
}

// IsWrongType asserts wrongTypeError.
func IsWrongType(err error) bool {
	return microerror.Cause(err) == wrongTypeError
}
