module github.com/giantswarm/app-operator/v2

go 1.15

require (
	github.com/CloudyKit/jet v2.1.3-0.20180809161101-62edd43e4f88+incompatible // indirect
	github.com/Joker/jade v1.0.1-0.20190614124447-d475f43051e7 // indirect
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/flosch/pongo2 v0.0.0-20190707114632-bbf5a6c351f4 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/giantswarm/apiextensions/v3 v3.13.1-0.20210104133707-4fe232b93488
	github.com/giantswarm/app/v4 v4.0.0
	github.com/giantswarm/appcatalog v0.3.2
	github.com/giantswarm/backoff v0.2.0
	github.com/giantswarm/errors v0.2.3
	github.com/giantswarm/helmclient/v4 v4.1.0
	github.com/giantswarm/k8sclient/v5 v5.0.0
	github.com/giantswarm/kubeconfig/v4 v4.0.0
	github.com/giantswarm/microendpoint v0.2.0
	github.com/giantswarm/microerror v0.3.0
	github.com/giantswarm/microkit v0.2.2
	github.com/giantswarm/micrologger v0.5.0
	github.com/giantswarm/operatorkit/v4 v4.1.1-0.20210105120921-5c069bded9db
	github.com/giantswarm/to v0.3.0
	github.com/giantswarm/versionbundle v0.2.0
	github.com/go-kit/kit v0.10.0
	github.com/google/go-cmp v0.5.4
	github.com/gorilla/mux v1.8.0
	github.com/iris-contrib/i18n v0.0.0-20171121225848-987a633949d0 // indirect
	github.com/mediocregopher/mediocre-go-lib v0.0.0-20181029021733-cb65787f37ed // indirect
	github.com/prometheus/client_golang v1.9.0
	github.com/spf13/afero v1.5.1
	github.com/spf13/viper v1.7.1
	k8s.io/api v0.19.4
	k8s.io/apimachinery v0.19.4
	k8s.io/client-go v0.19.4
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// Apply fix for CVE-2020-15114 not yet released in github.com/spf13/viper.
	github.com/bketelsen/crypt => github.com/bketelsen/crypt v0.0.3
	// Use moby v20.10.0-beta1 to fix build issue on darwin.
	github.com/docker/docker => github.com/moby/moby v20.10.0-beta1+incompatible
	// Use go-logr/logr v0.1.0 due to breaking changes in v0.2.0 that can't be applied.
	github.com/go-logr/logr v0.2.0 => github.com/go-logr/logr v0.1.0
	// Use mergo 0.3.11 due to bug in 0.3.9 merging Go structs.
	github.com/imdario/mergo => github.com/imdario/mergo v0.3.11
	// Same as go-logr/logr, klog/v2 is using logr v0.2.0
	k8s.io/klog/v2 v2.2.0 => k8s.io/klog/v2 v2.0.0
	// Use fork of CAPI with Kubernetes 1.18 support.
	sigs.k8s.io/cluster-api => github.com/giantswarm/cluster-api v0.3.10-gs
)
