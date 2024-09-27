package revision

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
)

func GetName(vmName, resourceName, resourceVersion string, resourceUID types.UID, resourceGeneration int64) string {
	return fmt.Sprintf("%s-%s-%s-%s-%d", vmName, resourceName, resourceVersion, resourceUID, resourceGeneration)
}

func CreateControllerRevision(vm *virtv1.VirtualMachine, object runtime.Object) (*appsv1.ControllerRevision, error) {
	obj, err := utils.GenerateKubeVirtGroupVersionKind(object)
	if err != nil {
		return nil, err
	}
	metaObj, ok := obj.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("unexpected object format returned from GenerateKubeVirtGroupVersionKind")
	}

	revisionName := GetName(
		vm.Name, metaObj.GetName(),
		obj.GetObjectKind().GroupVersionKind().Version,
		metaObj.GetUID(),
		metaObj.GetGeneration(),
	)

	// Removing unnecessary metadata
	metaObj.SetLabels(nil)
	metaObj.SetAnnotations(nil)
	metaObj.SetFinalizers(nil)
	metaObj.SetOwnerReferences(nil)
	metaObj.SetManagedFields(nil)

	return &appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:            revisionName,
			Namespace:       vm.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind)},
			Labels: map[string]string{
				apiinstancetype.ControllerRevisionObjectGenerationLabel: fmt.Sprintf("%d", metaObj.GetGeneration()),
				apiinstancetype.ControllerRevisionObjectKindLabel:       obj.GetObjectKind().GroupVersionKind().Kind,
				apiinstancetype.ControllerRevisionObjectNameLabel:       metaObj.GetName(),
				apiinstancetype.ControllerRevisionObjectUIDLabel:        string(metaObj.GetUID()),
				apiinstancetype.ControllerRevisionObjectVersionLabel:    obj.GetObjectKind().GroupVersionKind().Version,
			},
		},
		Data: runtime.RawExtension{
			Object: obj,
		},
	}, nil
}
