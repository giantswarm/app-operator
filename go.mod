module github.com/giantswarm/app-operator/v6

go 1.16

require (
	github.com/BurntSushi/toml v1.1.0 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/Masterminds/squirrel v1.5.3 // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/cenkalti/backoff/v4 v4.1.3 // indirect
	github.com/containerd/cgroups v1.0.4 // indirect
	github.com/containerd/containerd v1.6.6 // indirect
	github.com/docker/cli v20.10.16+incompatible // indirect
	github.com/docker/docker v20.10.16+incompatible // indirect
	github.com/emicklei/go-restful v2.15.0+incompatible // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20210407135951-1de76d718b3f // indirect
	github.com/getsentry/sentry-go v0.13.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/giantswarm/apiextensions-application v0.5.1
	github.com/giantswarm/app/v6 v6.11.2-0.20220707134646-f3c3d631252b
	github.com/giantswarm/appcatalog v0.8.0
	github.com/giantswarm/apptest v1.2.0
	github.com/giantswarm/backoff v1.0.0
	github.com/giantswarm/errors v0.3.0
	github.com/giantswarm/exporterkit v1.0.0 // indirect
	github.com/giantswarm/helmclient/v4 v4.10.1
	github.com/giantswarm/k8sclient/v6 v6.1.0
	github.com/giantswarm/k8smetadata v0.11.1
	github.com/giantswarm/kubeconfig/v4 v4.1.0
	github.com/giantswarm/microendpoint v1.0.0
	github.com/giantswarm/microerror v0.4.0
	github.com/giantswarm/microkit v1.0.0
	github.com/giantswarm/micrologger v1.0.0
	github.com/giantswarm/operatorkit/v6 v6.1.0
	github.com/giantswarm/to v0.4.0
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/swag v0.21.1 // indirect
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/go-cmp v0.5.8
	github.com/google/uuid v1.3.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/jmoiron/sqlx v1.3.5 // indirect
	github.com/klauspost/compress v1.15.6 // indirect
	github.com/lib/pq v1.10.6 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/moby/sys/mountinfo v0.6.2 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.56.3
	github.com/prometheus/client_golang v1.12.2
	github.com/prometheus/common v0.34.0 // indirect
	github.com/russross/blackfriday v1.6.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/spf13/afero v1.8.2
	github.com/spf13/viper v1.12.0
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xlab/treeprint v1.1.0 // indirect
	go.starlark.net v0.0.0-20220328144851-d1966c6b9fcd // indirect
	golang.org/x/crypto v0.0.0-20220525230936-793ad666bf5e // indirect
	golang.org/x/oauth2 v0.0.0-20220524215830-622c5d57e401 // indirect
	golang.org/x/sync v0.0.0-20220601150217-0de741cfad7f // indirect
	golang.org/x/term v0.0.0-20220526004731-065cf7ba2467 // indirect
	golang.org/x/time v0.0.0-20220411224347-583f2d630306 // indirect
	google.golang.org/genproto v0.0.0-20220607223854-30acc4cbd2aa // indirect
	gopkg.in/ini.v1 v1.66.5 // indirect
	helm.sh/helm/v3 v3.9.0 // indirect
	k8s.io/api v0.24.1
	k8s.io/apiextensions-apiserver v0.24.1
	k8s.io/apimachinery v0.24.1
	k8s.io/client-go v0.24.1
	k8s.io/kube-openapi v0.0.0-20220413171646-5e7f5fdc6da6 // indirect
	k8s.io/kubectl v0.24.1 // indirect
	oras.land/oras-go v1.1.1 // indirect
	sigs.k8s.io/controller-runtime v0.12.1
	sigs.k8s.io/json v0.0.0-20220525155127-227cbc7cc124 // indirect
	sigs.k8s.io/kustomize/api v0.11.5 // indirect
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
	github.com/microcosm-cc/bluemonday v1.0.2 => github.com/microcosm-cc/bluemonday v1.0.18
	github.com/nats-io/nats-server/v2 v2.5.0 => github.com/nats-io/nats-server/v2 v2.8.4
	github.com/opencontainers/runc v0.1.1 => github.com/opencontainers/runc v1.0.0-rc7
	github.com/opencontainers/runc v1.1.1 => github.com/opencontainers/runc v1.1.2
	github.com/ulikunitz/xz => github.com/ulikunitz/xz v0.5.10
	github.com/valyala/fasthttp v1.6.0 => github.com/valyala/fasthttp v1.37.0
	go.mongodb.org/mongo-driver v1.1.2 => go.mongodb.org/mongo-driver v1.9.1
	// Same as go-logr/logr, klog/v2 is using logr v0.2.0
	k8s.io/klog/v2 v2.2.0 => k8s.io/klog/v2 v2.0.0
)
