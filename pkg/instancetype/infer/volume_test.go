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

package infer_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"context"
	"errors"
	"fmt"

	"go.uber.org/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cdifake "kubevirt.io/client-go/containerized-data-importer/fake"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"kubevirt.io/kubevirt/pkg/instancetype/infer"
)

func TestVolumeInference(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Volume Inference Suite")
}

var _ = Describe("Volume Inference", func() {
	Context("fromLabels function", func() {
		It("should return defaultName and defaultKind when both labels are present", func() {
			labels := map[string]string{"nameLabel": "test-name", "kindLabel": "test-kind"}
			defaultNameLabel := "nameLabel"
			defaultKindLabel := "kindLabel"

			defaultName, defaultKind, err := infer.FromLabelsTestWrangler(labels, defaultNameLabel, defaultKindLabel)

			Expect(err).ToNot(HaveOccurred())
			Expect(defaultName).To(Equal("test-name"))
			Expect(defaultKind).To(Equal("test-kind"))
		})

		It("should return IgnoreableInferenceError when defaultNameLabel is missing", func() {
			labels := map[string]string{"kindLabel": "test-kind"}
			defaultNameLabel := "nameLabel"
			defaultKindLabel := "kindLabel"

			defaultName, defaultKind, err := infer.FromLabelsTestWrangler(labels, defaultNameLabel, defaultKindLabel)

			Expect(err).To(HaveOccurred())
			var ignoreableErr *infer.IgnoreableInferenceError
			Expect(errors.As(err, &ignoreableErr)).To(BeTrue())
			Expect(err.Error()).To(Equal(fmt.Sprintf(infer.MissingLabelFmtTestWrangler, defaultNameLabel)))
			Expect(defaultName).To(BeEmpty())
			Expect(defaultKind).To(BeEmpty())
		})

		It("should return defaultName and empty defaultKind when defaultKindLabel is missing", func() {
			labels := map[string]string{"nameLabel": "test-name"}
			defaultNameLabel := "nameLabel"
			defaultKindLabel := "kindLabel"

			defaultName, defaultKind, err := infer.FromLabelsTestWrangler(labels, defaultNameLabel, defaultKindLabel)

			Expect(err).ToNot(HaveOccurred())
			Expect(defaultName).To(Equal("test-name"))
			Expect(defaultKind).To(BeEmpty())
		})
	})

	Context("fromPVC function", func() {
		var (
			ctrl           *gomock.Controller
			virtClient     *kubecli.MockKubevirtClient
			fakeClientset  *fake.Clientset
			testHandler    *infer.Handler // Use the exported type from infer
			pvcName        string
			pvcNamespace   string
			nameLabel      string
			kindLabel      string
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			fakeClientset = fake.NewSimpleClientset()
			virtClient.EXPECT().CoreV1().Return(fakeClientset.CoreV1()).AnyTimes()
			// Ensure infer.New is accessible, if it's not, we might need to adjust visibility or use a constructor from the package
			// For now, assuming infer.New is available and returns an unexported type that can be cast or used if it implements an interface
			// If infer.handler is not exported, this line needs to be infer.New(virtClient) which returns *infer.handler
			// And then FromPVCTestWrangler should be called on this handler instance.
			// The type infer.Handler should be what infer.New returns.
			// Let's assume infer.New returns an interface or a concrete type that has FromPVCTestWrangler
			// Based on previous steps, FromPVCTestWrangler is a method on the *handler type.
			// The New function `func New(virtClient kubecli.KubevirtClient) *handler` is in the infer package.
			// So we use infer.New(virtClient) to create the handler.
			testHandler = infer.New(virtClient)

			pvcName = "test-pvc"
			pvcNamespace = "test-namespace"
			nameLabel = "instancetype.kubevirt.io/default-instancetype"
			kindLabel = "instancetype.kubevirt.io/default-instancetype-kind"
		})

		It("should return defaultName and defaultKind when PVC is found and labels are present", func() {
			pvc := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pvcName,
					Namespace: pvcNamespace,
					Labels: map[string]string{
						nameLabel: "test-name",
						kindLabel: "test-kind",
					},
				},
			}
			_, err := fakeClientset.CoreV1().PersistentVolumeClaims(pvcNamespace).Create(context.Background(), pvc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			defaultName, defaultKind, err := testHandler.FromPVCTestWrangler(pvcName, pvcNamespace, nameLabel, kindLabel)
			Expect(err).ToNot(HaveOccurred())
			Expect(defaultName).To(Equal("test-name"))
			Expect(defaultKind).To(Equal("test-kind"))
		})

		It("should return IgnoreableInferenceError when PVC is found but defaultNameLabel is missing", func() {
			pvc := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pvcName,
					Namespace: pvcNamespace,
					Labels: map[string]string{
						kindLabel: "test-kind",
					},
				},
			}
			_, err := fakeClientset.CoreV1().PersistentVolumeClaims(pvcNamespace).Create(context.Background(), pvc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			defaultName, defaultKind, err := testHandler.FromPVCTestWrangler(pvcName, pvcNamespace, nameLabel, kindLabel)
			Expect(err).To(HaveOccurred())
			var ignoreableErr *infer.IgnoreableInferenceError
			Expect(errors.As(err, &ignoreableErr)).To(BeTrue(), "error should be of type IgnoreableInferenceError")
			Expect(err.Error()).To(Equal(fmt.Sprintf(infer.MissingLabelFmtTestWrangler, nameLabel)))
			Expect(defaultName).To(BeEmpty())
			Expect(defaultKind).To(BeEmpty())
		})

		It("should return defaultName and empty defaultKind when PVC is found but defaultKindLabel is missing", func() {
			pvc := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pvcName,
					Namespace: pvcNamespace,
					Labels: map[string]string{
						nameLabel: "test-name",
					},
				},
			}
			_, err := fakeClientset.CoreV1().PersistentVolumeClaims(pvcNamespace).Create(context.Background(), pvc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			defaultName, defaultKind, err := testHandler.FromPVCTestWrangler(pvcName, pvcNamespace, nameLabel, kindLabel)
			Expect(err).ToNot(HaveOccurred())
			Expect(defaultName).To(Equal("test-name"))
			Expect(defaultKind).To(BeEmpty())
		})

		It("should return an error when PVC is not found", func() {
			// Ensure no PVC with pvcName exists by creating a new fake client or by deleting if it exists
			// For this test, we can also just react to the Get call failing.
			// fakeClientset.PrependReactor("get", "persistentvolumeclaims", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			// 	return true, nil, k8serrors.NewNotFound(k8sv1.Resource("persistentvolumeclaims"), pvcName)
			// })
			// The above reactor is more precise, but simply not creating the PVC is sufficient with a fresh fakeClientset

			defaultName, defaultKind, err := testHandler.FromPVCTestWrangler(pvcName, pvcNamespace, nameLabel, kindLabel)
			Expect(err).To(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue(), "error should be a NotFound error")
			Expect(defaultName).To(BeEmpty())
			Expect(defaultKind).To(BeEmpty())
		})
	})

	Context("fromDataVolume function", func() {
		var (
			ctrl             *gomock.Controller
			virtClient       *kubecli.MockKubevirtClient
			fakeClientset    *fake.Clientset
			cdiFakeClientset *cdifake.Clientset // Correct type for CDI fake client
			testHandler      *infer.Handler
			vm               *virtv1.VirtualMachine
			dvName           string
			pvcName          string
			vmNamespace      string
			nameLabel        string
			kindLabel        string
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)
			fakeClientset = fake.NewSimpleClientset()
			cdiFakeClientset = cdifake.NewSimpleClientset() // Initialize CDI fake client

			virtClient.EXPECT().CoreV1().Return(fakeClientset.CoreV1()).AnyTimes()
			virtClient.EXPECT().CdiClient().Return(cdiFakeClientset).AnyTimes() // Mock CdiClient

			testHandler = infer.New(virtClient)

			vmNamespace = "test-namespace"
			dvName = "test-dv"
			pvcName = "test-pvc-for-dv" // Different name to avoid conflict with dvName unless intended
			nameLabel = "instancetype.kubevirt.io/default-instancetype"
			kindLabel = "instancetype.kubevirt.io/default-instancetype-kind"

			vm = &virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vmNamespace,
				},
				Spec: virtv1.VirtualMachineSpec{ // Ensure Spec is not nil if DataVolumeTemplates might be accessed
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{},
				},
			}
		})

		It("should return defaults when DataVolume is found and has labels", func() {
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dvName,
					Namespace: vmNamespace,
					Labels: map[string]string{
						nameLabel: "dv-name",
						kindLabel: "dv-kind",
					},
				},
			}
			_, err := cdiFakeClientset.CdiV1beta1().DataVolumes(vmNamespace).Create(context.Background(), dv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			defaultName, defaultKind, err := testHandler.FromDataVolumeTestWrangler(vm, dvName, nameLabel, kindLabel)
			Expect(err).ToNot(HaveOccurred())
			Expect(defaultName).To(Equal("dv-name"))
			Expect(defaultKind).To(Equal("dv-kind"))
		})

		It("should return defaults from underlying PVC when DataVolume is found without labels but Spec.Source.PVC is set", func() {
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{Name: dvName, Namespace: vmNamespace},
				Spec: cdiv1.DataVolumeSpec{
					Source: &cdiv1.DataVolumeSource{
						PVC: &cdiv1.DataVolumeSourcePVC{Name: pvcName, Namespace: vmNamespace},
					},
				},
			}
			pvc := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pvcName,
					Namespace: vmNamespace,
					Labels:    map[string]string{nameLabel: "pvc-name", kindLabel: "pvc-kind"},
				},
			}
			_, err := cdiFakeClientset.CdiV1beta1().DataVolumes(vmNamespace).Create(context.Background(), dv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			_, err = fakeClientset.CoreV1().PersistentVolumeClaims(vmNamespace).Create(context.Background(), pvc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			defaultName, defaultKind, err := testHandler.FromDataVolumeTestWrangler(vm, dvName, nameLabel, kindLabel)
			Expect(err).ToNot(HaveOccurred())
			Expect(defaultName).To(Equal("pvc-name"))
			Expect(defaultKind).To(Equal("pvc-kind"))
		})

		It("should return IgnoreableInferenceError if DV has no labels and underlying PVC is missing nameLabel", func() {
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{Name: dvName, Namespace: vmNamespace},
				Spec: cdiv1.DataVolumeSpec{
					Source: &cdiv1.DataVolumeSource{
						PVC: &cdiv1.DataVolumeSourcePVC{Name: pvcName, Namespace: vmNamespace},
					},
				},
			}
			pvc := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pvcName,
					Namespace: vmNamespace,
					Labels:    map[string]string{kindLabel: "pvc-kind-only"}, // Missing nameLabel
				},
			}
			_, err := cdiFakeClientset.CdiV1beta1().DataVolumes(vmNamespace).Create(context.Background(), dv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			_, err = fakeClientset.CoreV1().PersistentVolumeClaims(vmNamespace).Create(context.Background(), pvc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			defaultName, defaultKind, err := testHandler.FromDataVolumeTestWrangler(vm, dvName, nameLabel, kindLabel)
			Expect(err).To(HaveOccurred())
			var ignoreableErr *infer.IgnoreableInferenceError
			Expect(errors.As(err, &ignoreableErr)).To(BeTrue())
			Expect(err.Error()).To(Equal(fmt.Sprintf(infer.MissingLabelFmtTestWrangler, nameLabel)))
			Expect(defaultName).To(BeEmpty())
			Expect(defaultKind).To(BeEmpty())
		})

		It("should return defaults from PVC with same name when DataVolume is not found (garbage collected)", func() {
			// Do not create DV
			pvc := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dvName, // PVC name is the same as the expected DV name
					Namespace: vmNamespace,
					Labels:    map[string]string{nameLabel: "pvc-fallback-name", kindLabel: "pvc-fallback-kind"},
				},
			}
			_, err := fakeClientset.CoreV1().PersistentVolumeClaims(vmNamespace).Create(context.Background(), pvc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Simulate DV not found by reacting to the Get call for DataVolumes
			cdiFakeClientset.PrependReactor("get", "datavolumes", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, nil, k8serrors.NewNotFound(cdiv1.Resource("datavolumes"), dvName)
			})

			defaultName, defaultKind, err := testHandler.FromDataVolumeTestWrangler(vm, dvName, nameLabel, kindLabel)
			Expect(err).ToNot(HaveOccurred())
			Expect(defaultName).To(Equal("pvc-fallback-name"))
			Expect(defaultKind).To(Equal("pvc-fallback-kind"))
		})

		It("should return error when DataVolume is not found and no corresponding PVC is found", func() {
			// Simulate DV not found
			cdiFakeClientset.PrependReactor("get", "datavolumes", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, nil, k8serrors.NewNotFound(cdiv1.Resource("datavolumes"), dvName)
			})
			// Simulate PVC also not found
			fakeClientset.PrependReactor("get", "persistentvolumeclaims", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, nil, k8serrors.NewNotFound(k8sv1.Resource("persistentvolumeclaims"), dvName)
			})

			defaultName, defaultKind, err := testHandler.FromDataVolumeTestWrangler(vm, dvName, nameLabel, kindLabel)
			Expect(err).To(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
			Expect(defaultName).To(BeEmpty())
			Expect(defaultKind).To(BeEmpty())
		})

		It("should return error from fromDataVolumeSpec if DV has no labels and fromDataVolumeSpec fails", func() {
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{Name: dvName, Namespace: vmNamespace},
				Spec: cdiv1.DataVolumeSpec{ // No Source.PVC and no SourceRef
					Source: &cdiv1.DataVolumeSource{},
				},
			}
			_, err := cdiFakeClientset.CdiV1beta1().DataVolumes(vmNamespace).Create(context.Background(), dv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			defaultName, defaultKind, err := testHandler.FromDataVolumeTestWrangler(vm, dvName, nameLabel, kindLabel)
			Expect(err).To(HaveOccurred())
			var ignoreableErr *infer.IgnoreableInferenceError
			Expect(errors.As(err, &ignoreableErr)).To(BeTrue())
			Expect(err.Error()).To(Equal(infer.UnsupportedDataVolumeSourceTestWrangler))
			Expect(defaultName).To(BeEmpty())
			Expect(defaultKind).To(BeEmpty())
		})
	})

	Context("fromDataVolumeSpec function", func() {
		var (
			// Re-use most of the setup from fromDataVolume, ensure vmNamespace is available
			// ctrl, virtClient, fakeClientset, cdiFakeClientset, testHandler already available from outer context if BeforeEach is structured per-Context
			// For clarity, let's assume they are scoped here or passed down if using nested Describes
			// However, Ginkgo's BeforeEach applies to the current and nested Contexts/Describes, so they should be available.
			// We will still need to define variables specific to this context.
			pvcName        string
			dataSourceName string
			nameLabel      string
			kindLabel      string
			// vmNamespace is already defined in the outer context's BeforeEach
		)

		BeforeEach(func() {
			// Variables specific to fromDataVolumeSpec tests
			pvcName = "test-spec-pvc"
			dataSourceName = "test-spec-ds"
			nameLabel = "instancetype.kubevirt.io/default-instancetype" // Same as outer context, but can be redefined if needed
			kindLabel = "instancetype.kubevirt.io/default-instancetype-kind" // Same as outer context
		})

		It("should return defaults from PVC when spec.Source.PVC is set and PVC has labels", func() {
			dataVolumeSpec := &cdiv1.DataVolumeSpec{
				Source: &cdiv1.DataVolumeSource{
					PVC: &cdiv1.DataVolumeSourcePVC{Name: pvcName, Namespace: vmNamespace},
				},
			}
			pvc := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pvcName,
					Namespace: vmNamespace,
					Labels:    map[string]string{nameLabel: "pvc-name", kindLabel: "pvc-kind"},
				},
			}
			_, err := fakeClientset.CoreV1().PersistentVolumeClaims(vmNamespace).Create(context.Background(), pvc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			defaultName, defaultKind, err := testHandler.FromDataVolumeSpecTestWrangler(dataVolumeSpec, nameLabel, kindLabel, vmNamespace)
			Expect(err).ToNot(HaveOccurred())
			Expect(defaultName).To(Equal("pvc-name"))
			Expect(defaultKind).To(Equal("pvc-kind"))
		})

		It("should return error from PVC when spec.Source.PVC is set but PVC is missing labels", func() {
			dataVolumeSpec := &cdiv1.DataVolumeSpec{
				Source: &cdiv1.DataVolumeSource{
					PVC: &cdiv1.DataVolumeSourcePVC{Name: pvcName, Namespace: vmNamespace},
				},
			}
			pvc := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pvcName,
					Namespace: vmNamespace,
					Labels:    map[string]string{kindLabel: "pvc-kind-only"}, // Missing nameLabel
				},
			}
			_, err := fakeClientset.CoreV1().PersistentVolumeClaims(vmNamespace).Create(context.Background(), pvc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			defaultName, defaultKind, err := testHandler.FromDataVolumeSpecTestWrangler(dataVolumeSpec, nameLabel, kindLabel, vmNamespace)
			Expect(err).To(HaveOccurred())
			var ignoreableErr *infer.IgnoreableInferenceError
			Expect(errors.As(err, &ignoreableErr)).To(BeTrue())
			Expect(err.Error()).To(Equal(fmt.Sprintf(infer.MissingLabelFmtTestWrangler, nameLabel)))
			Expect(defaultName).To(BeEmpty())
			Expect(defaultKind).To(BeEmpty())
		})

		It("should return defaults from DataSource when spec.SourceRef is set and DataSource has labels", func() {
			dataVolumeSpec := &cdiv1.DataVolumeSpec{
				SourceRef: &cdiv1.DataVolumeSourceRef{
					Kind: "DataSource",
					Name: dataSourceName,
					Namespace: &vmNamespace,
				},
			}
			dataSource := &cdiv1.DataSource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dataSourceName,
					Namespace: vmNamespace,
					Labels:    map[string]string{nameLabel: "ds-name", kindLabel: "ds-kind"},
				},
				Spec: cdiv1.DataSourceSpec{Source: cdiv1.DataSourceSource{}}, // Ensure Spec.Source is not nil
			}
			_, err := cdiFakeClientset.CdiV1beta1().DataSources(vmNamespace).Create(context.Background(), dataSource, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			defaultName, defaultKind, err := testHandler.FromDataVolumeSpecTestWrangler(dataVolumeSpec, nameLabel, kindLabel, vmNamespace)
			Expect(err).ToNot(HaveOccurred())
			Expect(defaultName).To(Equal("ds-name"))
			Expect(defaultKind).To(Equal("ds-kind"))
		})

		It("should return error from DataSource when spec.SourceRef is set but DataSource is missing labels (and no underlying PVC in DS)", func() {
			dataVolumeSpec := &cdiv1.DataVolumeSpec{
				SourceRef: &cdiv1.DataVolumeSourceRef{
					Kind: "DataSource",
					Name: dataSourceName,
					Namespace: &vmNamespace,
				},
			}
			dataSource := &cdiv1.DataSource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dataSourceName,
					Namespace: vmNamespace,
					// No labels
				},
				Spec: cdiv1.DataSourceSpec{
					Source: cdiv1.DataSourceSource{
						// No PVC defined in DataSource's spec
					},
				},
			}
			_, err := cdiFakeClientset.CdiV1beta1().DataSources(vmNamespace).Create(context.Background(), dataSource, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			defaultName, defaultKind, err := testHandler.FromDataVolumeSpecTestWrangler(dataVolumeSpec, nameLabel, kindLabel, vmNamespace)
			Expect(err).To(HaveOccurred())
			var ignoreableErr *infer.IgnoreableInferenceError
			Expect(errors.As(err, &ignoreableErr)).To(BeTrue(), "Error should be IgnoreableInferenceError")
			// This error comes from fromLabels within fromDataSource
			Expect(err.Error()).To(Equal(fmt.Sprintf(infer.MissingLabelFmtTestWrangler, nameLabel)))
			Expect(defaultName).To(BeEmpty())
			Expect(defaultKind).To(BeEmpty())
		})

		It("should return IgnoreableInferenceError when neither spec.Source.PVC nor spec.SourceRef is set", func() {
			dataVolumeSpec := &cdiv1.DataVolumeSpec{} // Empty spec

			defaultName, defaultKind, err := testHandler.FromDataVolumeSpecTestWrangler(dataVolumeSpec, nameLabel, kindLabel, vmNamespace)

			Expect(err).To(HaveOccurred())
			var ignoreableErr *infer.IgnoreableInferenceError
			Expect(errors.As(err, &ignoreableErr)).To(BeTrue())
			Expect(err.Error()).To(Equal(infer.UnsupportedDataVolumeSourceTestWrangler))
			Expect(defaultName).To(BeEmpty())
			Expect(defaultKind).To(BeEmpty())
		})
	})

	Context("fromDataSource function", func() {
		var (
			dataSourceName       string
			pvcName              string
			nameLabel            string
			kindLabel            string
			currentTestNamespace string // Can be vmNamespace from outer context
		)

		BeforeEach(func() {
			dataSourceName = "test-ds"
			pvcName = "test-ds-pvc"
			nameLabel = "instancetype.kubevirt.io/default-instancetype"
			kindLabel = "instancetype.kubevirt.io/default-instancetype-kind"
			currentTestNamespace = vmNamespace // vmNamespace is from the parent BeforeEach
		})

		It("should return defaults when DataSource is found and has labels", func() {
			dataSource := &cdiv1.DataSource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dataSourceName,
					Namespace: currentTestNamespace,
					Labels:    map[string]string{nameLabel: "ds-direct-name", kindLabel: "ds-direct-kind"},
				},
				Spec: cdiv1.DataSourceSpec{Source: cdiv1.DataSourceSource{}}, // Ensure Spec.Source is not nil
			}
			_, err := cdiFakeClientset.CdiV1beta1().DataSources(currentTestNamespace).Create(context.Background(), dataSource, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			defaultName, defaultKind, err := testHandler.FromDataSourceTestWrangler(dataSourceName, currentTestNamespace, nameLabel, kindLabel)
			Expect(err).ToNot(HaveOccurred())
			Expect(defaultName).To(Equal("ds-direct-name"))
			Expect(defaultKind).To(Equal("ds-direct-kind"))
		})

		It("should return defaults from underlying PVC when DataSource is found without labels but Spec.Source.PVC is set", func() {
			dataSource := &cdiv1.DataSource{
				ObjectMeta: metav1.ObjectMeta{Name: dataSourceName, Namespace: currentTestNamespace},
				Spec: cdiv1.DataSourceSpec{
					Source: cdiv1.DataSourceSource{
						PVC: &cdiv1.DataVolumeSourcePVC{Name: pvcName, Namespace: currentTestNamespace},
					},
				},
			}
			pvc := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pvcName,
					Namespace: currentTestNamespace,
					Labels:    map[string]string{nameLabel: "pvc-via-ds-name", kindLabel: "pvc-via-ds-kind"},
				},
			}
			_, err := cdiFakeClientset.CdiV1beta1().DataSources(currentTestNamespace).Create(context.Background(), dataSource, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			_, err = fakeClientset.CoreV1().PersistentVolumeClaims(currentTestNamespace).Create(context.Background(), pvc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			defaultName, defaultKind, err := testHandler.FromDataSourceTestWrangler(dataSourceName, currentTestNamespace, nameLabel, kindLabel)
			Expect(err).ToNot(HaveOccurred())
			Expect(defaultName).To(Equal("pvc-via-ds-name"))
			Expect(defaultKind).To(Equal("pvc-via-ds-kind"))
		})

		It("should return IgnoreableInferenceError if DS has no labels and underlying PVC is missing nameLabel", func() {
			dataSource := &cdiv1.DataSource{
				ObjectMeta: metav1.ObjectMeta{Name: dataSourceName, Namespace: currentTestNamespace},
				Spec: cdiv1.DataSourceSpec{
					Source: cdiv1.DataSourceSource{
						PVC: &cdiv1.DataVolumeSourcePVC{Name: pvcName, Namespace: currentTestNamespace},
					},
				},
			}
			pvc := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pvcName,
					Namespace: currentTestNamespace,
					Labels:    map[string]string{kindLabel: "pvc-kind-only"}, // Missing nameLabel
				},
			}
			_, err := cdiFakeClientset.CdiV1beta1().DataSources(currentTestNamespace).Create(context.Background(), dataSource, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			_, err = fakeClientset.CoreV1().PersistentVolumeClaims(currentTestNamespace).Create(context.Background(), pvc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			defaultName, defaultKind, err := testHandler.FromDataSourceTestWrangler(dataSourceName, currentTestNamespace, nameLabel, kindLabel)
			Expect(err).To(HaveOccurred())
			var ignoreableErr *infer.IgnoreableInferenceError
			Expect(errors.As(err, &ignoreableErr)).To(BeTrue())
			Expect(err.Error()).To(Equal(fmt.Sprintf(infer.MissingLabelFmtTestWrangler, nameLabel)))
			Expect(defaultName).To(BeEmpty())
			Expect(defaultKind).To(BeEmpty())
		})

		It("should return error when DataSource is not found", func() {
			cdiFakeClientset.PrependReactor("get", "datasources", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, nil, k8serrors.NewNotFound(cdiv1.Resource("datasources"), dataSourceName)
			})

			defaultName, defaultKind, err := testHandler.FromDataSourceTestWrangler(dataSourceName, currentTestNamespace, nameLabel, kindLabel)
			Expect(err).To(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
			Expect(defaultName).To(BeEmpty())
			Expect(defaultKind).To(BeEmpty())
		})

		It("should return IgnoreableInferenceError when DataSource is found without labels and Spec.Source.PVC is nil", func() {
			dataSource := &cdiv1.DataSource{
				ObjectMeta: metav1.ObjectMeta{Name: dataSourceName, Namespace: currentTestNamespace},
				Spec: cdiv1.DataSourceSpec{
					Source: cdiv1.DataSourceSource{ // PVC is nil by default here
						PVC: nil,
					},
				},
			}
			_, err := cdiFakeClientset.CdiV1beta1().DataSources(currentTestNamespace).Create(context.Background(), dataSource, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			defaultName, defaultKind, err := testHandler.FromDataSourceTestWrangler(dataSourceName, currentTestNamespace, nameLabel, kindLabel)
			Expect(err).To(HaveOccurred())
			var ignoreableErr *infer.IgnoreableInferenceError
			Expect(errors.As(err, &ignoreableErr)).To(BeTrue())
			Expect(err.Error()).To(Equal(infer.MissingDataVolumeSourcePVCTestWrangler))
			Expect(defaultName).To(BeEmpty())
			Expect(defaultKind).To(BeEmpty())
		})
	})

	Context("fromDataVolumeSourceRef function", func() {
		var (
			dataSourceName string
			nameLabel      string
			kindLabel      string
			// vmNamespace is from the parent BeforeEach
			otherNamespace string
		)

		BeforeEach(func() {
			dataSourceName = "test-ds-ref"
			nameLabel = "instancetype.kubevirt.io/default-instancetype"
			kindLabel = "instancetype.kubevirt.io/default-instancetype-kind"
			otherNamespace = "other-namespace"

			// Ensure cdiFakeClientset is clean for Get reactions or create new ones for each test if needed
			// For this context, we will rely on specific Get reactors per test.
		})

		It("should return defaults when SourceRef Kind is DataSource, namespace is provided, and DataSource has labels", func() {
			sourceRef := &cdiv1.DataVolumeSourceRef{
				Kind:      "DataSource",
				Name:      dataSourceName,
				Namespace: &otherNamespace,
			}
			dataSource := &cdiv1.DataSource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dataSourceName,
					Namespace: otherNamespace,
					Labels:    map[string]string{nameLabel: "ds-ref-name", kindLabel: "ds-ref-kind"},
				},
				Spec: cdiv1.DataSourceSpec{Source: cdiv1.DataSourceSource{}},
			}
			cdiFakeClientset.PrependReactor("get", "datasources", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				getAction := action.(k8stesting.GetAction)
				if getAction.GetName() == dataSourceName && getAction.GetNamespace() == otherNamespace {
					return true, dataSource.DeepCopy(), nil
				}
				return false, nil, nil // Fallthrough for other Get calls
			})

			defaultName, defaultKind, err := testHandler.FromDataVolumeSourceRefTestWrangler(sourceRef, nameLabel, kindLabel, vmNamespace)
			Expect(err).ToNot(HaveOccurred())
			Expect(defaultName).To(Equal("ds-ref-name"))
			Expect(defaultKind).To(Equal("ds-ref-kind"))
		})

		It("should return defaults when SourceRef Kind is DataSource, namespace is NOT provided, and DataSource in vmNamespace has labels", func() {
			sourceRef := &cdiv1.DataVolumeSourceRef{
				Kind: "DataSource",
				Name: dataSourceName,
				// Namespace is nil
			}
			dataSource := &cdiv1.DataSource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dataSourceName,
					Namespace: vmNamespace, // Expect to be fetched from vmNamespace
					Labels:    map[string]string{nameLabel: "ds-vmspace-name", kindLabel: "ds-vmspace-kind"},
				},
				Spec: cdiv1.DataSourceSpec{Source: cdiv1.DataSourceSource{}},
			}
			cdiFakeClientset.PrependReactor("get", "datasources", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				getAction := action.(k8stesting.GetAction)
				if getAction.GetName() == dataSourceName && getAction.GetNamespace() == vmNamespace {
					return true, dataSource.DeepCopy(), nil
				}
				return false, nil, nil
			})

			defaultName, defaultKind, err := testHandler.FromDataVolumeSourceRefTestWrangler(sourceRef, nameLabel, kindLabel, vmNamespace)
			Expect(err).ToNot(HaveOccurred())
			Expect(defaultName).To(Equal("ds-vmspace-name"))
			Expect(defaultKind).To(Equal("ds-vmspace-kind"))
		})

		It("should return error when SourceRef Kind is DataSource but referenced DataSource is not found", func() {
			sourceRef := &cdiv1.DataVolumeSourceRef{
				Kind:      "DataSource",
				Name:      dataSourceName,
				Namespace: &vmNamespace,
			}
			cdiFakeClientset.PrependReactor("get", "datasources", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				getAction := action.(k8stesting.GetAction)
				if getAction.GetName() == dataSourceName && getAction.GetNamespace() == vmNamespace {
					return true, nil, k8serrors.NewNotFound(cdiv1.Resource("datasources"), dataSourceName)
				}
				return false, nil, nil
			})

			defaultName, defaultKind, err := testHandler.FromDataVolumeSourceRefTestWrangler(sourceRef, nameLabel, kindLabel, vmNamespace)
			Expect(err).To(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
			Expect(defaultName).To(BeEmpty())
			Expect(defaultKind).To(BeEmpty())
		})

		It("should return IgnoreableInferenceError when SourceRef Kind is not DataSource", func() {
			invalidKind := "InvalidKind"
			sourceRef := &cdiv1.DataVolumeSourceRef{
				Kind: invalidKind,
				Name: dataSourceName,
			}

			defaultName, defaultKind, err := testHandler.FromDataVolumeSourceRefTestWrangler(sourceRef, nameLabel, kindLabel, vmNamespace)
			Expect(err).To(HaveOccurred())
			var ignoreableErr *infer.IgnoreableInferenceError
			Expect(errors.As(err, &ignoreableErr)).To(BeTrue())
			Expect(err.Error()).To(Equal(fmt.Sprintf(infer.UnsupportedDataVolumeSourceRefFmtTestWrangler, invalidKind)))
			Expect(defaultName).To(BeEmpty())
			Expect(defaultKind).To(BeEmpty())
		})
	})

	Context("fromVolumes function", func() {
		var (
			baseVM             *virtv1.VirtualMachine
			inferVolumeName    string
			pvcVolumeName      string
			dvVolumeName       string
			secretVolumeName   string
			pvcActualName      string
			dvActualName       string
			nonExistentPvcName string
			nameLabel          string
			kindLabel          string
			// vmNamespace is from the parent BeforeEach
		)

		BeforeEach(func() {
			// vmNamespace is inherited from the global BeforeEach
			pvcVolumeName = "pvc-volume"
			dvVolumeName = "dv-volume"
			secretVolumeName = "secret-volume"
			inferVolumeName = "non-existent-volume" // Used for testing "not found" case
			pvcActualName = "actual-pvc"
			dvActualName = "actual-dv"
			nonExistentPvcName = "i-do-not-exist-pvc"

			nameLabel = "instancetype.kubevirt.io/default-instancetype"
			kindLabel = "instancetype.kubevirt.io/default-instancetype-kind"

			baseVM = &virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{Namespace: vmNamespace},
				Spec: virtv1.VirtualMachineSpec{
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{ // Ensure Domain is not nil if needed, and Volumes slice
							Volumes: []virtv1.Volume{},
						},
					},
				},
			}
		})

		It("should return defaults from PVC when inferFromVolumeName matches a PVC-backed volume", func() {
			vm := baseVM.DeepCopy()
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: pvcVolumeName,
				VolumeSource: virtv1.VolumeSource{
					PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcActualName,
					}},
				},
			})
			pvc := &k8sv1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pvcActualName,
					Namespace: vmNamespace,
					Labels:    map[string]string{nameLabel: "pvc-vol-name", kindLabel: "pvc-vol-kind"},
				},
			}
			_, err := fakeClientset.CoreV1().PersistentVolumeClaims(vmNamespace).Create(context.Background(), pvc, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			defaultName, defaultKind, err := testHandler.FromVolumesTestWrangler(vm, pvcVolumeName, nameLabel, kindLabel)
			Expect(err).ToNot(HaveOccurred())
			Expect(defaultName).To(Equal("pvc-vol-name"))
			Expect(defaultKind).To(Equal("pvc-vol-kind"))
		})

		It("should return defaults from DataVolume when inferFromVolumeName matches a DataVolume-backed volume", func() {
			vm := baseVM.DeepCopy()
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: dvVolumeName,
				VolumeSource: virtv1.VolumeSource{
					DataVolume: &virtv1.DataVolumeSource{
						Name: dvActualName,
					},
				},
			})
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dvActualName,
					Namespace: vmNamespace,
					Labels:    map[string]string{nameLabel: "dv-vol-name", kindLabel: "dv-vol-kind"},
				},
			}
			_, err := cdiFakeClientset.CdiV1beta1().DataVolumes(vmNamespace).Create(context.Background(), dv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			defaultName, defaultKind, err := testHandler.FromVolumesTestWrangler(vm, dvVolumeName, nameLabel, kindLabel)
			Expect(err).ToNot(HaveOccurred())
			Expect(defaultName).To(Equal("dv-vol-name"))
			Expect(defaultKind).To(Equal("dv-vol-kind"))
		})

		It("should return IgnoreableInferenceError for unsupported volume type (e.g., Secret)", func() {
			vm := baseVM.DeepCopy()
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: secretVolumeName,
				VolumeSource: virtv1.VolumeSource{
					Secret: &virtv1.SecretVolumeSource{SecretName: "some-secret"},
				},
			})

			defaultName, defaultKind, err := testHandler.FromVolumesTestWrangler(vm, secretVolumeName, nameLabel, kindLabel)
			Expect(err).To(HaveOccurred())
			var ignoreableErr *infer.IgnoreableInferenceError
			Expect(errors.As(err, &ignoreableErr)).To(BeTrue())
			Expect(err.Error()).To(Equal(fmt.Sprintf(infer.UnsupportedVolumeTypeFmtTestWrangler, secretVolumeName)))
			Expect(defaultName).To(BeEmpty())
			Expect(defaultKind).To(BeEmpty())
		})

		It("should return error when no volume matches inferFromVolumeName", func() {
			vm := baseVM.DeepCopy() // Has no volumes matching inferVolumeName by default

			defaultName, defaultKind, err := testHandler.FromVolumesTestWrangler(vm, inferVolumeName, nameLabel, kindLabel)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("unable to find volume %s to infer defaults", inferVolumeName)))
			Expect(defaultName).To(BeEmpty())
			Expect(defaultKind).To(BeEmpty())
		})

		It("should propagate error from fromPVC if PVC is not found", func() {
			vm := baseVM.DeepCopy()
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, virtv1.Volume{
				Name: pvcVolumeName,
				VolumeSource: virtv1.VolumeSource{
					PersistentVolumeClaim: &virtv1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: nonExistentPvcName,
					}},
				},
			})
			// Do NOT create the PVC nonExistentPvcName
			fakeClientset.PrependReactor("get", "persistentvolumeclaims", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				getAction := action.(k8stesting.GetAction)
				if getAction.GetName() == nonExistentPvcName && getAction.GetNamespace() == vmNamespace {
					return true, nil, k8serrors.NewNotFound(k8sv1.Resource("persistentvolumeclaims"), nonExistentPvcName)
				}
				return false, nil, nil
			})


			defaultName, defaultKind, err := testHandler.FromVolumesTestWrangler(vm, pvcVolumeName, nameLabel, kindLabel)
			Expect(err).To(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
			Expect(defaultName).To(BeEmpty())
			Expect(defaultKind).To(BeEmpty())
		})
	})
})
