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

package migration

// Regression test for a migration deadlock that occurs during a rolling virt-handler
// upgrade when the source node already runs the new handler and the target node still
// runs the previous-release handler.
//
// The migration completes at the QEMU/libvirt level, but the VMIM is permanently
// stuck in Running because:
//   - The old target handler sets TargetNodeDomainDetected=true via the legacy
//     vm.go migrationTargetUpdateVMIStatus code path.
//   - The new source handler's MigrationSourceController.hasTargetDetectedReadyDomain
//     checks TargetState.DomainDetected (nil for regular migrations) instead of the
//     legacy field, so it never sees the target as ready and never does the handoff.
//   - EndTimestamp is never set, so migrationNeedsFinalization on the target side is
//     also never satisfied.
//
// To reproduce: split the virt-handler DaemonSet so sourceNode runs the current image
// and targetNode runs the previous-release image, then trigger a migration.

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libmigration"
	libmonitoring "kubevirt.io/kubevirt/tests/libmonitoring"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

const (
	prevHandlerDSName = "virt-handler-prev"
	// prevHandlerVersionLabel is a unique label added to pods from the prev-release DS
	// so we can find them independently of the main virt-handler label.
	prevHandlerVersionLabel = "kubevirt.io/virt-handler-version"
	prevHandlerVersionValue = "prev"

	// stuckMigrationCheckDuration is how long we hold the Consistently assertion.
	// Long enough for QEMU migration to finish (~30s) with generous headroom, but
	// short enough not to make the suite too slow.
	stuckMigrationCheckDuration = 3 * time.Minute
	stuckMigrationCheckInterval = 10 * time.Second
)

