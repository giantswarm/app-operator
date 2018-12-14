package main

import (
	"fmt"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/microkit/command"
	microserver "github.com/giantswarm/microkit/server"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/viper"

	"github.com/giantswarm/app-operator/flag"
	"github.com/giantswarm/app-operator/server"
	"github.com/giantswarm/app-operator/service"
)

var (
	description = "The app-operator manages apps in Kubernetes clusters."
	f           = flag.New()
	name        = "app-operator"
	gitCommit   = "n/a"
	source      = "https://github.com/giantswarm/app-operator"
)

func main() {
	err := mainWithError()
	if err != nil {
		panic(fmt.Sprintf("%#v\n", err))
	}
}

func mainWithError() (err error) {
	// Create a new logger that is used by all packages.
	var newLogger micrologger.Logger
	{
		c := micrologger.Config{}

		newLogger, err = micrologger.New(c)
		if err != nil {
			return microerror.Maskf(err, "micrologger.New")
		}
	}

	// Define server factory to create the custom server once all command line
	// flags are parsed and all microservice configuration is processed.
	newServerFactory := func(v *viper.Viper) microserver.Server {
		// New custom service implements the business logic.
		var newService *service.Service
		{
			c := service.Config{
				Flag:   f,
				Logger: newLogger,
				Viper:  v,

				Description: description,
				GitCommit:   gitCommit,
				ProjectName: name,
				Source:      source,
			}
			newService, err = service.New(c)
			if err != nil {
				panic(fmt.Sprintf("%#v\n", microerror.Maskf(err, "service.New")))
			}

			go newService.Boot()
		}

		// New custom server that bundles microkit endpoints.
		var newServer microserver.Server
		{
			c := server.Config{
				Logger:      newLogger,
				Service:     newService,
				Viper:       v,
				ProjectName: name,
			}

			newServer, err = server.New(c)
			if err != nil {
				panic(fmt.Sprintf("%#v\n", microerror.Maskf(err, "server.New")))
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

			Description: description,
			GitCommit:   gitCommit,
			Name:        name,
			Source:      source,
		}

		newCommand, err = command.New(c)
		if err != nil {
			return microerror.Maskf(err, "command.New")
		}
	}

	daemonCommand := newCommand.DaemonCommand().CobraCommand()

	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.Address, "", "Address used to connect to Kubernetes. When empty in-cluster config is created.")
	daemonCommand.PersistentFlags().Bool(f.Service.Kubernetes.InCluster, true, "Whether to use the in-cluster config to authenticate with Kubernetes.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.TLS.CAFile, "", "Certificate authority file path to use to authenticate with Kubernetes.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.TLS.CrtFile, "", "Certificate file path to use to authenticate with Kubernetes.")
	daemonCommand.PersistentFlags().String(f.Service.Kubernetes.TLS.KeyFile, "", "Key file path to use to authenticate with Kubernetes.")

	newCommand.CobraCommand().Execute()

	return nil
}
