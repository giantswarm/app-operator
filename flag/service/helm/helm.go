package helm

import (
	"github.com/giantswarm/app-operator/v4/flag/service/helm/http"
)

type Helm struct {
	HTTP            http.HTTP
	TillerNamespace string
}
