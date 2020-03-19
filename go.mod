module github.com/giantswarm/app-operator

go 1.13

require (
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/giantswarm/apiextensions v0.1.1
	github.com/giantswarm/appcatalog v0.1.11
	github.com/giantswarm/apprclient v0.0.0-20200304175413-045e7f42fdb3 // indirect
	github.com/giantswarm/backoff v0.0.0-20200209120535-b7cb1852522d
	github.com/giantswarm/e2e-harness v0.1.1-0.20191209134222-be7852f38d3e
	github.com/giantswarm/e2esetup v0.0.0-20191209131007-01b9f9061692
	github.com/giantswarm/e2etemplates v0.0.0-20200309112539-751eac7541c4
	github.com/giantswarm/errors v0.0.0-20200304180000-924f9ee38738
	github.com/giantswarm/exporterkit v0.0.0-20190619131829-9749deade60f
	github.com/giantswarm/helmclient v0.0.0-20200317180111-3fa03f5d7b76
	github.com/giantswarm/k8sclient v0.0.0-20200120104955-1542917096d6
	github.com/giantswarm/kubeconfig v0.0.0-20191209121754-c5784ae65a49
	github.com/giantswarm/microendpoint v0.0.0-20200205204116-c2c5b3af4bdb
	github.com/giantswarm/microerror v0.2.0
	github.com/giantswarm/microkit v0.0.0-20191023091504-429e22e73d3e
	github.com/giantswarm/micrologger v0.2.0
	github.com/giantswarm/operatorkit v0.0.0-20200205163802-6b6e6b2c208b
	github.com/giantswarm/versionbundle v0.0.0-20200205145509-6772c2bc7b34
	github.com/google/go-cmp v0.4.0
	github.com/prometheus/client_golang v1.0.0
	github.com/spf13/afero v1.2.2
	github.com/spf13/viper v1.6.2
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.17.3
	k8s.io/helm v2.16.3+incompatible
)
