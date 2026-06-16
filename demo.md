# VEP 172: Cross-Architecture Virtualization Demo

## Summary

This demonstrates running an ARM64 guest VM on an AMD64 host using QEMU TCG software emulation, enabled by the `CrossArchitectureVirtualization` feature gate introduced in [PR #17199](https://github.com/kubevirt/kubevirt/pull/17199).

## Prerequisites

The virt-launcher image must be built with the cross-arch QEMU packages from the `@virtmaint-sig/virt-preview` COPR repo:

```bash
# Regenerate RPM dependencies with cross-arch packages
KUBEVIRT_CROSS_ARCH_EMULATION=true make rpm-deps

# Bring up the cluster and deploy
KUBEVIRT_CROSS_ARCH_EMULATION=true make cluster-down cluster-up cluster-sync
```

This pulls `qemu-system-aarch64-core` and `edk2-aarch64` into the virt-launcher image from the virt-preview COPR, enabling libvirt to launch aarch64 domains via TCG.

### Image caching workaround

There is a known issue ([#18170](https://github.com/kubevirt/kubevirt/issues/18170)) where `cluster-sync` does not force the kubevirtci nodes to re-pull the virt-launcher image. The virt-handler DaemonSet init container uses `imagePullPolicy: IfNotPresent`, so CRI-O reuses the locally cached image even after the registry is updated.

If after `cluster-sync` the node labels or VMI launch don't reflect the new cross-arch packages, flush the cached image on the node:

```bash
# Delete the virt-handler pod
cluster-up/kubectl.sh delete pod -n kubevirt -l kubevirt.io=virt-handler --wait=true

# SSH into the node and remove the cached image
cluster-up/ssh.sh node01 -- "sudo crictl ps -a | grep virt-launcher | awk '{print \$1}' | xargs -r sudo crictl rm; \
  sudo crictl images | grep virt-launcher | awk '{print \$3}' | xargs -r sudo crictl rmi"
```

The DaemonSet will recreate the pod and pull the fresh image from the registry. A fix for the underlying issue is tracked in [#18170](https://github.com/kubevirt/kubevirt/issues/18170).

## Enable the feature gate

```bash
cluster-up/kubectl.sh patch kubevirt kubevirt -n kubevirt --type=merge \
  -p '{"spec":{"configuration":{"developerConfiguration":{"featureGates":["CrossArchitectureVirtualization"]}}}}'
```

The node-labeller loads cross-arch capabilities unconditionally at startup (by probing `virsh domcapabilities` for the cross architecture via the init container). When the feature gate is toggled on, the config-modified callback triggers a re-labelling and the `kubevirt.io/vm-arch-arm64=true` label appears without needing to restart virt-handler:

```
$ cluster-up/kubectl.sh get nodes --show-labels | grep -oP 'kubevirt\.io/vm-arch[^ ,]*'
kubevirt.io/vm-arch-amd64=true
kubevirt.io/vm-arch-arm64=true
```

## Launch an ARM64 VMI

The container disk image must be the arm64 variant. In a kubevirtci dev cluster, the `cluster-build` step pushes arch-suffixed tags (e.g. `devel-arm64`) to the local registry when `KUBEVIRT_CROSS_ARCH_EMULATION` is set:

```yaml
apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: cross-arch-arm64-test
spec:
  architecture: arm64
  domain:
    resources:
      requests:
        memory: 1Gi
    devices:
      rng: {}
      disks:
      - name: disk0
        disk:
          bus: virtio
  volumes:
  - name: disk0
    containerDisk:
      image: registry:5000/kubevirt/fedora-with-test-tooling-container-disk:devel-arm64
      imagePullPolicy: IfNotPresent
```

Key points:
- `spec.architecture: arm64` tells KubeVirt to use the arm64 guest architecture.
- The container disk must contain an arm64 disk image. Multi-arch manifest resolution pulls the host architecture by default, so use an explicit arch-suffixed tag or image SHA.
- The scheduler uses a preferred node affinity (weight 100) for native arm64 nodes but falls back to any node with the `kubevirt.io/vm-arch-arm64=true` label.

## Verification

The VMI reaches `Running` phase on the amd64 node:

```
$ cluster-up/kubectl.sh get vmi cross-arch-arm64-test
NAME                    AGE   PHASE     IP             NODENAME   READY
cross-arch-arm64-test   27s   Running   10.244.0.116   node01     True
```

The `SoftwareEmulation` condition confirms TCG is in use:

```
$ cluster-up/kubectl.sh get vmi cross-arch-arm64-test -o jsonpath='{range .status.conditions[*]}{.type}: {.status} ({.reason}) {.message}{"\n"}{end}'
Ready: True ()
LiveMigratable: False (InterfaceNotLiveMigratable) cannot migrate VMI which does not use masquerade or a migratable plugin to connect to the pod network
StorageLiveMigratable: False (NotMigratable) InterfaceNotLiveMigratable: ...
SoftwareEmulation: True (CrossArchitectureEmulation) VM running with software emulation (host=amd64, guest=arm64)
```

The virt-launcher pod receives the `--allow-cross-arch-emulation` flag:

```
$ cluster-up/kubectl.sh get pod virt-launcher-cross-arch-arm64-test-<hash> -o yaml | grep allow-cross-arch
    - --allow-cross-arch-emulation
```

## Running the functional tests

The cross-architecture emulation functional tests are gated behind the `requires-cross-arch-emulation` label. The `cluster-sync` step above automatically cross-builds and pushes arch-suffixed container disk images (e.g. `devel-arm64`) to the local registry when `KUBEVIRT_CROSS_ARCH_EMULATION` is set.

Use `KUBEVIRT_E2E_FOCUS` and `KUBEVIRT_FUNC_TEST_LABEL_FILTER` to run only the cross-arch tests and filter out entries for architectures not present in the cluster:

```bash
KUBEVIRT_E2E_FOCUS='Cross-architecture software emulation' \
  KUBEVIRT_FUNC_TEST_LABEL_FILTER='!requires-arm64 && !requires-s390x' \
  make functest
```

On an amd64 host, two of the four tests run:

| Test | Result | Notes |
|------|--------|-------|
| arm64 guest on amd64 host | PASS | Cross-arch emulation via TCG — boots, logs in, verifies `uname -m` returns `aarch64` |
| amd64 guest on amd64 host | PASS | Native-arch with feature gate enabled — verifies KVM is used, no `SoftwareEmulation` condition |
| amd64 guest on arm64 host | SKIP | Requires ARM64 host hardware |
| arm64 guest on arm64 host | SKIP | Requires ARM64 host hardware |

## Known issues

- Performance is significantly reduced compared to native KVM — TCG is pure software emulation. Serial console is the primary interaction method as the minimal QEMU binary lacks virtio-gpu support.
- The virt-handler DaemonSet does not force re-pull of the virt-launcher image after `cluster-sync` ([#18170](https://github.com/kubevirt/kubevirt/issues/18170)). See the [image caching workaround](#image-caching-workaround) above.
- Container disk images must use an explicit arch-suffixed tag or SHA for the guest architecture. Multi-arch manifest resolution defaults to the host architecture, which pulls the wrong disk for cross-arch emulation. Improving this UX is planned for beta.
