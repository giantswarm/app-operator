package status

import (
	"strings"
)

const (
	helmSchemaValidationErrorMsg     = "values don't meet the specifications of the schema(s) in the following chart(s)"
	invalidManifestErrorMsg          = "unable to build kubernetes objects from release manifest"
	resourceAlreadyExistsErrorMsg    = "rendered manifests contain a resource that already exists"
	releaseNameInvalidErrorMsgPrefix = "invalid release name, must match regex"
	releaseNameInvalidErrorMsgSuffix = "and the length must not be longer than 53"
	validationFailedErrorMsg         = "error validating data"
)

func isHelmSchemaValidation(m string) bool {
	return strings.Contains(m, helmSchemaValidationErrorMsg)
}

func isInvalidManifest(m string) bool {
	return strings.Contains(m, invalidManifestErrorMsg)
}

func isResourceAlreadyExists(m string) bool {
	return strings.Contains(m, resourceAlreadyExistsErrorMsg)
}

func isReleaseNameInvalid(m string) bool {
	return strings.Contains(m, releaseNameInvalidErrorMsgPrefix) && strings.Contains(m, releaseNameInvalidErrorMsgSuffix)
}

func isValidationFailedErrorMsg(m string) bool {
	return strings.Contains(m, validationFailedErrorMsg) && !strings.Contains(m, invalidManifestErrorMsg)
}
