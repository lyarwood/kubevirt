package revision

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

func (r *Revision) patchRevisionName(instancetypeRevision, preferenceRevision *appsv1.ControllerRevision, vm *virtv1.VirtualMachine) error {
	// Batch any writes to the VirtualMachine into a single Patch() call to avoid races in the controller.
	logger := func() *log.FilteredLogger { return log.Log.Object(vm) }
	revisionPatch, err := GeneratePatch(instancetypeRevision, preferenceRevision)
	if err != nil {
		return err
	}
	if len(revisionPatch) == 0 {
		return nil
	}
	if _, err := r.virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, revisionPatch, metav1.PatchOptions{}); err != nil {
		logger().Reason(err).Error("Failed to update VirtualMachine with instancetype and preference ControllerRevision references.")
		return err
	}
}

func GeneratePatch(instancetypeRevision, preferenceRevision *appsv1.ControllerRevision) ([]byte, error) {
	patchSet := patch.New()
	if instancetypeRevision != nil {
		patchSet.AddOption(
			patch.WithTest("/spec/instancetype/revisionName", nil),
			patch.WithAdd("/spec/instancetype/revisionName", instancetypeRevision.Name),
		)
	}

	if preferenceRevision != nil {
		patchSet.AddOption(
			patch.WithTest("/spec/preference/revisionName", nil),
			patch.WithAdd("/spec/preference/revisionName", preferenceRevision.Name),
		)
	}

	if patchSet.IsEmpty() {
		return nil, nil
	}

	payload, err := patchSet.GeneratePayload()
	if err != nil {
		// This is a programmer's error and should not happen
		return nil, fmt.Errorf("failed to generate patch payload: %w", err)
	}

	return payload, nil
}
