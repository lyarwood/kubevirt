package apply

import (
	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

func applyMachinePreferences(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	if preferenceSpec.Machine == nil {
		return
	}

	if vmiSpec.Domain.Machine == nil {
		vmiSpec.Domain.Machine = &virtv1.Machine{}
	}

	if preferenceSpec.Machine.PreferredMachineType != "" && vmiSpec.Domain.Machine.Type == "" {
		vmiSpec.Domain.Machine.Type = preferenceSpec.Machine.PreferredMachineType
	}
}
