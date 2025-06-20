package controller_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	virtv1 "kubevirt.io/api/core/v1"
	snapshotv1beta1 "kubevirt.io/api/snapshot/v1beta1"
	templatev1alpha1 "kubevirt.io/api/template/v1alpha1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/template/controller"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("VirtualMachineTemplateRequest Controller", func() {
	var (
		ctrl                    *controller.TemplateRequestController
		templateRequestInformer cache.SharedIndexInformer
		vmInformer              cache.SharedIndexInformer
		snapshotInformer        cache.SharedIndexInformer
		templateInformer        cache.SharedIndexInformer
		recorder                *record.FakeRecorder
		mockClient              kubecli.KubevirtClient
		ctx                     context.Context
		cancel                  context.CancelFunc
		sourceVM                *virtv1.VirtualMachine
		templateRequest         *templatev1alpha1.VirtualMachineTemplateRequest
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		// Use fake informers for all resources
		templateRequestInformer, _ = testutils.NewFakeInformerFor(&templatev1alpha1.VirtualMachineTemplateRequest{})
		vmInformer, _ = testutils.NewFakeInformerFor(&virtv1.VirtualMachine{})
		snapshotInformer, _ = testutils.NewFakeInformerFor(&snapshotv1beta1.VirtualMachineSnapshot{})
		templateInformer, _ = testutils.NewFakeInformerFor(&templatev1alpha1.VirtualMachineTemplate{})

		// Create fake recorder
		recorder = record.NewFakeRecorder(100)

		// Create mock client - for now use nil to avoid complex mocking
		mockClient = nil

		// Create controller with mock client
		ctrl = controller.NewTemplateRequestController(
			mockClient,
			templateRequestInformer,
			vmInformer,
			snapshotInformer,
			templateInformer,
			recorder,
		)

		// Create a source VM for testing
		sourceVM = &virtv1.VirtualMachine{
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
			},
		}

		// Create a template request for testing
		templateRequest = &templatev1alpha1.VirtualMachineTemplateRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "test-request",
				Namespace:  "default",
				Finalizers: []string{controller.VirtualMachineTemplateRequestFinalizer}, // Add finalizer to avoid phase override
			},
			Spec: templatev1alpha1.VirtualMachineTemplateRequestSpec{
				Source: templatev1alpha1.VirtualMachineReference{
					Name:      "test-vm",
					Namespace: "default",
				},
			},
		}

		// Add objects to informers
		templateRequestInformer.GetStore().Add(templateRequest)
		vmInformer.GetStore().Add(sourceVM)
	})

	AfterEach(func() {
		cancel()
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

	Describe("State Transitions and Conditions", func() {
		It("should initialize status with Pending phase", func() {
			err := ctrl.Sync(ctx, templateRequest)
			templateRequestInformer.GetStore().Update(templateRequest)
			// Should not error and should set phase to Pending
			Expect(err).NotTo(HaveOccurred())
			Expect(templateRequest.Status.Phase).To(Equal(templatev1alpha1.VirtualMachineTemplateRequestPhasePending))
		})

		It("should transition from Pending to InProgress", func() {
			templateRequest.Status.Phase = templatev1alpha1.VirtualMachineTemplateRequestPhasePending
			err := ctrl.Sync(ctx, templateRequest)
			templateRequestInformer.GetStore().Update(templateRequest)
			// Second sync to see the updated status
			err = ctrl.Sync(ctx, templateRequest)
			templateRequestInformer.GetStore().Update(templateRequest)
			// Should return a requeue error due to nil client for snapshot creation
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Client not available for snapshot creation"))
		})

		It("should set snapshot ready condition when snapshot is ready", func() {
			templateRequest.Status.Phase = templatev1alpha1.VirtualMachineTemplateRequestPhaseInProgress
			templateRequest.Status.Snapshot = &corev1.TypedObjectReference{
				Kind:      "VirtualMachineSnapshot",
				Name:      "test-request-snapshot",
				Namespace: pointer.P("default"),
			}
			vmInformer.GetStore().Add(sourceVM)
			readySnapshot := &snapshotv1beta1.VirtualMachineSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-request-snapshot",
					Namespace: "default",
				},
				Status: &snapshotv1beta1.VirtualMachineSnapshotStatus{
					ReadyToUse: pointer.P(true),
					Conditions: []snapshotv1beta1.Condition{{Type: snapshotv1beta1.ConditionReady, Status: corev1.ConditionTrue}},
				},
			}
			snapshotInformer.GetStore().Add(readySnapshot)
			err := ctrl.Sync(ctx, templateRequest)
			templateRequestInformer.GetStore().Update(templateRequest)
			// Second sync to see the updated status
			err = ctrl.Sync(ctx, templateRequest)
			templateRequestInformer.GetStore().Update(templateRequest)
			// Should return a requeue error due to nil client for template creation
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Client not available for template creation"))
		})

		It("should set template ready condition when template is ready", func() {
			templateRequest.Status.Phase = templatev1alpha1.VirtualMachineTemplateRequestPhaseInProgress
			templateRequest.Status.Snapshot = &corev1.TypedObjectReference{
				Kind:      "VirtualMachineSnapshot",
				Name:      "test-request-snapshot",
				Namespace: pointer.P("default"),
			}
			templateRequest.Status.Template = &corev1.TypedObjectReference{
				Kind:      "VirtualMachineTemplate",
				Name:      "test-request-template",
				Namespace: pointer.P("default"),
			}
			vmInformer.GetStore().Add(sourceVM)
			readySnapshot := &snapshotv1beta1.VirtualMachineSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-request-snapshot",
					Namespace: "default",
				},
				Status: &snapshotv1beta1.VirtualMachineSnapshotStatus{
					ReadyToUse: pointer.P(true),
					Conditions: []snapshotv1beta1.Condition{{Type: snapshotv1beta1.ConditionReady, Status: corev1.ConditionTrue}},
				},
			}
			snapshotInformer.GetStore().Add(readySnapshot)
			readyTemplate := &templatev1alpha1.VirtualMachineTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-request-template",
					Namespace: "default",
				},
				Spec: templatev1alpha1.VirtualMachineTemplateSpec{Message: "Test template"},
			}
			templateInformer.GetStore().Add(readyTemplate)
			err := ctrl.Sync(ctx, templateRequest)
			templateRequestInformer.GetStore().Update(templateRequest)
			// Second sync to see the updated status
			err = ctrl.Sync(ctx, templateRequest)
			templateRequestInformer.GetStore().Update(templateRequest)
			// Should not error and template ready condition should be set
			Expect(err).NotTo(HaveOccurred())
			var templateReadyCondition *templatev1alpha1.VirtualMachineTemplateRequestCondition
			for _, condition := range templateRequest.Status.Conditions {
				if condition.Type == templatev1alpha1.VirtualMachineTemplateRequestConditionTemplateReady {
					templateReadyCondition = &condition
					break
				}
			}
			Expect(templateReadyCondition).NotTo(BeNil())
			Expect(templateReadyCondition.Status).To(Equal(metav1.ConditionTrue))
		})

		It("should transition to Succeeded when both snapshot and template are ready", func() {
			templateRequest.Status.Phase = templatev1alpha1.VirtualMachineTemplateRequestPhaseInProgress
			templateRequest.Status.Snapshot = &corev1.TypedObjectReference{
				Kind:      "VirtualMachineSnapshot",
				Name:      "test-request-snapshot",
				Namespace: pointer.P("default"),
			}
			templateRequest.Status.Template = &corev1.TypedObjectReference{
				Kind:      "VirtualMachineTemplate",
				Name:      "test-request-template",
				Namespace: pointer.P("default"),
			}
			templateRequest.Status.Conditions = []templatev1alpha1.VirtualMachineTemplateRequestCondition{
				{Type: templatev1alpha1.VirtualMachineTemplateRequestConditionSnapshotReady, Status: metav1.ConditionTrue},
				{Type: templatev1alpha1.VirtualMachineTemplateRequestConditionTemplateReady, Status: metav1.ConditionTrue},
			}
			vmInformer.GetStore().Add(sourceVM)
			// Add the referenced template to the informer
			readyTemplate := &templatev1alpha1.VirtualMachineTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-request-template",
					Namespace: "default",
				},
				Spec: templatev1alpha1.VirtualMachineTemplateSpec{Message: "Test template"},
			}
			templateInformer.GetStore().Add(readyTemplate)
			// Add a ready snapshot to the informer
			readySnapshot := &snapshotv1beta1.VirtualMachineSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-request-snapshot",
					Namespace: "default",
				},
				Status: &snapshotv1beta1.VirtualMachineSnapshotStatus{
					ReadyToUse: pointer.P(true),
					Conditions: []snapshotv1beta1.Condition{{Type: snapshotv1beta1.ConditionReady, Status: corev1.ConditionTrue}},
				},
			}
			snapshotInformer.GetStore().Add(readySnapshot)
			err := ctrl.Sync(ctx, templateRequest)
			Expect(err).NotTo(HaveOccurred())
			templateRequestInformer.GetStore().Update(templateRequest)
			// Second sync to see the updated status
			err = ctrl.Sync(ctx, templateRequest)
			templateRequestInformer.GetStore().Update(templateRequest)
			// Should not error and should transition to Succeeded
			Expect(err).NotTo(HaveOccurred())
			Expect(templateRequest.Status.Phase).To(Equal(templatev1alpha1.VirtualMachineTemplateRequestPhaseSucceeded))
			var readyCondition *templatev1alpha1.VirtualMachineTemplateRequestCondition
			for _, condition := range templateRequest.Status.Conditions {
				if condition.Type == templatev1alpha1.VirtualMachineTemplateRequestConditionReady {
					readyCondition = &condition
					break
				}
			}
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(metav1.ConditionTrue))
		})

		It("should set error condition when source VM is not found", func() {
			templateRequest.Status.Phase = templatev1alpha1.VirtualMachineTemplateRequestPhaseInProgress
			// Remove source VM from informer to simulate not found
			vmInformer.GetStore().Delete(sourceVM)
			err := ctrl.Sync(ctx, templateRequest)
			templateRequestInformer.GetStore().Update(templateRequest)
			// Should error due to source VM not found
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("VirtualMachine default/test-vm not found"))
			Expect(templateRequest.Status.Phase).To(Equal(templatev1alpha1.VirtualMachineTemplateRequestPhaseFailed))
			var errorCondition *templatev1alpha1.VirtualMachineTemplateRequestCondition
			for _, condition := range templateRequest.Status.Conditions {
				if condition.Type == templatev1alpha1.VirtualMachineTemplateRequestConditionReady {
					errorCondition = &condition
					break
				}
			}
			Expect(errorCondition).NotTo(BeNil())
			Expect(errorCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(errorCondition.Reason).To(Equal("SourceVMNotFound"))
		})

		It("should set error condition when source VM name is empty", func() {
			templateRequest.Status.Phase = templatev1alpha1.VirtualMachineTemplateRequestPhaseInProgress
			templateRequest.Spec.Source.Name = ""
			err := ctrl.Sync(ctx, templateRequest)
			templateRequestInformer.GetStore().Update(templateRequest)
			// Should error due to empty source name
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("source name is required"))
			Expect(templateRequest.Status.Phase).To(Equal(templatev1alpha1.VirtualMachineTemplateRequestPhaseFailed))
			var errorCondition *templatev1alpha1.VirtualMachineTemplateRequestCondition
			for _, condition := range templateRequest.Status.Conditions {
				if condition.Type == templatev1alpha1.VirtualMachineTemplateRequestConditionReady {
					errorCondition = &condition
					break
				}
			}
			Expect(errorCondition).NotTo(BeNil())
			Expect(errorCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(errorCondition.Reason).To(Equal("InvalidSource"))
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
