//nolint:lll
package apply

import (
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

func applyMemory(field *k8sfield.Path, instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {
	if vmiSpec.Domain.Memory != nil {
		return Conflicts{field.Child("domain", "memory")}
	}

	if _, hasMemoryRequests := vmiSpec.Domain.Resources.Requests[k8sv1.ResourceMemory]; hasMemoryRequests {
		return Conflicts{field.Child("domain", "resources", "requests", string(k8sv1.ResourceMemory))}
	}

	if _, hasMemoryLimits := vmiSpec.Domain.Resources.Limits[k8sv1.ResourceMemory]; hasMemoryLimits {
		return Conflicts{field.Child("domain", "resources", "limits", string(k8sv1.ResourceMemory))}
	}

	instancetypeMemoryGuest := instancetypeSpec.Memory.Guest.DeepCopy()
	vmiSpec.Domain.Memory = &virtv1.Memory{
		Guest: &instancetypeMemoryGuest,
	}

	// If memory overcommit has been requested, set the memory requests to be
	// lower than the guest memory by the requested percent.
	const totalPercentage = 100
	if instancetypeMemoryOvercommit := instancetypeSpec.Memory.OvercommitPercent; instancetypeMemoryOvercommit > 0 {
		if vmiSpec.Domain.Resources.Requests == nil {
			vmiSpec.Domain.Resources.Requests = k8sv1.ResourceList{}
		}
		podRequestedMemory := int64(float32(instancetypeSpec.Memory.Guest.Value()) * (1 - float32(instancetypeSpec.Memory.OvercommitPercent)/totalPercentage))

		vmiSpec.Domain.Resources.Requests[k8sv1.ResourceMemory] = *resource.NewQuantity(podRequestedMemory, instancetypeMemoryGuest.Format)
	}

	if instancetypeSpec.Memory.Hugepages != nil {
		vmiSpec.Domain.Memory.Hugepages = instancetypeSpec.Memory.Hugepages.DeepCopy()
	}

	if instancetypeSpec.Memory.MaxGuest != nil {
		m := instancetypeSpec.Memory.MaxGuest.DeepCopy()
		vmiSpec.Domain.Memory.MaxGuest = &m
	}

	return nil
}