var _ = Describe(SIG("virt-handler version skew during rolling upgrade", func() {
	// Serial: modifies cluster-wide DaemonSets and scales virt-operator.
	// RequiresTwoSchedulableNodes: we pin one node per handler version.
	Describe("Live migration with mixed virt-handler versions",
		decorators.RequiresTwoSchedulableNodes, Serial, func() {

			var (
				virtClient    kubecli.KubevirtClient
				sourceNode    string
				targetNode    string
				origHandlerDS *appsv1.DaemonSet
			)

			BeforeEach(func() {
				if flags.PreviousReleaseTag == "" {
					Skip("Skipping handler version-skew test: -previous-release-tag not set. " +
						"Pass e.g. -previous-release-tag=v1.5.3 -previous-release-registry=quay.io/kubevirt")
				}

				var err error
				virtClient, err = kubecli.GetKubevirtClient()
				Expect(err).NotTo(HaveOccurred())

				By("Selecting two schedulable nodes")
				nodes := libnode.GetAllSchedulableNodes(virtClient)
				Expect(len(nodes.Items)).To(BeNumerically(">=", 2),
					"at least two schedulable nodes are required for version-skew tests")
				sourceNode = nodes.Items[0].Name // runs the new (current) virt-handler
				targetNode = nodes.Items[1].Name // will run the old (previous) virt-handler

				By("Snapshotting the live virt-handler DaemonSet")
				origHandlerDS, err = virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).
					Get(context.Background(), "virt-handler", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				By("Scaling virt-operator to 0 so it does not reconcile our DaemonSet changes")
				scales := libmonitoring.NewScaling(virtClient, []string{"virt-operator"})
				scales.UpdateScale("virt-operator", 0)
				DeferCleanup(scales.RestoreAllScales)

				By(fmt.Sprintf("Restricting the existing virt-handler DS to %s (keeps current image)", sourceNode))
				restrictDSToNode(virtClient, "virt-handler", sourceNode)
				waitForHandlerPodRunning(virtClient, sourceNode, "kubevirt.io=virt-handler")

				By(fmt.Sprintf("Creating virt-handler-prev DS on %s with previous-release image", targetNode))
				createPrevHandlerDS(virtClient, origHandlerDS, targetNode)
				waitForHandlerPodRunning(virtClient, targetNode,
					fmt.Sprintf("kubevirt.io=virt-handler,%s=%s", prevHandlerVersionLabel, prevHandlerVersionValue))

				DeferCleanup(func() {
					By("Removing the previous-release virt-handler DaemonSet")
					_ = virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).
						Delete(context.Background(), prevHandlerDSName, metav1.DeleteOptions{})

					By("Restoring the original virt-handler DaemonSet")
					restoreDS(virtClient, origHandlerDS)
				})
			})

			// Once a fix lands in MigrationSourceController (falling back to the old
			// TargetNodeDomainDetected field when TargetState is nil), flip the
			// Consistently below to an Eventually expecting MigrationSucceeded and add
			// libmigration.ConfirmVMIPostMigration().
			FIt("should not permanently block VMIM when source has new and target has old virt-handler", func() {
				By(fmt.Sprintf("Creating VMI pinned to %s (managed by new virt-handler)", sourceNode))
				vmi := libvmifact.NewAlpineWithTestTooling(
					libnet.WithMasqueradeNetworking(),
					libvmi.WithNodeAffinityFor(sourceNode),
				)
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				By(fmt.Sprintf("Triggering live migration toward %s (managed by old virt-handler)", targetNode))
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigration(virtClient, migration)

				By("Waiting for QEMU-level migration to complete (targetNodeDomainDetected becomes true)")
				Eventually(func() bool {
					vmiObj, err := virtClient.VirtualMachineInstance(vmi.Namespace).
						Get(context.Background(), vmi.Name, metav1.GetOptions{})
					if err != nil {
						return false
					}
					return vmiObj.Status.MigrationState != nil &&
						vmiObj.Status.MigrationState.TargetNodeDomainDetected
				}, 2*time.Minute, 5*time.Second).Should(BeTrue(),
					"QEMU migration must succeed: old target handler should detect the domain")

				By("Asserting VMIM remains permanently stuck in Running (version-skew regression check)")
				Consistently(func(g Gomega) {
					m, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).
						Get(context.Background(), migration.Name, metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(m.Status.Phase).To(Equal(v1.MigrationRunning),
						"VMIM must stay in Running — the handoff handshake is broken across handler versions")

					vmiObj, err := virtClient.VirtualMachineInstance(vmi.Namespace).
						Get(context.Background(), vmi.Name, metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(vmiObj.Status.MigrationState).NotTo(BeNil())
					g.Expect(vmiObj.Status.MigrationState.EndTimestamp).To(BeNil(),
						"EndTimestamp must not be set: new source handler checks TargetState.DomainDetected "+
							"which the old target handler never sets")
					g.Expect(vmiObj.Status.MigrationState.Completed).To(BeFalse(),
						"migration must not be marked completed")
				}, stuckMigrationCheckDuration, stuckMigrationCheckInterval)

				By("Applying manual remediation: patching VMI labels to reflect target ownership")
				labelPatch, err := json.Marshal(map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"kubevirt.io/nodeName":                targetNode,
							"kubevirt.io/outdatedLauncherImage":   nil,
							"kubevirt.io/migrationTargetNodeName": nil,
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())
				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).
					Patch(context.Background(), vmi.Name, types.MergePatchType, labelPatch, metav1.PatchOptions{})
				Expect(err).NotTo(HaveOccurred())

				By("Patching VMI status: completing migration and updating nodeName to target")
				statusPatch, err := json.Marshal(map[string]interface{}{
					"status": map[string]interface{}{
						"nodeName":                      targetNode,
						"launcherContainerImageVersion": "",
						"migrationState": map[string]interface{}{
							"completed":    true,
							"endTimestamp": time.Now().UTC().Format(time.RFC3339),
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())
				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).
					Patch(context.Background(), vmi.Name, types.MergePatchType, statusPatch, metav1.PatchOptions{}, "status")
				Expect(err).NotTo(HaveOccurred())

				By("Asserting VMIM transitions to Succeeded after manual remediation")
				Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
					m, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).
						Get(context.Background(), migration.Name, metav1.GetOptions{})
					if err != nil {
						return ""
					}
					return m.Status.Phase
				}, 2*time.Minute, 5*time.Second).Should(Equal(v1.MigrationSucceeded),
					"VMIM should transition to Succeeded after manual label/status remediation")

				By("Confirming VMI nodeName reflects the target node")
				vmiObj, err := virtClient.VirtualMachineInstance(vmi.Namespace).
					Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(vmiObj.Status.NodeName).To(Equal(targetNode))
			})
		})
}))

