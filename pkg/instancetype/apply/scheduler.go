//nolint:lll
package apply

import (
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

func applySchedulerName(field *k8sfield.Path, instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {
	if instancetypeSpec.SchedulerName == "" {
		return nil
	}

	if vmiSpec.SchedulerName != "" {
		return Conflicts{field.Child("schedulerName")}
	}

	vmiSpec.SchedulerName = instancetypeSpec.SchedulerName

	return nil
}
