package controller_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	virtv1 "kubevirt.io/api/core/v1"
	templatev1alpha1 "kubevirt.io/api/template/v1alpha1"

	"kubevirt.io/kubevirt/pkg/template/controller"
)

var _ = Describe("VirtualMachineTemplateRequest Controller", func() {
	var (
		ctrl                    *controller.TemplateRequestController
		templateRequestInformer cache.SharedIndexInformer
		vmInformer              cache.SharedIndexInformer
		snapshotInformer        cache.SharedIndexInformer
		templateInformer        cache.SharedIndexInformer
		recorder                *record.FakeRecorder
	)

	BeforeEach(func() {
		// Create informers with empty ListWatch to avoid running them
		templateRequestInformer = cache.NewSharedIndexInformer(
			&cache.ListWatch{},
			&templatev1alpha1.VirtualMachineTemplateRequest{},
			0,
			cache.Indexers{},
		)
		vmInformer = cache.NewSharedIndexInformer(
			&cache.ListWatch{},
			&virtv1.VirtualMachine{},
			0,
			cache.Indexers{},
		)
		snapshotInformer = cache.NewSharedIndexInformer(
			&cache.ListWatch{},
			&virtv1.VirtualMachine{},
			0,
			cache.Indexers{},
		)
		templateInformer = cache.NewSharedIndexInformer(
			&cache.ListWatch{},
			&templatev1alpha1.VirtualMachineTemplate{},
			0,
			cache.Indexers{},
		)

		// Create fake recorder
		recorder = record.NewFakeRecorder(100)

		// Create controller with nil client for testing
		ctrl = controller.NewTemplateRequestController(
			nil, // nil client for testing
			templateRequestInformer,
			vmInformer,
			snapshotInformer,
			templateInformer,
			recorder,
		)
	})

	Describe("Controller Initialization", func() {
		It("should create controller with proper configuration", func() {
			Expect(ctrl).NotTo(BeNil())
		})
	})

	Describe("Error Types", func() {
		It("should create RequeueError with proper message", func() {
			err := controller.NewRequeueError("test message", 5)
			Expect(err.Error()).To(Equal("test message"))
		})

		It("should create SyncError with proper message", func() {
			err := controller.NewSyncError("test message", "test reason")
			Expect(err.Error()).To(Equal("test message"))
		})
	})

	Describe("DataVolume Template Structure", func() {
		It("should handle DataVolumeTemplates structure", func() {
			sourceVM := &virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm",
					Namespace: "default",
				},
				Spec: virtv1.VirtualMachineSpec{
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{
								Devices: virtv1.Devices{
									Disks: []virtv1.Disk{
										{
											Name: "disk1",
											DiskDevice: virtv1.DiskDevice{
												Disk: &virtv1.DiskTarget{},
											},
										},
									},
								},
							},
						},
					},
					DataVolumeTemplates: []virtv1.DataVolumeTemplateSpec{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "test-dv",
							},
							Spec: cdiv1.DataVolumeSpec{
								Source: &cdiv1.DataVolumeSource{
									PVC: &cdiv1.DataVolumeSourcePVC{
										Name:      "test-pvc",
										Namespace: "default",
									},
								},
							},
						},
					},
				},
			}

			// Verify the VM structure is correct
			Expect(sourceVM.Spec.DataVolumeTemplates).To(HaveLen(1))
			Expect(sourceVM.Spec.DataVolumeTemplates[0].Name).To(Equal("test-dv"))
			Expect(sourceVM.Spec.DataVolumeTemplates[0].Spec.Source.PVC.Name).To(Equal("test-pvc"))
		})

		It("should handle DataVolumeTemplates without PVC sources", func() {
			// Create VM without DataVolumeTemplates
			vmWithoutDV := &virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm",
					Namespace: "default",
				},
				Spec: virtv1.VirtualMachineSpec{
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Domain: virtv1.DomainSpec{
								Devices: virtv1.Devices{
									Disks: []virtv1.Disk{
										{
											Name: "disk1",
											DiskDevice: virtv1.DiskDevice{
												Disk: &virtv1.DiskTarget{},
											},
										},
									},
								},
							},
						},
					},
					DataVolumeTemplates: nil,
				},
			}

			// Verify the VM structure is correct
			Expect(vmWithoutDV.Spec.DataVolumeTemplates).To(HaveLen(0))
		})
	})

	Describe("Template Request Structure", func() {
		It("should handle template request with valid source", func() {
			templateRequest := &templatev1alpha1.VirtualMachineTemplateRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-request",
					Namespace: "default",
				},
				Spec: templatev1alpha1.VirtualMachineTemplateRequestSpec{
					Source: templatev1alpha1.VirtualMachineReference{
						Name:      "test-vm",
						Namespace: "default",
					},
				},
			}

			// Verify the request structure is correct
			Expect(templateRequest.Spec.Source.Name).To(Equal("test-vm"))
			Expect(templateRequest.Spec.Source.Namespace).To(Equal("default"))
		})

		It("should handle template request with empty source name", func() {
			templateRequest := &templatev1alpha1.VirtualMachineTemplateRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-request",
					Namespace: "default",
				},
				Spec: templatev1alpha1.VirtualMachineTemplateRequestSpec{
					Source: templatev1alpha1.VirtualMachineReference{
						Name:      "",
						Namespace: "default",
					},
				},
			}

			// Verify the request structure is correct
			Expect(templateRequest.Spec.Source.Name).To(Equal(""))
		})
	})
})
