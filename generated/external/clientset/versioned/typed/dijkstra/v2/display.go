/*
Copyright (C) 2024 JinLi Co.,Ltd. All rights reserved.

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

package v2

import (
	"context"
	"time"

	v2 "jinli.io/crdshortestpath/api/dijkstra/v2"
	scheme "jinli.io/crdshortestpath/generated/external/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// DisplaysGetter has a method to return a DisplayInterface.
// A group's client should implement this interface.
type DisplaysGetter interface {
	Displays(namespace string) DisplayInterface
}

// DisplayInterface has methods to work with Display resources.
type DisplayInterface interface {
	Create(ctx context.Context, display *v2.Display, opts v1.CreateOptions) (*v2.Display, error)
	Update(ctx context.Context, display *v2.Display, opts v1.UpdateOptions) (*v2.Display, error)
	UpdateStatus(ctx context.Context, display *v2.Display, opts v1.UpdateOptions) (*v2.Display, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v2.Display, error)
	List(ctx context.Context, opts v1.ListOptions) (*v2.DisplayList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v2.Display, err error)
	DisplayExpansion
}

// displays implements DisplayInterface
type displays struct {
	client rest.Interface
	ns     string
}

// newDisplays returns a Displays
func newDisplays(c *DijkstraV2Client, namespace string) *displays {
	return &displays{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the display, and returns the corresponding display object, and an error if there is any.
func (c *displays) Get(ctx context.Context, name string, options v1.GetOptions) (result *v2.Display, err error) {
	result = &v2.Display{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("displays").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Displays that match those selectors.
func (c *displays) List(ctx context.Context, opts v1.ListOptions) (result *v2.DisplayList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v2.DisplayList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("displays").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested displays.
func (c *displays) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("displays").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a display and creates it.  Returns the server's representation of the display, and an error, if there is any.
func (c *displays) Create(ctx context.Context, display *v2.Display, opts v1.CreateOptions) (result *v2.Display, err error) {
	result = &v2.Display{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("displays").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(display).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a display and updates it. Returns the server's representation of the display, and an error, if there is any.
func (c *displays) Update(ctx context.Context, display *v2.Display, opts v1.UpdateOptions) (result *v2.Display, err error) {
	result = &v2.Display{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("displays").
		Name(display.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(display).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *displays) UpdateStatus(ctx context.Context, display *v2.Display, opts v1.UpdateOptions) (result *v2.Display, err error) {
	result = &v2.Display{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("displays").
		Name(display.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(display).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the display and deletes it. Returns an error if one occurs.
func (c *displays) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("displays").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *displays) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("displays").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched display.
func (c *displays) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v2.Display, err error) {
	result = &v2.Display{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("displays").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
