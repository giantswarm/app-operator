package chart

import "github.com/giantswarm/microerror"

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var catalogEmptyError = &microerror.Error{
	Kind: "catalogEmptyError",
}

var appNotFoundError = &microerror.Error{
	Kind: "appNotFoundError",
}

var appVersionNotFoundError = &microerror.Error{
	Kind: "appVersionNotFoundError",
}

// IsNotFound asserts:
// appVersionNotFoundError OR appNotFoundError OR catalogEmptyError.
func IsNotFound(err error) bool {
	return microerror.Cause(err) == appVersionNotFoundError ||
		microerror.Cause(err) == appNotFoundError ||
		microerror.Cause(err) == catalogEmptyError
}

var wrongTypeError = &microerror.Error{
	Kind: "wrongTypeError",
}

// IsWrongType asserts wrongTypeError.
func IsWrongType(err error) bool {
	return microerror.Cause(err) == wrongTypeError
}
