package service

import (
	"testing"

	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/spf13/viper"

	"github.com/giantswarm/app-operator/flag"
)

func Test_Service_New(t *testing.T) {
	testCases := []struct {
		name         string
		config       func() Config
		errorMatcher func(error) bool
	}{
		{
			name: "case 0: valid config returns no error",
			config: func() Config {
				c := Config{
					Flag:   flag.New(),
					Logger: microloggertest.New(),
					Viper:  viper.New(),

					Description: "test",
					GitCommit:   "test",
					ProjectName: "chart-operator",
					Source:      "test",
				}

				c.Viper.Set(c.Flag.Service.Chart.Namespace, "giantswarm")
				c.Viper.Set(c.Flag.Service.AppCatalog.Index.Namespace, "giantswarm")
				c.Viper.Set(c.Flag.Service.Kubernetes.Address, "kubernetes")
				c.Viper.Set(c.Flag.Service.Kubernetes.InCluster, false)
				c.Viper.Set(c.Flag.Service.Kubernetes.Watch.Namespace, "giantswarm")

				return c
			},
			errorMatcher: nil,
		},
		{
			name: "case 1: invalid config returns error",
			config: func() Config {
				c := Config{
					Flag:  flag.New(),
					Viper: viper.New(),
				}

				return c
			},
			errorMatcher: IsInvalidConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(tc.config())

			switch {
			case err != nil && tc.errorMatcher == nil:
				t.Fatalf("error == %#v, want nil", err)
			case err == nil && tc.errorMatcher != nil:
				t.Fatalf("error == nil, want non-nil")
			case tc.errorMatcher != nil && !tc.errorMatcher(err):
				t.Fatalf("error == %#v, want matching", err)
			}
		})
	}
}
