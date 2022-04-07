module github.com/giantswarm/app-operator/v5

go 1.16

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/ghodss/yaml v1.0.0
	github.com/giantswarm/apiextensions-application v0.3.1
	github.com/giantswarm/app/v6 v6.9.0
	github.com/giantswarm/appcatalog v0.6.0
	github.com/giantswarm/apptest v1.1.1
	github.com/giantswarm/backoff v1.0.0
	github.com/giantswarm/errors v0.3.0
	github.com/giantswarm/helmclient/v4 v4.10.0
	github.com/giantswarm/k8sclient/v6 v6.1.0
	github.com/giantswarm/k8smetadata v0.9.3
	github.com/giantswarm/kubeconfig/v4 v4.1.0
	github.com/giantswarm/microendpoint v1.0.0
	github.com/giantswarm/microerror v0.4.0
	github.com/giantswarm/microkit v1.0.0
	github.com/giantswarm/micrologger v0.6.0
	github.com/giantswarm/operatorkit/v6 v6.1.0
	github.com/giantswarm/to v0.4.0
	github.com/google/go-cmp v0.5.7
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/prometheus/client_golang v1.12.1
	github.com/spf13/afero v1.8.0
	github.com/spf13/viper v1.10.1
	k8s.io/api v0.23.5
	k8s.io/apiextensions-apiserver v0.23.5
	k8s.io/apimachinery v0.23.5
	k8s.io/client-go v0.23.5
	sigs.k8s.io/controller-runtime v0.11.2
	sigs.k8s.io/yaml v1.3.0
)

replace (
	github.com/Microsoft/hcsshim v0.8.7 => github.com/Microsoft/hcsshim v0.8.10
	github.com/bketelsen/crypt => github.com/bketelsen/crypt v0.0.3
	github.com/coreos/etcd v3.3.10+incompatible => github.com/coreos/etcd v3.3.25+incompatible
	github.com/dgrijalva/jwt-go => github.com/dgrijalva/jwt-go/v4 v4.0.0-preview1
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	// Use moby v20.10.0-beta1 to fix build issue on darwin.
	github.com/docker/docker => github.com/moby/moby v20.10.9+incompatible
	// Use go-logr/logr v0.1.0 due to breaking changes in v0.2.0 that can't be applied.
	github.com/go-logr/logr v0.2.0 => github.com/go-logr/logr v0.1.0
	github.com/gogo/protobuf v1.3.1 => github.com/gogo/protobuf v1.3.2
	// Use mergo 0.3.11 due to bug in 0.3.9 merging Go structs.
	github.com/imdario/mergo => github.com/imdario/mergo v0.3.11
	github.com/opencontainers/runc v0.1.1 => github.com/opencontainers/runc v1.0.0-rc7
	github.com/ulikunitz/xz => github.com/ulikunitz/xz v0.5.10
	// Same as go-logr/logr, klog/v2 is using logr v0.2.0
	k8s.io/klog/v2 v2.2.0 => k8s.io/klog/v2 v2.0.0
)
