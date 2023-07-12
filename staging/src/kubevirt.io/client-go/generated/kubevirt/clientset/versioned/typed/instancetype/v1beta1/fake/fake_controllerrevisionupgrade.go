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
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

// FakeControllerRevisionUpgrades implements ControllerRevisionUpgradeInterface
type FakeControllerRevisionUpgrades struct {
	Fake *FakeInstancetypeV1beta1
	ns   string
}

var controllerrevisionupgradesResource = schema.GroupVersionResource{Group: "instancetype.kubevirt.io", Version: "v1beta1", Resource: "controllerrevisionupgrades"}

var controllerrevisionupgradesKind = schema.GroupVersionKind{Group: "instancetype.kubevirt.io", Version: "v1beta1", Kind: "ControllerRevisionUpgrade"}

// Get takes name of the controllerRevisionUpgrade, and returns the corresponding controllerRevisionUpgrade object, and an error if there is any.
func (c *FakeControllerRevisionUpgrades) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta1.ControllerRevisionUpgrade, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(controllerrevisionupgradesResource, c.ns, name), &v1beta1.ControllerRevisionUpgrade{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ControllerRevisionUpgrade), err
}

// List takes label and field selectors, and returns the list of ControllerRevisionUpgrades that match those selectors.
func (c *FakeControllerRevisionUpgrades) List(ctx context.Context, opts v1.ListOptions) (result *v1beta1.ControllerRevisionUpgradeList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(controllerrevisionupgradesResource, controllerrevisionupgradesKind, c.ns, opts), &v1beta1.ControllerRevisionUpgradeList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.ControllerRevisionUpgradeList{ListMeta: obj.(*v1beta1.ControllerRevisionUpgradeList).ListMeta}
	for _, item := range obj.(*v1beta1.ControllerRevisionUpgradeList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested controllerRevisionUpgrades.
func (c *FakeControllerRevisionUpgrades) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(controllerrevisionupgradesResource, c.ns, opts))

}

// Create takes the representation of a controllerRevisionUpgrade and creates it.  Returns the server's representation of the controllerRevisionUpgrade, and an error, if there is any.
func (c *FakeControllerRevisionUpgrades) Create(ctx context.Context, controllerRevisionUpgrade *v1beta1.ControllerRevisionUpgrade, opts v1.CreateOptions) (result *v1beta1.ControllerRevisionUpgrade, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(controllerrevisionupgradesResource, c.ns, controllerRevisionUpgrade), &v1beta1.ControllerRevisionUpgrade{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ControllerRevisionUpgrade), err
}

// Update takes the representation of a controllerRevisionUpgrade and updates it. Returns the server's representation of the controllerRevisionUpgrade, and an error, if there is any.
func (c *FakeControllerRevisionUpgrades) Update(ctx context.Context, controllerRevisionUpgrade *v1beta1.ControllerRevisionUpgrade, opts v1.UpdateOptions) (result *v1beta1.ControllerRevisionUpgrade, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(controllerrevisionupgradesResource, c.ns, controllerRevisionUpgrade), &v1beta1.ControllerRevisionUpgrade{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ControllerRevisionUpgrade), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeControllerRevisionUpgrades) UpdateStatus(ctx context.Context, controllerRevisionUpgrade *v1beta1.ControllerRevisionUpgrade, opts v1.UpdateOptions) (*v1beta1.ControllerRevisionUpgrade, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(controllerrevisionupgradesResource, "status", c.ns, controllerRevisionUpgrade), &v1beta1.ControllerRevisionUpgrade{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ControllerRevisionUpgrade), err
}

// Delete takes name of the controllerRevisionUpgrade and deletes it. Returns an error if one occurs.
func (c *FakeControllerRevisionUpgrades) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(controllerrevisionupgradesResource, c.ns, name), &v1beta1.ControllerRevisionUpgrade{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeControllerRevisionUpgrades) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(controllerrevisionupgradesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1beta1.ControllerRevisionUpgradeList{})
	return err
}

// Patch applies the patch and returns the patched controllerRevisionUpgrade.
func (c *FakeControllerRevisionUpgrades) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta1.ControllerRevisionUpgrade, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(controllerrevisionupgradesResource, c.ns, name, pt, data, subresources...), &v1beta1.ControllerRevisionUpgrade{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.ControllerRevisionUpgrade), err
}
