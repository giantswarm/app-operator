package index

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var wrongTypeError = &microerror.Error{
	Kind: "wrongTypeError",
}

// IsWrongType asserts wrongTypeError.
func IsWrongType(err error) bool {
	return microerror.Cause(err) == wrongTypeError
}

var indexNotFound = &microerror.Error{
	Kind: "indexNotFound",
}

// IsIndexNotFound asserts indexNotFound.
func IsIndexNotFound(err error) bool {
	return microerror.Cause(err) == indexNotFound
}
