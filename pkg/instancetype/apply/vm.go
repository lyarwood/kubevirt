package apply

import (
	"fmt"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/instancetype/find"
	preferenceFind "kubevirt.io/kubevirt/pkg/preference/find"
)

type VMApplier struct {
	vmiApplier         *VMIApplier
	instancetypeFinder *find.SpecFinder
	preferenceFinder   *preferenceFind.SpecFinder
}

func NewVMApplier(instancetypeFinder *find.SpecFinder, preferenceFinder *preferenceFind.SpecFinder) *VMApplier {
	return &VMApplier{
		vmiApplier:         NewVMIApplier(),
		instancetypeFinder: instancetypeFinder,
		preferenceFinder:   preferenceFinder,
	}
}

func (a *VMApplier) Apply(vm *virtv1.VirtualMachine) error {
	if vm.Spec.Instancetype == nil && vm.Spec.Preference == nil {
		return nil
	}
	instancetypeSpec, err := a.instancetypeFinder.Find(vm)
	if err != nil {
		return err
	}
	preferenceSpec, err := a.preferenceFinder.Find(vm)
	if err != nil {
		return err
	}
	if conflicts := a.vmiApplier.Apply(
		k8sfield.NewPath("spec"),
		instancetypeSpec,
		preferenceSpec,
		&vm.Spec.Template.Spec,
		&vm.ObjectMeta,
	); len(conflicts) > 0 {
		return fmt.Errorf("VM conflicts with instancetype spec in fields: [%s]", conflicts.String())
	}
	return nil
}
