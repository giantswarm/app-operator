package key

import (
	"fmt"

	"github.com/giantswarm/app-operator/v2/pkg/project"
)

func AppCatalogEntryManagedBy() string {
	return fmt.Sprintf("%s-unique", project.Name())
}
