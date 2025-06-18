package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=virtualmachinetemplates,singular=virtualmachinetemplate,categories=all
type VirtualMachineTemplate struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is the standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec VirtualMachineTemplateSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

type VirtualMachineTemplateSpec struct {
	// message is an optional instructional message that will
	// be displayed when this template is instantiated.
	// This field should inform the user how to utilize the newly created resources.
	// Parameter substitution will be performed on the message before being
	// displayed so that generated credentials and other parameters can be
	// included in the output.
	Message string `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`

	// +kubebuilder:pruning:PreserveUnknownFields
	VirtualMachine runtime.RawExtension `json:"virtualMachine" protobuf:"bytes,3,opt,name=virtualMachine"`

	// parameters is an optional array of Parameters used during the
	// Template to Config transformation.
	Parameters []Parameter `json:"parameters,omitempty" protobuf:"bytes,4,rep,name=parameters"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// VirtualMachineTemplateList is a list of Template objects.
type VirtualMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is the standard list's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// items is a list of templates
	Items []VirtualMachineTemplate `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// Parameter defines a name/value variable that is to be processed during
// the Template to Config transformation.
type Parameter struct {
	// name must be set and it can be referenced in Template
	// Items using ${PARAMETER_NAME}. Required.
	// +kubebuilder:validation:Required
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Optional: The name that will show in UI instead of parameter 'Name'
	DisplayName string `json:"displayName,omitempty" protobuf:"bytes,2,opt,name=displayName"`

	// description of a parameter. Optional.
	Description string `json:"description,omitempty" protobuf:"bytes,3,opt,name=description"`

	// value holds the Parameter data. If specified, the generator will be
	// ignored. The value replaces all occurrences of the Parameter ${Name}
	// expression during the Template to Config transformation. Optional.
	Value string `json:"value,omitempty" protobuf:"bytes,4,opt,name=value"`

	// generate specifies the generator to be used to generate random string
	// from an input value specified by From field. The result string is
	// stored into Value field. If empty, no generator is being used, leaving
	// the result Value untouched. Optional.
	//
	// The only supported generator is "expression", which accepts a "from"
	// value in the form of a simple regular expression containing the
	// range expression "[a-zA-Z0-9]", and the length expression "a{length}".
	//
	// Examples:
	//
	// from             | value
	// -----------------------------
	// "test[0-9]{1}x"  | "test7x"
	// "[0-1]{8}"       | "01001100"
	// "0x[A-F0-9]{4}"  | "0xB3AF"
	// "[a-zA-Z0-9]{8}" | "hW4yQU5i"
	//
	Generate string `json:"generate,omitempty" protobuf:"bytes,5,opt,name=generate"`

	// from is an input value for the generator. Optional.
	From string `json:"from,omitempty" protobuf:"bytes,6,opt,name=from"`

	// Optional: Indicates the parameter must have a value.  Defaults to false.
	Required bool `json:"required,omitempty" protobuf:"varint,7,opt,name=required"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=virtualmachinetemplaterequests,singular=virtualmachinetemplaterequest,categories=all
// +kubebuilder:subresource:status
type VirtualMachineTemplateRequest struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is the standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// spec defines the desired state of VirtualMachineTemplateRequest
	Spec VirtualMachineTemplateRequestSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// status defines the observed state of VirtualMachineTemplateRequest
	Status VirtualMachineTemplateRequestStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

type VirtualMachineTemplateRequestSpec struct {
	// source is a reference to the VirtualMachine to create a template from
	// +kubebuilder:validation:Required
	Source VirtualMachineReference `json:"source" protobuf:"bytes,1,opt,name=source"`
}

type VirtualMachineTemplateRequestStatus struct {
	// phase represents the current phase of the template request
	Phase VirtualMachineTemplateRequestPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase"`

	// snapshot references the VirtualMachineSnapshot created for this request
	Snapshot *corev1.TypedObjectReference `json:"snapshot,omitempty" protobuf:"bytes,2,opt,name=snapshot"`

	// template references the VirtualMachineTemplate created from this request
	Template *corev1.TypedObjectReference `json:"template,omitempty" protobuf:"bytes,3,opt,name=template"`

	// conditions represent the latest available observations of the template request's current state
	// +listType=map
	// +listMapKey=type
	Conditions []VirtualMachineTemplateRequestCondition `json:"conditions,omitempty" protobuf:"bytes,4,rep,name=conditions"`

	// observedGeneration is the most recent generation observed for this resource
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,5,opt,name=observedGeneration"`
}

// VirtualMachineTemplateRequestPhase represents the phase of a VirtualMachineTemplateRequest
type VirtualMachineTemplateRequestPhase string

const (
	// VirtualMachineTemplateRequestPhasePending indicates the request is pending processing
	VirtualMachineTemplateRequestPhasePending VirtualMachineTemplateRequestPhase = "Pending"
	// VirtualMachineTemplateRequestPhaseInProgress indicates the request is being processed
	VirtualMachineTemplateRequestPhaseInProgress VirtualMachineTemplateRequestPhase = "InProgress"
	// VirtualMachineTemplateRequestPhaseSucceeded indicates the request completed successfully
	VirtualMachineTemplateRequestPhaseSucceeded VirtualMachineTemplateRequestPhase = "Succeeded"
	// VirtualMachineTemplateRequestPhaseFailed indicates the request failed
	VirtualMachineTemplateRequestPhaseFailed VirtualMachineTemplateRequestPhase = "Failed"
)

// VirtualMachineReference represents a reference to a VirtualMachine
type VirtualMachineReference struct {
	// name is the name of the VirtualMachine
	// +kubebuilder:validation:Required
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// namespace is the namespace of the VirtualMachine
	// If not specified, defaults to the same namespace as the request
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,2,opt,name=namespace"`
}

// VirtualMachineTemplateRequestCondition represents a condition of a VirtualMachineTemplateRequest
type VirtualMachineTemplateRequestCondition struct {
	// type of the condition
	Type VirtualMachineTemplateRequestConditionType `json:"type" protobuf:"bytes,1,opt,name=type"`

	// status of the condition, one of True, False, Unknown
	Status metav1.ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status"`

	// lastTransitionTime is the last time the condition transitioned from one status to another
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,3,opt,name=lastTransitionTime"`

	// reason is a unique, one-word, CamelCase reason for the condition's last transition
	Reason string `json:"reason,omitempty" protobuf:"bytes,4,opt,name=reason"`

	// message is a human-readable message indicating details about the transition
	Message string `json:"message,omitempty" protobuf:"bytes,5,opt,name=message"`
}

// VirtualMachineTemplateRequestConditionType represents the type of condition
type VirtualMachineTemplateRequestConditionType string

const (
	// VirtualMachineTemplateRequestConditionReady indicates the request is ready
	VirtualMachineTemplateRequestConditionReady VirtualMachineTemplateRequestConditionType = "Ready"
	// VirtualMachineTemplateRequestConditionSourceReady indicates the source VM is ready
	VirtualMachineTemplateRequestConditionSourceReady VirtualMachineTemplateRequestConditionType = "SourceReady"
	// VirtualMachineTemplateRequestConditionSnapshotReady indicates the snapshot is ready
	VirtualMachineTemplateRequestConditionSnapshotReady VirtualMachineTemplateRequestConditionType = "SnapshotReady"
	// VirtualMachineTemplateRequestConditionTemplateReady indicates the template is ready
	VirtualMachineTemplateRequestConditionTemplateReady VirtualMachineTemplateRequestConditionType = "TemplateReady"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// VirtualMachineTemplateRequestList is a list of VirtualMachineTemplateRequest objects.
type VirtualMachineTemplateRequestList struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is the standard list's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// items is a list of VirtualMachineTemplateRequest objects
	Items []VirtualMachineTemplateRequest `json:"items" protobuf:"bytes,2,rep,name=items"`
}