// restrictDSToNode patches the named DaemonSet so it only schedules on nodeName
// by setting kubernetes.io/hostname in its nodeSelector.
func restrictDSToNode(virtClient kubecli.KubevirtClient, dsName, nodeName string) {
	ctx := context.Background()
	ns := flags.KubeVirtInstallNamespace

	ds, err := virtClient.AppsV1().DaemonSets(ns).Get(ctx, dsName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	if ds.Spec.Template.Spec.NodeSelector == nil {
		ds.Spec.Template.Spec.NodeSelector = map[string]string{}
	}
	ds.Spec.Template.Spec.NodeSelector["kubernetes.io/hostname"] = nodeName

	patchBytes, err := json.Marshal(ds)
	Expect(err).NotTo(HaveOccurred())

	_, err = virtClient.AppsV1().DaemonSets(ns).
		Patch(ctx, dsName, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	Expect(err).NotTo(HaveOccurred())
}

// createPrevHandlerDS creates a second virt-handler DaemonSet using the previous-release
// image pinned to targetNode. It deep-copies the original DS so that the service account,
// volumes, environment, and security context are all preserved without extra RBAC work.
func createPrevHandlerDS(virtClient kubecli.KubevirtClient, orig *appsv1.DaemonSet, targetNode string) {
	ctx := context.Background()
	ns := flags.KubeVirtInstallNamespace

	prevDS := orig.DeepCopy()
	prevDS.Name = prevHandlerDSName
	prevDS.ResourceVersion = ""
	prevDS.UID = ""
	prevDS.OwnerReferences = nil
	prevDS.Annotations = nil

	// Pin to the target node only.
	if prevDS.Spec.Template.Spec.NodeSelector == nil {
		prevDS.Spec.Template.Spec.NodeSelector = map[string]string{}
	}
	prevDS.Spec.Template.Spec.NodeSelector["kubernetes.io/hostname"] = targetNode

	// Use the previous-release virt-handler image.
	prevDS.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf(
		"%s/virt-handler:%s", flags.PreviousReleaseRegistry, flags.PreviousReleaseTag)

	// Add a unique label to both the selector and the pod template so we can
	// distinguish these pods from those of the primary virt-handler DS.
	if prevDS.Spec.Selector == nil {
		prevDS.Spec.Selector = &metav1.LabelSelector{}
	}
	if prevDS.Spec.Selector.MatchLabels == nil {
		prevDS.Spec.Selector.MatchLabels = map[string]string{}
	}
	prevDS.Spec.Selector.MatchLabels[prevHandlerVersionLabel] = prevHandlerVersionValue

	if prevDS.Spec.Template.Labels == nil {
		prevDS.Spec.Template.Labels = map[string]string{}
	}
	prevDS.Spec.Template.Labels[prevHandlerVersionLabel] = prevHandlerVersionValue

	_, err := virtClient.AppsV1().DaemonSets(ns).Create(ctx, prevDS, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
}

// waitForHandlerPodRunning polls until at least one pod matching labelSelector
// is fully Ready on nodeName.
func waitForHandlerPodRunning(virtClient kubecli.KubevirtClient, nodeName, labelSelector string) {
	ctx := context.Background()
	ns := flags.KubeVirtInstallNamespace

	Eventually(func() bool {
		pods, err := virtClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
			FieldSelector: "spec.nodeName=" + nodeName,
		})
		if err != nil || len(pods.Items) == 0 {
			return false
		}
		for i := range pods.Items {
			if isPodReady(&pods.Items[i]) {
				return true
			}
		}
		return false
	}, 3*time.Minute, 5*time.Second).Should(BeTrue(),
		fmt.Sprintf("expected a ready virt-handler pod with labels %q on node %s", labelSelector, nodeName))
}

func isPodReady(pod *k8sv1.Pod) bool {
	if pod.Status.Phase != k8sv1.PodRunning {
		return false
	}
	for _, c := range pod.Status.ContainerStatuses {
		if !c.Ready {
			return false
		}
	}
	return true
}

// restoreDS merge-patches the original DaemonSet back, removing our nodeSelector addition.
func restoreDS(virtClient kubecli.KubevirtClient, orig *appsv1.DaemonSet) {
	ctx := context.Background()
	ns := flags.KubeVirtInstallNamespace

	restore := orig.DeepCopy()
	restore.ResourceVersion = ""

	patchBytes, err := json.Marshal(restore)
	if err != nil {
		return
	}
	_, _ = virtClient.AppsV1().DaemonSets(ns).
		Patch(ctx, orig.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
}
