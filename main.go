package main

import (
	"context"
	"fmt"
	applicationv1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"
	"github.com/giantswarm/k8sclient/v4/pkg/k8sclient"
	"github.com/giantswarm/k8sclient/v4/pkg/k8srestconfig"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/microkit/command"
	microserver "github.com/giantswarm/microkit/server"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/viper"
	"k8s.io/client-go/rest"

	"github.com/giantswarm/app-operator/v2/flag"
	"github.com/giantswarm/app-operator/v2/pkg/project"
	"github.com/giantswarm/app-operator/v2/server"
	"github.com/giantswarm/app-operator/v2/service"
)

var (
	f = flag.New()
)

func main() {
	err := mainWithError()
	if err != nil {
		panic(fmt.Sprintf("%#v\n", err))
	}
}

func mainWithError() (err error) {
	ctx := context.Background()

	// Create a new logger that is used by all packages.
	var newLogger micrologger.Logger
	{
		c := micrologger.Config{}

		newLogger, err = micrologger.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Define server factory to create the custom server once all command line
	// flags are parsed and all microservice configuration is processed.
	newServerFactory := func(v *viper.Viper) microserver.Server {
		var restConfig *rest.Config
		{
			c := k8srestconfig.Config{
				Logger: newLogger,

				Address:    v.GetString(f.Service.Kubernetes.Address),
				InCluster:  v.GetBool(f.Service.Kubernetes.InCluster),
				KubeConfig: v.GetString(f.Service.Kubernetes.KubeConfig),
				TLS: k8srestconfig.ConfigTLS{
					CAFile:  v.GetString(f.Service.Kubernetes.TLS.CAFile),
					CrtFile: v.GetString(f.Service.Kubernetes.TLS.CrtFile),
					KeyFile: v.GetString(f.Service.Kubernetes.TLS.KeyFile),
				},
			}

			restConfig, err = k8srestconfig.New(c)
			if err != nil {
				panic(err)
			}
		}

		var k8sClient k8sclient.Interface
		{
			c := k8sclient.ClientsConfig{
				Logger: newLogger,
				SchemeBuilder: k8sclient.SchemeBuilder{
					applicationv1alpha1.AddToScheme,
				},

				RestConfig: restConfig,
			}

			k8sClient, err = k8sclient.NewClients(c)
			if err != nil {
				panic(err)
			}
		}
		// New custom service implements the business logic.
		var newService *service.Service
		{
			c := service.Config{
				Logger: newLogger,

				Flag:      f,
				Viper:     v,
				K8sClient: k8sClient,
			}
			newService, err = service.New(c)
			if err != nil {
				panic(fmt.Sprintf("%#v\n", microerror.Mask(err)))
			}

			go newService.Boot(ctx)
		}

		// New custom server that bundles microkit endpoints.
		var newServer microserver.Server
		{
			c := server.Config{
				Logger:  newLogger,
				Service: newService,

				Viper:     v,
				K8sClient: k8sClient,
			}

			newServer, err = server.New(c)
			if err != nil {
				panic(fmt.Sprintf("%#v\n", microerror.Mask(err)))
			}
		}

		return newServer
	}

	// Create a new microkit command that manages operator daemon.
	var newCommand command.Command
	{
		c := command.Config{
			Logger:        newLogger,
			ServerFactory: newServerFactory,

			Description: project.Description(),
			GitCommit:   project.GitSHA(),
			Name:        project.Name(),
			Source:      project.Source(),
			Version:     project.Version(),
		}

		newCommand, err = command.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	daemonCommand := newCommand.DaemonCommand().CobraCommand()

	daemonCommand.PersistentFlags().Bool(f.Service.App.Unique, false, "Whether the operator is deployed as a unique app.")
	daemonCommand.PersistentFlags().String(f.Service.Chart.Namespace, "giantswarm", "The namespace where chart CRs are located.")
	daemonCommand.PersistentFlags().String(f.Service.Helm.HTTP.ClientTimeout, "5s", "HTTP timeout for pulling chart tarballs.")
	daemonCommand.PersistentFlags().String(f.Service.Image.Registry, "quay.io", "The container registry for pulling Tiller images.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.Address, "", "Address used to connect to Kubernetes. When empty in-cluster config is created.")
	daemonCommand.PersistentFlags().Bool(f.Service.Kubernetes.InCluster, true, "Whether to use the in-cluster config to authenticate with Kubernetes.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.KubeConfig, "", "KubeConfig used to connect to Kubernetes. When empty other settings are used.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.TLS.CAFile, "", "Certificate authority file path to use to authenticate with Kubernetes.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.TLS.CrtFile, "", "Certificate file path to use to authenticate with Kubernetes.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.TLS.KeyFile, "", "Key file path to use to authenticate with Kubernetes.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.Watch.Namespace, "default", "The namespace where appcatalog and app CRs are located.")

	err = newCommand.CobraCommand().Execute()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
