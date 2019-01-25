package kubeconfigtest

import (
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned/fake"
	"github.com/giantswarm/micrologger/microloggertest"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/giantswarm/app-operator/service/controller/app/v1/kubeconfig"
)

func New(g8sClient versioned.Interface) (*kubeconfig.KubeConfig, error) {
	if g8sClient == nil {
		g8sClient = fake.NewSimpleClientset()
	}
	c := kubeconfig.Config{
		G8sClient: g8sClient,
		K8sClient: k8sfake.NewSimpleClientset(),
		Logger:    microloggertest.New(),
	}

	kc, err := kubeconfig.New(c)
	if err != nil {
		return nil, err
	}

	return kc, nil
}
