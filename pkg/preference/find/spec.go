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
	preferenceFinder        *PreferenceFinder
	clusterPreferenceFinder *ClusterPreferenceFinder
	revisionFinder          *RevisionFinder
}

func NewSpecFinder(store, clusterStore, revisionStore cache.Store, virtClient kubecli.KubevirtClient) *SpecFinder {
	return &SpecFinder{
		preferenceFinder:        NewPreferenceFinder(store, virtClient),
		clusterPreferenceFinder: NewClusterPreferenceFinder(clusterStore, virtClient),
		revisionFinder:          NewRevisionFinder(revisionStore, virtClient),
	}
}

const unexpectedKindFmt = "got unexpected kind in PreferenceMatcher: %s"

func (f *SpecFinder) Find(vm *virtv1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error) {
	if vm.Spec.Preference == nil {
		return nil, nil
	}

	if vm.Spec.Preference.RevisionName != "" {
		revision, err := f.revisionFinder.Find(vm)
		if err != nil {
			return nil, err
		}
		return compatibility.GetPreferenceSpec(revision)
	}

	switch strings.ToLower(vm.Spec.Preference.Kind) {
	case api.SingularPreferenceResourceName, api.PluralPreferenceResourceName:
		preference, err := f.preferenceFinder.Find(vm)
		if err != nil {
			return nil, err
		}
		return &preference.Spec, nil

	case api.ClusterSingularPreferenceResourceName, api.ClusterPluralPreferenceResourceName, "":
		clusterPreference, err := f.clusterPreferenceFinder.Find(vm)
		if err != nil {
			return nil, err
		}
		return &clusterPreference.Spec, nil

	default:
		return nil, fmt.Errorf(unexpectedKindFmt, vm.Spec.Preference.Kind)
	}
}
