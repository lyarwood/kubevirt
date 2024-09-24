package find

import (
	"fmt"
	"strings"

	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	api "kubevirt.io/api/instancetype"
	"kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/instancetype/compatibility"
)

type SpecFinder struct {
	instancetypeFinder        *InstancetypeFinder
	clusterInstancetypeFinder *ClusterInstancetypeFinder
	revisionFinder            *RevisionFinder
}

func NewSpecFinder(store, clusterStore, revisionStore cache.Store, virtClient kubecli.KubevirtClient) *SpecFinder {
	return &SpecFinder{
		instancetypeFinder:        NewInstancetypeFinder(store, virtClient),
		clusterInstancetypeFinder: NewClusterInstancetypeFinder(clusterStore, virtClient),
		revisionFinder:            NewRevisionFinder(revisionStore, virtClient),
	}
}

const unexpectedKindFmt = "got unexpected kind in InstancetypeMatcher: %s"

func (f *SpecFinder) Find(vm *virtv1.VirtualMachine) (*v1beta1.VirtualMachineInstancetypeSpec, error) {
	if vm.Spec.Instancetype == nil {
		return nil, nil
	}

	if vm.Spec.Instancetype.RevisionName != "" {
		revision, err := f.revisionFinder.Find(vm)
		if err != nil {
			return nil, err
		}
		return compatibility.GetInstancetypeSpec(revision)
	}

	switch strings.ToLower(vm.Spec.Instancetype.Kind) {
	case api.SingularResourceName, api.PluralResourceName:
		instancetype, err := f.instancetypeFinder.Find(vm)
		if err != nil {
			return nil, err
		}
		return &instancetype.Spec, nil

	case api.ClusterSingularResourceName, api.ClusterPluralResourceName, "":
		clusterInstancetype, err := f.clusterInstancetypeFinder.Find(vm)
		if err != nil {
			return nil, err
		}
		return &clusterInstancetype.Spec, nil

	default:
		return nil, fmt.Errorf(unexpectedKindFmt, vm.Spec.Instancetype.Kind)
	}
}
