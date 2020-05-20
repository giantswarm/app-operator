package helm

import (
	"github.com/giantswarm/app-operator/flag/service/helm/http"
)

type Helm struct {
	HTTP            http.HTTP
	TillerNamespace string
}
