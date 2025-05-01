/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
type VirtualMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VirtualMachineTemplateSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineTemplate `json:"items"`
}

type VirtualMachineTemplateSpec struct {
	// A static VirtualMachine definition
	// +optional
	VirtualMachine *virtv1.VirtualMachine `json:"virtualMachine,omitempty"`

	// A reference to an existing VirtualMachine
	// +optional
	VirtualMachineRef *corev1.ObjectReference `json:"virtualMachineRef,omitempty"`

	// A reference to an existing VirtualMachineSnapshotRef
	// +optional
	VirtualMachineSnapshotRef *corev1.ObjectReference `json:"virtualMachineSnapshotRef,omitempty"`
}

type VirtualMachineTemplateStatus struct {
	// A reference to a captured VirtualMachineSnapshotRef
	// +optional
	VirtualMachineSnapshotRef *corev1.ObjectReference `json:"virtualMachineSnapshotRef,omitempty"`
}
