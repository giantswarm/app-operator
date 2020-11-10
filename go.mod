module github.com/giantswarm/app-operator/v2

go 1.15

require (
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/semver/v3 v3.1.0
	github.com/ghodss/yaml v1.0.0
	github.com/giantswarm/apiextensions/v3 v3.6.0
	github.com/giantswarm/app/v3 v3.1.1-0.20201110085730-32719a4bda8f
	github.com/giantswarm/appcatalog v0.2.7
	github.com/giantswarm/backoff v0.2.0
	github.com/giantswarm/errors v0.2.3
	github.com/giantswarm/helmclient/v3 v3.0.1
	github.com/giantswarm/k8sclient/v5 v5.0.0
	github.com/giantswarm/kubeconfig/v3 v3.0.0
	github.com/giantswarm/microendpoint v0.2.0
	github.com/giantswarm/microerror v0.2.1
	github.com/giantswarm/microkit v0.2.2
	github.com/giantswarm/micrologger v0.3.4
	github.com/giantswarm/operatorkit/v4 v4.0.0
	github.com/giantswarm/to v0.3.0
	github.com/giantswarm/versionbundle v0.2.0
	github.com/go-kit/kit v0.10.0
	github.com/google/go-cmp v0.5.2
	github.com/gorilla/mux v1.8.0
	github.com/spf13/afero v1.4.1
	github.com/spf13/viper v1.7.1
	k8s.io/api v0.18.9
	k8s.io/apimachinery v0.18.9
	k8s.io/client-go v0.18.9
	sigs.k8s.io/yaml v1.2.0
)

replace (
	// Apply fix for CVE-2020-15114 not yet released in github.com/spf13/viper.
	github.com/bketelsen/crypt => github.com/bketelsen/crypt v0.0.3
	// Use moby v20.10.0-beta1 to fix build issue on darwin.
	github.com/docker/docker => github.com/moby/moby v20.10.0-beta1+incompatible
	// Use mergo 0.3.11 due to bug in 0.3.9 merging Go structs.
	github.com/imdario/mergo => github.com/imdario/mergo v0.3.11
	// Use fork of CAPI with Kubernetes 1.18 support.
	sigs.k8s.io/cluster-api => github.com/giantswarm/cluster-api v0.3.10-gs
)
