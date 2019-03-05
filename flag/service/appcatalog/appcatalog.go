package appcatalog

import (
	"github.com/giantswarm/app-operator/flag/service/appcatalog/index"
)

// AppCatalog is a data structure to hold AppCatalog specific command line configuration flags.
type AppCatalog struct {
	Index index.Index
}
