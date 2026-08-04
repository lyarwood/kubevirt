# API Design Guidelines Audit: Current State vs. Guidelines

This document audits the existing VM, VMI, and instancetype APIs against the
proposed [API Design Guidelines](https://github.com/kubevirt/kubevirt/pull/18612).
The goal is to quantify the gap between the guidelines' stated rules and current
reality, and to frame the guidelines as aspirational targets for new API work
rather than descriptions of current practice.

## Context

PR [#18612](https://github.com/kubevirt/kubevirt/pull/18612) introduces
`docs/api/design_guidelines.md` - a comprehensive reference codifying API
design decisions. The document uses strong normative language ("must",
"strictly forbidden", "must be avoided") throughout. This audit applies those
rules to the existing `core/v1` and `instancetype/v1beta1` APIs to identify
where the current codebase does not conform.

The findings suggest that a VEP would be valuable to frame these guidelines as
target-state documentation, provide a roadmap for retrofitting existing APIs,
and establish priorities for which violations to address first.

---

## 1. Backend Agnosticism (Guideline Section 1.1)

### Field name violations

The only cases where a backend name leaks into an actual field name (as opposed
to comments):

| Location | Field | JSON key | Issue |
|---|---|---|---|
| `schema.go` | `QemuGuestAgentSSHPublicKeyAccessCredentialPropagation` | `qemuGuestAgent` | Go type and JSON key embed "qemu" in public API |
| `schema.go` | `QemuGuestAgentUserPasswordAccessCredentialPropagation` | `qemuGuestAgent` | Same |

A backend-neutral name like `GuestAgent` would comply with Section 1.1.

### Backend references in API doc comments

Extensive QEMU/libvirt references in doc comments across core and instancetype
types. These do not affect the wire format but leak implementation details into
the API documentation surface:

- Disk/IO comments: "QEMU disk IO mode" (schema.go, instancetype types.go)
- Timer comments: "what happens when QEMU misses a deadline" (6 occurrences)
- Volume source comments: "Directly attached to the vmi via qemu" (4 occurrences)
- CPU model: Links to `github.com/libvirt/libvirt/tree/master/src/cpu_map` as
  canonical list (schema.go, instancetype types.go)
- Machine type: "QEMU machine type is the actual chipset" (schema.go)
- Cache: "kvm disk cache mode" (schema.go)
- Migration state: "source libvirt domain" (types.go)
- Condition comments: "QEMU guest agent" in AgentConnected (types.go)
- Guest agent probes: "executed on the guest through the qemu-guest-agent" (types.go)
- Firmware: "OVMF roms" in deprecated SecureBoot field (instancetype types.go)

### Conformant justified exceptions

- `KVMTimer` / `FeatureKVM` - KVM clock is a named paravirtualized clock source
- `kubevirt.io/libvirt-log-filters` annotation - debug escape hatch
- `HypervisorConfiguration.Name` values - field exists to select a backend
- `PreferredKvm` / `PreferredHyperv` - guest-visible CPU feature domains

---

## 2. Data Primitives (Guideline Section 3.1)

### uint32/uint64 technical debt

The guidelines state: "Unsigned integers (uint32/uint64) exist in the API as
technical debt and must not be used in new fields."

**Core VM/VMI types:**

| Location | Field | Type |
|---|---|---|
| `schema.go` | `CPU.Cores` | `uint32` |
| `schema.go` | `CPU.Sockets` | `uint32` |
| `schema.go` | `CPU.MaxSockets` | `uint32` |
| `schema.go` | `CPU.Threads` | `uint32` |
| `schema.go` | `Disk.BootOrder` | `*uint` |
| `schema.go` | `Interface.BootOrder` | `*uint` |
| `schema.go` | `CustomBlockSize.Logical` | `uint` |
| `schema.go` | `CustomBlockSize.Physical` | `uint` |
| `schema.go` | `FeatureSpinlocks.Retries` | `*uint32` |
| `types.go` | `VMIStatus.RuntimeUser` | `uint64` |
| `types.go` | `VMIStatus.VSOCKCID` | `*uint32` |
| `types.go` | `VSOCKOptions.TargetPort` | `uint32` |

Additionally, multiple KubeVirt configuration fields use unsigned types:
`MemBalloonStatsPeriod`, `ParallelOutbound/ParallelMigrations`,
`MaxHotplugRatio`, `MaxCpuSockets`, `MaxDowntimeMs` (uint64), etc.

**Instancetype types:**

| Location | Field | Type |
|---|---|---|
| instancetype `types.go` | `CPUInstancetype.Guest` | `uint32` |
| instancetype `types.go` | `CPUInstancetype.MaxSockets` | `*uint32` |
| instancetype `types.go` | `SpreadOptions.Ratio` | `*uint32` |
| instancetype `types.go` | `CPUPreferenceRequirement.Guest` | `uint32` |
| instancetype `types.go` | `MemoryInstancetype.OvercommitPercent` | bare `int` (should be `int32`) |

### Float violations

The guidelines state: "Raw floating-point numbers are strictly forbidden within
the public API spec."

| Location | Field | Type | Issue |
|---|---|---|---|
| `types.go` | `TokenBucketRateLimiter.QPS` | `float32` | Should use milli-units |
| `types.go` | `VirtualMachineInstanceGuestOSUser.LoginTime` | `float64` | Float in status |

### Size fields not using resource.Quantity

The guidelines state: "Any field representing size, memory allocation, or
storage limits must use resource.Quantity."

| Location | Field | Type |
|---|---|---|
| `types.go` | `VolumeStatus.Size` | `int64` |
| `types.go` | `VirtualMachineInstanceFileSystem.UsedBytes` | `int` |
| `types.go` | `VirtualMachineInstanceFileSystem.TotalBytes` | `int` |

---

## 3. Hard Enums (Guideline Section 3.2)

The guidelines state: "the use of hard OpenAPI enum arrays for schema validation
must be avoided on fields representing extensible lists of choices."

### Violations in core/v1

| Location | Field | Values | Issue |
|---|---|---|---|
| `schema.go` | `RebootPolicy` | `Reboot;Terminate` | Could gain future policies |
| `types.go` | `VMRolloutStrategy` | `Stage;LiveUpdate` | Extensible strategy |
| `types.go` | `InstancetypeReferencePolicy` | `reference;expand;expandAll` | Extensible policy |
| `types.go` | `MigrationCompression` | `none;zstd` | New algorithms could be added |
| `types.go` | `RoleAggregationStrategy` | `AggregateToDefault;Manual` | Borderline |
| `types.go` | `HypervisorConfiguration.Name` | `kvm;hyperv-direct` | Cited as justified in Section 1.1 but contradicts Section 3.2 |

### Violations in other API packages

| Package | Field | Values |
|---|---|---|
| `pool/v1alpha1,v1beta1` | `StatePreservation` | `Disabled;Offline;Online` |
| `pool/v1alpha1,v1beta1` | `SortPolicy` | 5 values |
| `clone/v1beta1` | `VolumeNamePolicy` | `RandomizeNames;PrefixTargetName` |
| `backup/v1alpha1` | `Mode` | `Push;Pull` |
| `snapshot/v1beta1` | `TargetReadinessPolicy` | 4 values |

### Conformant

Many choice fields correctly use open strings validated in webhooks:
RunStrategy, IOThreadsPolicy, DiskBus, InputBus, SoundModel, etc. The
instancetype API is fully conformant - all preference fields use open string
aliases with no enum markers.

---

## 4. Immutable Field Enforcement (Guideline Section 3.3)

The guidelines state: hardware identity fields "should have CEL immutability
rules with `self == oldSelf`" and provides the exact pattern to use.

### Current state

**None** of the hardware identity fields have CEL immutability rules:

| Field | Location | CEL rule? | Webhook protection? |
|---|---|---|---|
| `Firmware.UUID` | `schema.go` | No | No |
| `Firmware.Serial` | `schema.go` | No | No |
| `Chassis.Serial` | `schema.go` | No | No |
| `Interface.MacAddress` | `schema.go` | No | No |

The VM validating webhook has no update-specific field immutability enforcement
at all. Users can freely change firmware UUID, SMBIOS serial, and MAC addresses
on existing VMs with no validation blocking the change.

The VMI update admitter rejects all spec changes from non-KubeVirt service
accounts (blanket protection), but this is on the VMI, not the VM where users
edit their configuration.

The `backup/v1alpha1` API already uses `self == oldSelf` CEL rules correctly,
showing the project knows how to apply this pattern.

---

## 5. CEL Validation Adoption (Guideline Section 4.2)

The guidelines state: cross-field validation "must prioritize CEL rules" with
webhooks as a "secondary fallback reserved exclusively for highly dynamic,
external database lookups."

### Current state

Only **3 XValidation markers** exist in all of `core/v1`, all on a single
newly-added struct (`ContainerPathVolumeSource`). The webhook admitters contain:

- ~28 cross-field validations in VMI create admitter (all CEL-expressible)
- ~9 structural invariants (exactly-one-of checks, mutual exclusion)
- ~19 simple field constraints

Examples of CEL-expressible checks currently in webhooks:

- `DedicatedCPUPlacement` requires `IsolateEmulatorThread`
- `SecureBoot` requires `SMM`
- Disk backend mutual exclusivity (the guidelines use this exact example)
- `RunStrategy` vs `Running` mutual exclusion
- Memory request must equal limit when dedicated CPU is set

~26 dynamic lookups (feature gates, cluster config, instancetype resolution)
are appropriately in webhooks and justified by the guidelines.

### Instancetype API

Also has zero CEL rules, though cross-field relationships exist (e.g.,
`MaxGuest >= Guest`, SecureBoot requires EFI).

---

## 6. Status and Conditions (Guideline Section 6)

### Custom condition structs

All condition types use custom structs without per-condition
`observedGeneration`. Migration to `metav1.Condition` is tracked by VEP #376.

- `VirtualMachineCondition`
- `VirtualMachineInstanceCondition`
- `VirtualMachineInstanceMigrationCondition`
- `VirtualMachineInstanceReplicaSetCondition`
- `KubeVirtCondition`

### Phase-based state machines

The guidelines state: "Do not introduce Phase string fields or linear
state-machine tracking in new API resources." Only `vmi.Status.Phase` is
acknowledged as a historical exception.

Phase-like patterns added after `vmi.Status.Phase` that are not covered by the
historical exception:

| Type | Location | Values |
|---|---|---|
| `VolumePhase` | `types.go` | 10 phases |
| `MemoryDumpPhase` | `types.go` | 6 phases |
| `ChangedBlockTrackingState` | `types.go` | 6 states |
| `VirtualMachineInstanceMigrationPhase` | `types.go` | 10 phases |
| `VirtualMachinePrintableStatus` | `types.go` | 16 values (display-oriented) |

### observedGeneration

- `VirtualMachineStatus` has it - conformant
- `VirtualMachineInstanceStatus` does not - acknowledged gap

### VMI /status subresource

VMI has `+genclient:noStatus` - no `/status` subresource. Both the VMI
controller and virt-handler write to `vmi.Status`. Acknowledged architectural
tension per Section 6.6.2.

---

## 7. Server-Side Apply

The guidelines reference server-side apply (SSA) in Section 1.2 for field
ownership. KubeVirt does not yet use SSA - this is tracked by
[VEP 389](https://github.com/kubevirt/enhancements/issues/389).

---

## Summary

| Guideline Section | Status | Count of Issues |
|---|---|---|
| 1.1 Backend agnosticism (field names) | 2 field name violations, ~20 comment references | Moderate |
| 3.1 Data primitives (uint) | ~17 uint32/uint64 fields | High |
| 3.1 Data primitives (float) | 2 float fields | Low |
| 3.1 Data primitives (Quantity) | 3 size fields not using Quantity | Low |
| 3.2 Hard enums | 6 in core/v1, 5+ in other packages | High |
| 3.3 Immutable fields | 4 unprotected identity fields, 0 CEL rules | High |
| 4.2 CEL validation | 3 CEL rules vs ~37 webhook checks that could be CEL | High |
| 6.1 Phase state machines | 5 post-VMI phase patterns | Moderate |
| 6.5/6.7 Conditions/observedGeneration | Custom structs, VEP #376 tracked | Tracked |
| 6.6 /status subresource | VMI lacks it | Tracked |
| SSA | Not implemented, VEP 389 | Tracked |

### Recommendation

The guidelines document codifies valuable design knowledge and would serve the
project well as a reference. However, introducing it as a normative document
without acknowledging the current state risks confusion - contributors reading
"must" rules will find the existing APIs violate them extensively.

A VEP would provide the right framing:

1. Establish the guidelines as the target state for new API work
2. Categorize existing violations as acknowledged technical debt
3. Prioritize which violations to address (e.g., immutable fields and CEL
   validation are high-impact improvements; uint32 is low-risk debt)
4. Define a migration path for retrofitting existing APIs
5. Track progress toward conformance
