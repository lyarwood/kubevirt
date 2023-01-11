/*
Copyright 2023 The KubeVirt Authors.

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

package fake

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
	v1alpha3 "kubevirt.io/api/instancetype/v1alpha3"
)

// FakeVirtualMachineInstancetypes implements VirtualMachineInstancetypeInterface
type FakeVirtualMachineInstancetypes struct {
	Fake *FakeInstancetypeV1alpha3
	ns   string
}

var virtualmachineinstancetypesResource = schema.GroupVersionResource{Group: "instancetype.kubevirt.io", Version: "v1alpha3", Resource: "virtualmachineinstancetypes"}

var virtualmachineinstancetypesKind = schema.GroupVersionKind{Group: "instancetype.kubevirt.io", Version: "v1alpha3", Kind: "VirtualMachineInstancetype"}

// Get takes name of the virtualMachineInstancetype, and returns the corresponding virtualMachineInstancetype object, and an error if there is any.
func (c *FakeVirtualMachineInstancetypes) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha3.VirtualMachineInstancetype, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(virtualmachineinstancetypesResource, c.ns, name), &v1alpha3.VirtualMachineInstancetype{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha3.VirtualMachineInstancetype), err
}

// List takes label and field selectors, and returns the list of VirtualMachineInstancetypes that match those selectors.
func (c *FakeVirtualMachineInstancetypes) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha3.VirtualMachineInstancetypeList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(virtualmachineinstancetypesResource, virtualmachineinstancetypesKind, c.ns, opts), &v1alpha3.VirtualMachineInstancetypeList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha3.VirtualMachineInstancetypeList{ListMeta: obj.(*v1alpha3.VirtualMachineInstancetypeList).ListMeta}
	for _, item := range obj.(*v1alpha3.VirtualMachineInstancetypeList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested virtualMachineInstancetypes.
func (c *FakeVirtualMachineInstancetypes) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(virtualmachineinstancetypesResource, c.ns, opts))

}

// Create takes the representation of a virtualMachineInstancetype and creates it.  Returns the server's representation of the virtualMachineInstancetype, and an error, if there is any.
func (c *FakeVirtualMachineInstancetypes) Create(ctx context.Context, virtualMachineInstancetype *v1alpha3.VirtualMachineInstancetype, opts v1.CreateOptions) (result *v1alpha3.VirtualMachineInstancetype, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(virtualmachineinstancetypesResource, c.ns, virtualMachineInstancetype), &v1alpha3.VirtualMachineInstancetype{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha3.VirtualMachineInstancetype), err
}

// Update takes the representation of a virtualMachineInstancetype and updates it. Returns the server's representation of the virtualMachineInstancetype, and an error, if there is any.
func (c *FakeVirtualMachineInstancetypes) Update(ctx context.Context, virtualMachineInstancetype *v1alpha3.VirtualMachineInstancetype, opts v1.UpdateOptions) (result *v1alpha3.VirtualMachineInstancetype, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(virtualmachineinstancetypesResource, c.ns, virtualMachineInstancetype), &v1alpha3.VirtualMachineInstancetype{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha3.VirtualMachineInstancetype), err
}

// Delete takes name of the virtualMachineInstancetype and deletes it. Returns an error if one occurs.
func (c *FakeVirtualMachineInstancetypes) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(virtualmachineinstancetypesResource, c.ns, name), &v1alpha3.VirtualMachineInstancetype{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeVirtualMachineInstancetypes) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(virtualmachineinstancetypesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha3.VirtualMachineInstancetypeList{})
	return err
}

// Patch applies the patch and returns the patched virtualMachineInstancetype.
func (c *FakeVirtualMachineInstancetypes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha3.VirtualMachineInstancetype, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(virtualmachineinstancetypesResource, c.ns, name, pt, data, subresources...), &v1alpha3.VirtualMachineInstancetype{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha3.VirtualMachineInstancetype), err
}
