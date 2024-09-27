package apply

import (
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

func applyGPUs(
	field *k8sfield.Path,
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) Conflicts {
	if len(instancetypeSpec.GPUs) == 0 {
		return nil
	}

	if len(vmiSpec.Domain.Devices.GPUs) >= 1 {
		return Conflicts{field.Child("domain", "devices", "gpus")}
	}

	vmiSpec.Domain.Devices.GPUs = make([]virtv1.GPU, len(instancetypeSpec.GPUs))
	copy(vmiSpec.Domain.Devices.GPUs, instancetypeSpec.GPUs)

	return nil
}
