package apply

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
)

func applyInstanceTypeAnnotations(annotations map[string]string, target metav1.Object) (conflicts Conflicts) {
	if target.GetAnnotations() == nil {
		target.SetAnnotations(make(map[string]string))
	}

	targetAnnotations := target.GetAnnotations()
	for key, value := range annotations {
		if targetValue, exists := targetAnnotations[key]; exists {
			if targetValue != value {
				conflicts = append(conflicts, k8sfield.NewPath("annotations", key))
			}
			continue
		}
		targetAnnotations[key] = value
	}

	return conflicts
}
