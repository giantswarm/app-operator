module github.com/giantswarm/app-operator

go 1.14

require (
	github.com/Masterminds/semver/v3 v3.1.0
	github.com/giantswarm/apiextensions v0.4.17
	github.com/giantswarm/appcatalog v0.2.7
	github.com/giantswarm/backoff v0.2.0
	github.com/giantswarm/errors v0.2.3
	github.com/giantswarm/exporterkit v0.2.0
	github.com/giantswarm/helmclient v1.0.5
	github.com/giantswarm/k8sclient/v3 v3.1.2
	github.com/giantswarm/kubeconfig v0.2.1
	github.com/giantswarm/microendpoint v0.2.0
	github.com/giantswarm/microerror v0.2.1
	github.com/giantswarm/microkit v0.2.1
	github.com/giantswarm/micrologger v0.3.1
	github.com/giantswarm/operatorkit v1.2.0
	github.com/giantswarm/versionbundle v0.2.0
	github.com/google/go-cmp v0.5.1
	github.com/prometheus/client_golang v1.7.1
	github.com/spf13/afero v1.3.3
	github.com/spf13/viper v1.7.0
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.17.3
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/ghodss/yaml => github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/mailru/easyjson => github.com/mailru/easyjson v0.7.0
	github.com/mattn/go-colorable => github.com/mattn/go-colorable v0.1.2
	github.com/mattn/go-isatty => github.com/mattn/go-isatty v0.0.9
)
