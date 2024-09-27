//nolint:lll
package apply

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	preferenceApply "kubevirt.io/kubevirt/pkg/preference/apply"
)

type VMIApplier struct {
	preferenceApplier *preferenceApply.VMIApplier
}

func NewVMIApplier() *VMIApplier {
	return &VMIApplier{
		preferenceApplier: &preferenceApply.VMIApplier{},
	}
}

func (a *VMIApplier) Apply(field *k8sfield.Path, instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec, vmiMetadata *metav1.ObjectMeta) (conflicts Conflicts) {
	if instancetypeSpec == nil && preferenceSpec == nil {
		return
	}

	if instancetypeSpec != nil {
		conflicts = append(conflicts, applyNodeSelector(field, instancetypeSpec, vmiSpec)...)
		conflicts = append(conflicts, applySchedulerName(field, instancetypeSpec, vmiSpec)...)
		conflicts = append(conflicts, applyCPU(field, instancetypeSpec, preferenceSpec, vmiSpec)...)
		conflicts = append(conflicts, applyMemory(field, instancetypeSpec, vmiSpec)...)
		conflicts = append(conflicts, applyIOThreadPolicy(field, instancetypeSpec, vmiSpec)...)
		conflicts = append(conflicts, applyLaunchSecurity(field, instancetypeSpec, vmiSpec)...)
		conflicts = append(conflicts, applyGPUs(field, instancetypeSpec, vmiSpec)...)
		conflicts = append(conflicts, applyHostDevices(field, instancetypeSpec, vmiSpec)...)
		conflicts = append(conflicts, applyInstanceTypeAnnotations(instancetypeSpec.Annotations, vmiMetadata)...)
	}

	if len(conflicts) > 0 {
		return
	}

	a.preferenceApplier.Apply(preferenceSpec, vmiSpec, vmiMetadata)

	return
}
