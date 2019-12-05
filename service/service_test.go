package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/giantswarm/micrologger/microloggertest"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"

	"github.com/giantswarm/app-operator/flag"
)

func Test_Service_New(t *testing.T) {
	// fake server to initialize helm client
	// there are two calls to this server during initialization,
	// getting the name of tiller pod and port forwarding to it
	h := func(w http.ResponseWriter, r *http.Request) {
		podList := v1.PodList{
			Items: []v1.Pod{},
		}
		pods, err := json.Marshal(podList)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(pods)
	}
	ts := httptest.NewServer(http.HandlerFunc(h))
	defer ts.Close()

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

					Viper: viper.New(),
				}

				c.Viper.Set(c.Flag.Service.Chart.Namespace, "giantswarm")
				c.Viper.Set(c.Flag.Service.Image.Registry, "quay.io")
				c.Viper.Set(c.Flag.Service.Kubernetes.Address, ts.URL)
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
