package values

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

var parsingError = &microerror.Error{
	Kind: "parsingError",
}

// IsParsingError asserts parsingError.
func IsParsingError(err error) bool {
	return microerror.Cause(err) == parsingError
}
