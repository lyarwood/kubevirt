package find

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
)

type ClusterPreferenceFinder struct {
	store      cache.Store
	virtClient kubecli.KubevirtClient
}

func NewClusterPreferenceFinder(store cache.Store, virtClient kubecli.KubevirtClient) *ClusterPreferenceFinder {
	return &ClusterPreferenceFinder{
		store:      store,
		virtClient: virtClient,
	}
}

func (f *ClusterPreferenceFinder) Find(vm *virtv1.VirtualMachine) (*v1beta1.VirtualMachineClusterPreference, error) {
	if vm.Spec.Preference == nil {
		return nil, nil
	}
	if f.store == nil {
		return f.virtClient.VirtualMachineClusterPreference().Get(
			context.Background(), vm.Spec.Preference.Name, metav1.GetOptions{})
	}

	obj, exists, err := f.store.GetByKey(vm.Spec.Preference.Name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return f.virtClient.VirtualMachineClusterPreference().Get(
			context.Background(), vm.Spec.Preference.Name, metav1.GetOptions{})
	}
	instancetype, ok := obj.(*v1beta1.VirtualMachineClusterPreference)
	if !ok {
		return nil, fmt.Errorf("unknown object type found in VirtualMachineClusterPreference informer")
	}
	return instancetype, nil
}
