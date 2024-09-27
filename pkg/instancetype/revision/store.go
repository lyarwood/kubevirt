package revision

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"

	virtv1 "kubevirt.io/api/core/v1"
	api "kubevirt.io/api/instancetype"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/instancetype/find"
)

func (r *Revision) Store(vm *virtv1.VirtualMachine) error {
	// Lazy logger construction
	logger := func() *log.FilteredLogger { return log.Log.Object(vm) }
	instancetypeRevision, err := r.storeInstancetypeRevision(vm)
	if err != nil {
		logger().Reason(err).Error("Failed to store ControllerRevision of VirtualMachineInstancetypeSpec for the Virtualmachine.")
		return err
	}

	preferenceRevision, err := r.storePreferenceRevision(vm)
	if err != nil {
		logger().Reason(err).Error("Failed to store ControllerRevision of VirtualMachinePreferenceSpec for the Virtualmachine.")
		return err
	}

	return r.patchRevisionName(instancetypeRevision, preferenceRevision, vm)
}

func (r *Revision) createInstancetypeRevision(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
	switch strings.ToLower(vm.Spec.Instancetype.Kind) {
	case api.SingularResourceName, api.PluralResourceName:
		instancetype, err := find.NewInstancetypeFinder(r.instancetypeStore, r.virtClient).Find(vm)
		if err != nil {
			return nil, err
		}

		// There is still a window where the instancetype can be updated between the VirtualMachine validation webhook accepting
		// the VirtualMachine and the VirtualMachine controller creating a ControllerRevison. As such we need to check one final
		// time that there are no conflicts when applying the instancetype to the VirtualMachine before continuing.
		if err := m.checkForInstancetypeConflicts(&instancetype.Spec, &vm.Spec.Template.Spec, &vm.Spec.Template.ObjectMeta); err != nil {
			return nil, err
		}
		return CreateControllerRevision(vm, instancetype)

	case api.ClusterSingularResourceName, api.ClusterPluralResourceName:
		finder := find.NewClusterInstancetypeFinder(m.InstancetypeStore, m.Clientset)
		clusterInstancetype, err := finder.Find(vm)
		if err != nil {
			return nil, err
		}

		// There is still a window where the instancetype can be updated between the VirtualMachine validation webhook accepting
		// the VirtualMachine and the VirtualMachine controller creating a ControllerRevison. As such we need to check one final
		// time that there are no conflicts when applying the instancetype to the VirtualMachine before continuing.
		if err := m.checkForInstancetypeConflicts(&clusterInstancetype.Spec, &vm.Spec.Template.Spec, &vm.Spec.Template.ObjectMeta); err != nil {
			return nil, err
		}
		return CreateControllerRevision(vm, clusterInstancetype)

	default:
		return nil, fmt.Errorf("got unexpected kind in InstancetypeMatcher: %s", vm.Spec.Instancetype.Kind)
	}
}

func (r *Revision) storeInstancetypeRevision(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
	if vm.Spec.Instancetype == nil || vm.Spec.Instancetype.RevisionName != "" {
		return nil, nil
	}

	instancetypeRevision, err := m.createInstancetypeRevision(vm)
	if err != nil {
		return nil, err
	}

	storedRevision, err := storeRevision(instancetypeRevision, m.Clientset)
	if err != nil {
		return nil, err
	}

	vm.Spec.Instancetype.RevisionName = storedRevision.Name
	return storedRevision, nil
}

func (r *Revision) createPreferenceRevision(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
	switch strings.ToLower(vm.Spec.Preference.Kind) {
	case api.SingularPreferenceResourceName, api.PluralPreferenceResourceName:
		preference, err := preferenceFind.NewPreferenceFinder(r.preferenceStore, r.virtClient).Find(vm)
		if err != nil {
			return nil, err
		}
		return CreateControllerRevision(vm, preference)
	case api.ClusterSingularPreferenceResourceName, api.ClusterPluralPreferenceResourceName:
		clusterPreference, err := preferenceFind.NewClusterPreferenceFinder(r.clusterPreferenceStore, r.virtClient)
		if err != nil {
			return nil, err
		}
		return CreateControllerRevision(vm, clusterPreference)
	default:
		return nil, fmt.Errorf("got unexpected kind in PreferenceMatcher: %s", vm.Spec.Preference.Kind)
	}
}

func (r *Revision) storePreferenceRevision(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
	if vm.Spec.Preference == nil || vm.Spec.Preference.RevisionName != "" {
		return nil, nil
	}

	preferenceRevision, err := m.createPreferenceRevision(vm)
	if err != nil {
		return nil, err
	}

	storedRevision, err := storeRevision(preferenceRevision)
	if err != nil {
		return nil, err
	}

	vm.Spec.Preference.RevisionName = storedRevision.Name
	return storedRevision, nil
}

func (r *Revision) storeRevision(revision *appsv1.ControllerRevision) (*appsv1.ControllerRevision, error) {
	createdRevision, err := r.virtClient.AppsV1().ControllerRevisions(revision.Namespace).Create(context.Background(), revision, metav1.CreateOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create ControllerRevision: %w", err)
		}

		// Grab the existing revision to check the data it contains
		existingRevision, err := clientset.AppsV1().ControllerRevisions(revision.Namespace).Get(context.Background(), revision.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get ControllerRevision: %w", err)
		}

		equal, err := Compare(revision, existingRevision)
		if err != nil {
			return nil, err
		}
		if !equal {
			return nil, fmt.Errorf("found existing ControllerRevision with unexpected data: %s", revision.Name)
		}
		return existingRevision, nil
	}
	return createdRevision, nil
}
