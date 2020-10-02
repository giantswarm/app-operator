module github.com/giantswarm/app-operator/v2

go 1.14

require (
	github.com/Masterminds/semver/v3 v3.1.0
	github.com/ghodss/yaml v1.0.0
	github.com/giantswarm/apiextensions/v2 v2.5.2
	github.com/giantswarm/appcatalog v0.2.7
	github.com/giantswarm/backoff v0.2.0
	github.com/giantswarm/errors v0.2.3
	github.com/giantswarm/helmclient/v2 v2.1.3
	github.com/giantswarm/k8sclient/v4 v4.0.0
	github.com/giantswarm/kubeconfig/v2 v2.0.0
	github.com/giantswarm/microendpoint v0.2.0
	github.com/giantswarm/microerror v0.2.1
	github.com/giantswarm/microkit v0.2.2
	github.com/giantswarm/micrologger v0.3.3
	github.com/giantswarm/operatorkit/v2 v2.0.1
	github.com/giantswarm/versionbundle v0.2.0
	github.com/google/go-cmp v0.5.2
	github.com/spf13/afero v1.4.0
	github.com/spf13/viper v1.7.1
	k8s.io/api v0.18.9
	k8s.io/apimachinery v0.18.9
	k8s.io/client-go v0.18.9
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/ghodss/yaml => github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/lib/pq => github.com/lib/pq v1.3.0
	github.com/mailru/easyjson => github.com/mailru/easyjson v0.7.0
	github.com/mattn/go-colorable => github.com/mattn/go-colorable v0.1.2
	github.com/mattn/go-isatty => github.com/mattn/go-isatty v0.0.9
	github.com/mattn/go-runewidth => github.com/mattn/go-runewidth v0.0.4
)
