package find

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

type RevisionFinder struct {
	store      cache.Store
	virtClient kubecli.KubevirtClient
}

func NewRevisionFinder(store cache.Store, virtClient kubecli.KubevirtClient) *RevisionFinder {
	return &RevisionFinder{
		store:      store,
		virtClient: virtClient,
	}
}

func (f *RevisionFinder) Find(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
	if vm.Spec.Instancetype == nil {
		return nil, nil
	}
	namespacedName := types.NamespacedName{
		Namespace: vm.Namespace,
		Name:      vm.Spec.Instancetype.RevisionName,
	}
	if f.store == nil {
		return f.virtClient.AppsV1().ControllerRevisions(namespacedName.Namespace).Get(
			context.Background(), namespacedName.Name, metav1.GetOptions{})
	}

	obj, exists, err := f.store.GetByKey(namespacedName.String())
	if err != nil {
		return nil, err
	}
	if !exists {
		return f.virtClient.AppsV1().ControllerRevisions(namespacedName.Namespace).Get(
			context.Background(), namespacedName.Name, metav1.GetOptions{})
	}
	revision, ok := obj.(*appsv1.ControllerRevision)
	if !ok {
		return nil, fmt.Errorf("unknown object type found in ControllerRevision informer")
	}
	return revision, nil
}
