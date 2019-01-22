package chart

import (
	"context"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

type FakeKubeConfig struct {
	g8sClient versioned.Interface
	k8sClient kubernetes.Interface
}

func (f FakeKubeConfig) NewG8sClientFromSecret(ctx context.Context, secretName, secretNamespace string) (versioned.Interface, error) {
	return f.g8sClient, nil
}

func (f FakeKubeConfig) NewK8sClientFromSecret(ctx context.Context, secretName, secretNamespace string) (kubernetes.Interface, error) {
	return f.k8sClient, nil
}
