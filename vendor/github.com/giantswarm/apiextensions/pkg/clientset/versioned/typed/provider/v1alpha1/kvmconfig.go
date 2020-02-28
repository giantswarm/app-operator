/*
Copyright 2020 Giant Swarm GmbH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"time"

	v1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	scheme "github.com/giantswarm/apiextensions/pkg/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// KVMConfigsGetter has a method to return a KVMConfigInterface.
// A group's client should implement this interface.
type KVMConfigsGetter interface {
	KVMConfigs(namespace string) KVMConfigInterface
}

// KVMConfigInterface has methods to work with KVMConfig resources.
type KVMConfigInterface interface {
	Create(*v1alpha1.KVMConfig) (*v1alpha1.KVMConfig, error)
	Update(*v1alpha1.KVMConfig) (*v1alpha1.KVMConfig, error)
	UpdateStatus(*v1alpha1.KVMConfig) (*v1alpha1.KVMConfig, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.KVMConfig, error)
	List(opts v1.ListOptions) (*v1alpha1.KVMConfigList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.KVMConfig, err error)
	KVMConfigExpansion
}

// kVMConfigs implements KVMConfigInterface
type kVMConfigs struct {
	client rest.Interface
	ns     string
}

// newKVMConfigs returns a KVMConfigs
func newKVMConfigs(c *ProviderV1alpha1Client, namespace string) *kVMConfigs {
	return &kVMConfigs{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the kVMConfig, and returns the corresponding kVMConfig object, and an error if there is any.
func (c *kVMConfigs) Get(name string, options v1.GetOptions) (result *v1alpha1.KVMConfig, err error) {
	result = &v1alpha1.KVMConfig{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("kvmconfigs").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of KVMConfigs that match those selectors.
func (c *kVMConfigs) List(opts v1.ListOptions) (result *v1alpha1.KVMConfigList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.KVMConfigList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("kvmconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested kVMConfigs.
func (c *kVMConfigs) Watch(opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("kvmconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a kVMConfig and creates it.  Returns the server's representation of the kVMConfig, and an error, if there is any.
func (c *kVMConfigs) Create(kVMConfig *v1alpha1.KVMConfig) (result *v1alpha1.KVMConfig, err error) {
	result = &v1alpha1.KVMConfig{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("kvmconfigs").
		Body(kVMConfig).
		Do().
		Into(result)
	return
}

// Update takes the representation of a kVMConfig and updates it. Returns the server's representation of the kVMConfig, and an error, if there is any.
func (c *kVMConfigs) Update(kVMConfig *v1alpha1.KVMConfig) (result *v1alpha1.KVMConfig, err error) {
	result = &v1alpha1.KVMConfig{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("kvmconfigs").
		Name(kVMConfig.Name).
		Body(kVMConfig).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *kVMConfigs) UpdateStatus(kVMConfig *v1alpha1.KVMConfig) (result *v1alpha1.KVMConfig, err error) {
	result = &v1alpha1.KVMConfig{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("kvmconfigs").
		Name(kVMConfig.Name).
		SubResource("status").
		Body(kVMConfig).
		Do().
		Into(result)
	return
}

// Delete takes name of the kVMConfig and deletes it. Returns an error if one occurs.
func (c *kVMConfigs) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("kvmconfigs").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *kVMConfigs) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("kvmconfigs").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched kVMConfig.
func (c *kVMConfigs) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.KVMConfig, err error) {
	result = &v1alpha1.KVMConfig{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("kvmconfigs").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
