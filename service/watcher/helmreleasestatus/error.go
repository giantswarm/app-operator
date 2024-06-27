package helmreleasestatus

import (
	"strings"

	"github.com/giantswarm/microerror"
)

var invalidConfigError = &microerror.Error{
	Kind: "invalidConfigError",
}

// IsInvalidConfig asserts invalidConfigError.
func IsInvalidConfig(err error) bool {
	return microerror.Cause(err) == invalidConfigError
}

var resourceNotFoundError = &microerror.Error{
	Kind: "resourceNotFoundError",
}

// IsResourceNotFound asserts resource not found error from the Kubernetes API.
func IsResourceNotFound(err error) bool {
	if err == nil {
		return false
	}

	c := microerror.Cause(err)

	if c == resourceNotFoundError {
		return true
	}
	if strings.Contains(c.Error(), "the server could not find the requested resource") {
		return true
	}

	return false
}
