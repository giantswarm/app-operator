package status

import "github.com/giantswarm/microerror"

var decodeFailedError = &microerror.Error{
	Kind: "decodeFailedError",
}

// IsDecodeFailed asserts deleteFailedError.
func IsDecodeFailed(err error) bool {
	return microerror.Cause(err) == decodeFailedError
}

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var wrongTokenError = &microerror.Error{
	Kind: "wrongTokenError",
}

// IsWrongTokenError asserts wrongTokenError.
func IsWrongTokenError(err error) bool {
	return microerror.Cause(err) == wrongTokenError
}
