package apply

import (
	"maps"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

func applyNodeSelector(
	field *k8sfield.Path,
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) Conflicts {
	if instancetypeSpec.NodeSelector == nil {
		return nil
	}

	if vmiSpec.NodeSelector != nil {
		return Conflicts{field.Child("nodeSelector")}
	}

	vmiSpec.NodeSelector = maps.Clone(instancetypeSpec.NodeSelector)

	return nil
}
