//nolint:gocyclo
package apply

import (
	k8sv1 "k8s.io/api/core/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	preferenceApply "kubevirt.io/kubevirt/pkg/preference/apply"
)

func applyCPU(
	field *k8sfield.Path,
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) Conflicts {
	if vmiSpec.Domain.CPU == nil {
		vmiSpec.Domain.CPU = &virtv1.CPU{}
	}

	// If we have any conflicts return as there's no need to apply the topology below
	if conflicts := validateCPU(field, instancetypeSpec, vmiSpec); len(conflicts) > 0 {
		return conflicts
	}

	if vmiSpec.Domain.CPU.Model == "" && instancetypeSpec.CPU.Model != nil {
		vmiSpec.Domain.CPU.Model = *instancetypeSpec.CPU.Model
	}

	if instancetypeSpec.CPU.DedicatedCPUPlacement != nil {
		vmiSpec.Domain.CPU.DedicatedCPUPlacement = *instancetypeSpec.CPU.DedicatedCPUPlacement
	}

	if instancetypeSpec.CPU.IsolateEmulatorThread != nil {
		vmiSpec.Domain.CPU.IsolateEmulatorThread = *instancetypeSpec.CPU.IsolateEmulatorThread
	}

	if vmiSpec.Domain.CPU.NUMA == nil && instancetypeSpec.CPU.NUMA != nil {
		vmiSpec.Domain.CPU.NUMA = instancetypeSpec.CPU.NUMA.DeepCopy()
	}

	if vmiSpec.Domain.CPU.Realtime == nil && instancetypeSpec.CPU.Realtime != nil {
		vmiSpec.Domain.CPU.Realtime = instancetypeSpec.CPU.Realtime.DeepCopy()
	}

	if instancetypeSpec.CPU.MaxSockets != nil {
		vmiSpec.Domain.CPU.MaxSockets = *instancetypeSpec.CPU.MaxSockets
	}

	applyGuestCPUTopology(instancetypeSpec.CPU.Guest, preferenceSpec, vmiSpec)

	return nil
}

func applyGuestCPUTopology(vCPUs uint32, preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	// Apply the default topology here to avoid duplication below
	vmiSpec.Domain.CPU.Cores = 1
	vmiSpec.Domain.CPU.Sockets = 1
	vmiSpec.Domain.CPU.Threads = 1

	if vCPUs == 1 {
		return
	}

	switch preferenceApply.GetPreferredTopology(preferenceSpec) {
	case v1beta1.DeprecatedPreferCores, v1beta1.Cores:
		vmiSpec.Domain.CPU.Cores = vCPUs
	case v1beta1.DeprecatedPreferSockets, v1beta1.DeprecatedPreferAny, v1beta1.Sockets, v1beta1.Any:
		vmiSpec.Domain.CPU.Sockets = vCPUs
	case v1beta1.DeprecatedPreferThreads, v1beta1.Threads:
		vmiSpec.Domain.CPU.Threads = vCPUs
	case v1beta1.DeprecatedPreferSpread, v1beta1.Spread:
		ratio, across := preferenceApply.GetSpreadOptions(preferenceSpec)
		switch across {
		case v1beta1.SpreadAcrossSocketsCores:
			vmiSpec.Domain.CPU.Cores = ratio
			vmiSpec.Domain.CPU.Sockets = vCPUs / ratio
		case v1beta1.SpreadAcrossCoresThreads:
			vmiSpec.Domain.CPU.Threads = ratio
			vmiSpec.Domain.CPU.Cores = vCPUs / ratio
		case v1beta1.SpreadAcrossSocketsCoresThreads:
			const threadsPerCore = 2
			vmiSpec.Domain.CPU.Threads = threadsPerCore
			vmiSpec.Domain.CPU.Cores = ratio
			vmiSpec.Domain.CPU.Sockets = vCPUs / threadsPerCore / ratio
		}
	}
}

func validateCPU(
	field *k8sfield.Path,
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) (conflicts Conflicts) {
	if _, hasCPURequests := vmiSpec.Domain.Resources.Requests[k8sv1.ResourceCPU]; hasCPURequests {
		conflicts = append(conflicts, field.Child("domain", "resources", "requests", string(k8sv1.ResourceCPU)))
	}

	if _, hasCPULimits := vmiSpec.Domain.Resources.Limits[k8sv1.ResourceCPU]; hasCPULimits {
		conflicts = append(conflicts, field.Child("domain", "resources", "limits", string(k8sv1.ResourceCPU)))
	}

	if vmiSpec.Domain.CPU.Sockets != 0 {
		conflicts = append(conflicts, field.Child("domain", "cpu", "sockets"))
	}

	if vmiSpec.Domain.CPU.Cores != 0 {
		conflicts = append(conflicts, field.Child("domain", "cpu", "cores"))
	}

	if vmiSpec.Domain.CPU.Threads != 0 {
		conflicts = append(conflicts, field.Child("domain", "cpu", "threads"))
	}

	if vmiSpec.Domain.CPU.Model != "" && instancetypeSpec.CPU.Model != nil {
		conflicts = append(conflicts, field.Child("domain", "cpu", "model"))
	}

	if vmiSpec.Domain.CPU.DedicatedCPUPlacement && instancetypeSpec.CPU.DedicatedCPUPlacement != nil {
		conflicts = append(conflicts, field.Child("domain", "cpu", "dedicatedCPUPlacement"))
	}

	if vmiSpec.Domain.CPU.IsolateEmulatorThread && instancetypeSpec.CPU.IsolateEmulatorThread != nil {
		conflicts = append(conflicts, field.Child("domain", "cpu", "isolateEmulatorThread"))
	}

	if vmiSpec.Domain.CPU.NUMA != nil && instancetypeSpec.CPU.NUMA != nil {
		conflicts = append(conflicts, field.Child("domain", "cpu", "numa"))
	}

	if vmiSpec.Domain.CPU.Realtime != nil && instancetypeSpec.CPU.Realtime != nil {
		conflicts = append(conflicts, field.Child("domain", "cpu", "realtime"))
	}

	return conflicts
}
