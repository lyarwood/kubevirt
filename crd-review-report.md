# CRD Review Report: KubeVirt instancetype.kubevirt.io/v1beta1

**Generated:** 2025-11-14
**API Location:** `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go`
**API Group:** `instancetype.kubevirt.io`
**API Version:** `v1beta1`

## Summary

This report reviews the KubeVirt instancetype v1beta1 API definitions against Kubernetes and OpenShift API conventions. The API is generally well-structured and follows most best practices. However, there are several areas for improvement, particularly around boolean fields, documentation clarity, and validation markers.

**Findings Summary:**
- ❌ **3 Critical Issues** - API design patterns that should be addressed
- ⚠️ **12 Warnings** - Recommended improvements for better API quality
- 💡 **5 Info** - Optional enhancements to consider

---

## Table of Contents

- [Critical Issues](#critical-issues-)
- [Warnings](#warnings-)
- [Informational](#informational-)
- [Positive Findings](#positive-findings-)
- [Recommendations Summary](#recommendations-summary)
- [References](#references)

---

## Critical Issues (❌)

### 1. Boolean Fields Should Be Enums (OpenShift Convention)

**Location:** `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:151`

**Issue:** `DedicatedCPUPlacement` uses a boolean where an enum would be clearer.

```go
DedicatedCPUPlacement *bool `json:"dedicatedCPUPlacement,omitempty"`
```

**Why this matters:** Boolean fields force users to understand implementation details rather than expressing their intent. Enums are more extensible and user-friendly.

**Recommendation:**
```go
// CPUPlacementPolicy defines how vCPUs are placed on the host
// +kubebuilder:validation:Enum=Dedicated;Shared;Auto
CPUPlacementPolicy *string `json:"cpuPlacementPolicy,omitempty"`
```

**Breaking Change:** Yes - This would require API migration. Consider for v1 or v2.

**References:**
- [OpenShift API Conventions - Avoid Booleans](https://github.com/openshift/enhancements/blob/master/dev-guide/api-conventions.md#avoid-boolean-fields)

---

### 2. Boolean Fields in Device Preferences

**Location:** `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:415-456`

**Issue:** Multiple preference fields use booleans where enums would be clearer:
- `PreferredAutoattachGraphicsDevice`
- `PreferredAutoattachMemBalloon`
- `PreferredAutoattachPodInterface`
- `PreferredAutoattachSerialConsole`
- `PreferredAutoattachInputDevice`
- `PreferredDisableHotplug`
- `PreferredUseVirtioTransitional`
- `PreferredUseBios`
- `PreferredUseBiosSerial`

**Why this matters:** These fields express user intent about device attachment behavior. Enums would allow for more nuanced control and future expansion (e.g., "Auto", "Always", "Never").

**Recommendation:**
```go
// AttachmentPolicy defines how devices are attached
// +kubebuilder:validation:Enum=Auto;Always;Never
type AttachmentPolicy string

const (
    AttachmentPolicyAuto   AttachmentPolicy = "Auto"
    AttachmentPolicyAlways AttachmentPolicy = "Always"
    AttachmentPolicyNever  AttachmentPolicy = "Never"
)

// PreferredGraphicsDevicePolicy optionally defines the preferred graphics device attachment policy
// +optional
PreferredGraphicsDevicePolicy *AttachmentPolicy `json:"preferredGraphicsDevicePolicy,omitempty"`
```

**Breaking Change:** Yes - Consider for v2 or create new fields alongside deprecated boolean ones.

**References:**
- [OpenShift API Conventions - Avoid Booleans](https://github.com/openshift/enhancements/blob/master/dev-guide/api-conventions.md#avoid-boolean-fields)

---

### 3. Boolean Field: `IsolateEmulatorThread`

**Location:** `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:160`

**Issue:** Uses boolean for thread isolation policy.

```go
IsolateEmulatorThread *bool `json:"isolateEmulatorThread,omitempty"`
```

**Recommendation:**
```go
// EmulatorThreadPolicy defines how emulator threads are isolated
// +kubebuilder:validation:Enum=Shared;Isolated;Auto
EmulatorThreadPolicy *string `json:"emulatorThreadPolicy,omitempty"`
```

**Breaking Change:** Yes

**References:**
- [OpenShift API Conventions - Avoid Booleans](https://github.com/openshift/enhancements/blob/master/dev-guide/api-conventions.md#avoid-boolean-fields)

---

## Warnings (⚠️)

### 4. Unsigned Integer Type Used

**Location:** `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:138`

**Issue:** `Guest` field uses `uint32` instead of `int32`.

```go
Guest uint32 `json:"guest"`
```

**Why this matters:** Kubernetes API conventions prefer signed integers to avoid overflow issues and ensure consistent behavior across languages.

**Recommendation:**
```go
// guest is the required number of vCPUs to expose to the guest.
// The resulting CPU topology is derived from the optional preferredCPUTopology
// attribute of CPUPreferences which defaults to sockets.
// +kubebuilder:validation:Minimum=1
Guest int32 `json:"guest"`
```

**Breaking Change:** Yes - Type change would break API compatibility. Consider for v2.

**References:**
- [Kubernetes API Conventions - Integers](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#primitive-types)

---

### 5. Additional Unsigned Integer Fields

**Location:** Multiple locations:
- `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:168` (`MaxSockets`)
- `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:304` (`PreferSpreadSocketToCoreRatio`)
- `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:406` (`Ratio`)
- `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:651` (`Guest` in CPUPreferenceRequirement)

**Issue:** All use `uint32` instead of preferred `int32`.

**Recommendation:** Convert to `int32` with appropriate validation:
```go
// +kubebuilder:validation:Minimum=1
MaxSockets *int32 `json:"maxSockets,omitempty"`
```

**Breaking Change:** Yes

---

### 6. Inconsistent Use of JSON Names in Documentation

**Location:** `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:77-90`

**Issue:** Documentation mentions "NodeSelector" and "SchedulerName" in comments but should reference JSON field names.

**Current:**
```go
// NodeSelector is a selector which must be true for the vmi to fit on a node.
// Selector which must match a node's labels for the vmi to be scheduled on that node.
// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
//
// NodeSelector is the name of the custom node selector for the instancetype.
```

**Recommendation:**
```go
// nodeSelector is a selector which must be true for the VMI to fit on a node.
// The selector must match a node's labels for the VMI to be scheduled on that node.
// When specified by the instancetype, this selector is applied to VirtualMachineInstances
// that reference this instancetype.
// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
```

**Breaking Change:** No - Documentation only

**References:**
- [OpenShift API Conventions - Documentation](https://github.com/openshift/enhancements/blob/master/dev-guide/api-conventions.md#documentation)

---

### 7. Missing Field-Level Validation

**Location:** `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:138`

**Issue:** `Guest` field (CPU count) lacks minimum value validation.

**Current:**
```go
Guest uint32 `json:"guest"`
```

**Recommendation:**
```go
// guest is the required number of vCPUs to expose to the guest.
// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Required
Guest uint32 `json:"guest"`
```

**Breaking Change:** No - Adds validation

---

### 8. Missing Validation for MaxSockets

**Location:** `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:168`

**Issue:** `MaxSockets` should have validation to ensure it's greater than or equal to current socket count.

**Recommendation:**
```go
// maxSockets specifies the maximum amount of sockets that can be hotplugged.
// This value must be greater than or equal to the initial socket count.
// +kubebuilder:validation:Minimum=1
// +optional
MaxSockets *uint32 `json:"maxSockets,omitempty"`
```

**Breaking Change:** No

---

### 9. Typo in Documentation

**Location:** `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:276`

**Issue:** "VirtualMachineInstace" should be "VirtualMachineInstance"

```go
// Volumes optionally defines preferences associated with the Volumes attribute of a VirtualMachineInstace DomainSpec
```

**Recommendation:**
```go
// Volumes optionally defines preferences associated with the Volumes attribute of a VirtualMachineInstance DomainSpec
```

**Breaking Change:** No

---

### 10. Typo in Field Name

**Location:** `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:314`

**Issue:** "PreffereedStorageClassName" has extra 'e'

```go
PreffereedStorageClassName string `json:"preferredStorageClassName,omitempty"`
```

**Recommendation:**
```go
// preferredStorageClassName optionally defines the preferred StorageClass for volumes.
// When specified, this storage class is suggested for PersistentVolumeClaims created
// for VirtualMachineInstances using this preference.
PreferredStorageClassName string `json:"preferredStorageClassName,omitempty"`
```

**Breaking Change:** No for Go struct field rename, but JSON field name should remain the same for API compatibility

---

### 11. Typo in Documentation

**Location:** `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:291`

**Issue:** "minium" should be "minimum"

```go
// Requirements defines the minium amount of instance type defined resources required by a set of preferences
```

**Recommendation:**
```go
// requirements defines the minimum amount of instancetype-defined resources required by a set of preferences.
// When specified, these requirements ensure that VirtualMachines referencing this preference
// use instancetypes that meet or exceed these minimums.
```

**Breaking Change:** No

---

### 12. Typo in Documentation

**Location:** `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:306`

**Issue:** "prefeerred" should be "preferred"

```go
// PreferredArchitecture defines a prefeerred architecture for the VirtualMachine
```

**Recommendation:**
```go
// preferredArchitecture defines a preferred architecture for the VirtualMachine.
// Examples include: "amd64", "arm64", "ppc64le", "s390x"
```

**Breaking Change:** No

---

### 13. Typo in Documentation

**Location:** `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:580`

**Issue:** "transmitts" should be "transmits"

```go
// PreferredUseBiosSerial optionally transmitts BIOS output over the serial.
```

**Recommendation:**
```go
// preferredUseBiosSerial optionally transmits BIOS output over the serial console.
// This requires preferredUseBios to be enabled.
```

**Breaking Change:** No

---

### 14. Typo in Documentation

**Location:** `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:624`

**Issue:** "whih" should be "which"

```go
// Timer specifies whih timers are attached to the vmi.
```

**Recommendation:**
```go
// preferredTimer specifies which timers are attached to the VirtualMachineInstance.
```

**Breaking Change:** No

---

### 15. Missing Validation for SpreadOptions.Ratio

**Location:** `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:406`

**Issue:** The documentation states "Only a ratio of 2 is currently accepted" but there's no kubebuilder validation enforcing this.

**Recommendation:**
```go
// ratio optionally defines the ratio to spread vCPUs across the guest visible topology:
//
// CoresThreads        - 1:2   - Controls the ratio of cores to threads. Only a ratio of 2 is currently accepted.
// SocketsCores        - 1:N   - Controls the ratio of socket to cores.
// SocketsCoresThreads - 1:N:2 - Controls the ratio of socket to cores. Each core providing 2 threads.
//
// Default: 2
//
// +kubebuilder:validation:Minimum=1
// +optional
Ratio *uint32 `json:"ratio,omitempty"`
```

Or if only 2 is valid for some modes:
```go
// +kubebuilder:validation:Minimum=1
// +kubebuilder:validation:Maximum=128
```

**Breaking Change:** No

---

## Informational (💡)

### 16. Missing Status Subresource

**Location:** All top-level resource types

**Issue:** The API resources (`VirtualMachineInstancetype`, `VirtualMachineClusterInstancetype`, `VirtualMachinePreference`, `VirtualMachineClusterPreference`) don't have Status subresources.

**Why this matters:** While these appear to be configuration-only resources (similar to ConfigMap), having a Status subresource would allow tracking:
- Applied/observed generations
- Conditions (e.g., "Valid", "InUse")
- Usage metrics (number of VMs using this instancetype)

**Recommendation:** Consider if status tracking would be valuable:
```go
type VirtualMachineInstancetype struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   VirtualMachineInstancetypeSpec   `json:"spec"`
    // +optional
    Status VirtualMachineInstancetypeStatus `json:"status,omitempty"`
}

type VirtualMachineInstancetypeStatus struct {
    // observedGeneration is the last generation observed by the controller
    // +optional
    ObservedGeneration int64 `json:"observedGeneration,omitempty"`

    // conditions represent the latest available observations of the instancetype's state
    // +listType=map
    // +listMapKey=type
    // +optional
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

**Breaking Change:** No - Adding optional status is backwards compatible

**References:**
- [Kubernetes API Conventions - Spec and Status](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#spec-and-status)

---

### 17. PreferredCPUTopology Enum Could Use Validation

**Location:** `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:365`

**Issue:** The `PreferredCPUTopology` field lacks kubebuilder validation marker.

**Recommendation:**
```go
// preferredCPUTopology optionally defines the preferred guest visible CPU topology.
// Valid values are: "cores", "sockets", "threads", "spread", "any"
// Defaults to "sockets" when not specified.
// +kubebuilder:validation:Enum=cores;sockets;threads;spread;any
// +optional
PreferredCPUTopology *PreferredCPUTopology `json:"preferredCPUTopology,omitempty"`
```

**Breaking Change:** No - Adds validation for existing enum

---

### 18. SpreadAcross Enum Could Use Validation

**Location:** `staging/src/kubevirt.io/api/instancetype/v1beta1/types.go:395`

**Issue:** Missing kubebuilder validation marker.

**Recommendation:**
```go
// across optionally defines how to spread vCPUs across the guest visible topology.
// Valid values are: "SocketsCoresThreads", "SocketsCores", "CoresThreads"
// Default: "SocketsCores"
// +kubebuilder:validation:Enum=SocketsCoresThreads;SocketsCores;CoresThreads
// +optional
Across *SpreadAcross `json:"across,omitempty"`
```

**Breaking Change:** No

---

### 19. Documentation Could Be More User-Focused

**Location:** Throughout the file

**Issue:** Some documentation references Go struct names rather than focusing on user behavior and outcomes.

**Example from line 77:**
```go
// NodeSelector is the name of the custom node selector for the instancetype.
```

**Recommendation:**
```go
// nodeSelector allows you to constrain VirtualMachineInstances to nodes with specific labels.
// When a VirtualMachineInstance references this instancetype, the VMI will only be scheduled
// on nodes that match all the specified label selectors.
// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
// +optional
```

**Breaking Change:** No

**References:**
- [OpenShift API Conventions - Documentation](https://github.com/openshift/enhancements/blob/master/dev-guide/api-conventions.md#documentation)

---

### 20. Missing Examples in Documentation

**Location:** Throughout

**Issue:** Complex fields like `SpreadOptions` would benefit from examples in the documentation.

**Recommendation:**
```go
// spreadOptions optionally defines advanced vCPU spreading configuration.
//
// Example for 8 vCPUs with SocketsCores spread and ratio 2:
//   across: SocketsCores
//   ratio: 2
// Results in: 4 sockets × 2 cores = 8 vCPUs
//
// Example for 8 vCPUs with SocketsCoresThreads spread and ratio 2:
//   across: SocketsCoresThreads
//   ratio: 2
// Results in: 2 sockets × 2 cores × 2 threads = 8 vCPUs
//
// +optional
SpreadOptions *SpreadOptions `json:"spreadOptions,omitempty"`
```

**Breaking Change:** No

---

## Positive Findings ✅

The API demonstrates several strengths:

1. **Proper API Structure**: All resources correctly embed `TypeMeta` and `ObjectMeta`
2. **Clear Separation**: Good separation between Spec and preferences
3. **Cluster-Scoped Variants**: Proper implementation of both namespaced and cluster-scoped resources
4. **List Types**: Appropriate use of `+listType` markers
5. **Optional Fields**: Correct use of pointers for optional fields
6. **Enum Types**: Good use of string constants for enum values with clear const definitions
7. **Deprecation Markers**: Deprecated fields are properly marked (e.g., `DeprecatedPreferredUseEfi`)
8. **Resource References**: Uses `resource.Quantity` appropriately for memory values

---

## Recommendations Summary

### High Priority (for v2 or future major version)
1. Replace boolean fields with enums throughout the API
2. Convert `uint32` to `int32` with validation markers
3. Add Status subresources if lifecycle tracking is needed

### Medium Priority (can be done in minor versions)
1. Add missing kubebuilder validation markers
2. Improve documentation to be more user-focused
3. Add examples to complex field documentation
4. Fix all typos

### Low Priority
1. Consider adding usage examples in API documentation
2. Review field naming for consistency with JSON naming in docs

---

## OpenShift API Review Command Reference

📚 **Note:** If you're working with APIs in the [openshift/api](https://github.com/openshift/api) repository, there's a dedicated API review command available:

**Command:** `/api-review` in the openshift/api repository

This command provides OpenShift-specific API review tailored for that ecosystem. For more information, see the [openshift/api review documentation](https://github.com/openshift/api/blob/master/.claude/commands/api-review.md).

---

## References

- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
- [OpenShift API Conventions](https://github.com/openshift/enhancements/blob/master/dev-guide/api-conventions.md)
- [Kubebuilder Markers](https://book.kubebuilder.io/reference/markers.html)
- [Controller Runtime API](https://pkg.go.dev/sigs.k8s.io/controller-runtime)

---

## Appendix: Quick Reference

### Typos to Fix
- Line 276: "VirtualMachineInstace" → "VirtualMachineInstance"
- Line 291: "minium" → "minimum"
- Line 306: "prefeerred" → "preferred"
- Line 314: "PreffereedStorageClassName" → "PreferredStorageClassName" (Go field name only)
- Line 580: "transmitts" → "transmits"
- Line 624: "whih" → "which"

### Fields Needing Validation Markers
- Line 138: `Guest` (CPU count) - add `+kubebuilder:validation:Minimum=1`
- Line 168: `MaxSockets` - add `+kubebuilder:validation:Minimum=1`
- Line 365: `PreferredCPUTopology` - add enum validation
- Line 395: `Across` - add enum validation
- Line 406: `Ratio` - add range validation

### Boolean Fields to Consider Converting
- Line 151: `DedicatedCPUPlacement`
- Line 160: `IsolateEmulatorThread`
- Lines 415-456: Multiple device preference booleans
- Lines 577-585: BIOS-related booleans

---

**Overall Assessment:** The KubeVirt instancetype API is well-designed and follows most Kubernetes conventions. The main areas for improvement are around OpenShift-specific patterns (avoiding booleans) and adding more comprehensive validation markers. Most issues can be addressed in a future API version without breaking existing users.
