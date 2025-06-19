package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	virtv1 "kubevirt.io/api/core/v1"
	snapshotv1beta1 "kubevirt.io/api/snapshot/v1beta1"
	templatev1alpha1 "kubevirt.io/api/template/v1alpha1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
)

// RequeueError is a special error type that indicates the controller should requeue
// the item after a delay
type RequeueError struct {
	message string
	delay   time.Duration
}

func (r *RequeueError) Error() string {
	return r.message
}

func NewRequeueError(message string, delay time.Duration) *RequeueError {
	return &RequeueError{
		message: message,
		delay:   delay,
	}
}

const (
	// Event reasons
	SuccessSynced        = "Synced"
	FailedSyncReason     = "FailedSync"
	FailedCreateSnapshot = "FailedCreateSnapshot"
	FailedCreateTemplate = "FailedCreateTemplate"
	FailedUpdateStatus   = "FailedUpdateStatus"
	SourceVMNotFound     = "SourceVMNotFound"
	SnapshotCreated      = "SnapshotCreated"
	TemplateCreated      = "TemplateCreated"

	// Event messages
	MessageResourceSynced = "VirtualMachineTemplateRequest synced successfully"

	// Log levels
	LogLevelInfo    = 2
	LogLevelVerbose = 4

	// Requeue delay when waiting for resources to become ready
	requeueDelay = 5 * time.Second
)

type TemplateRequestController struct {
	clientset               kubecli.KubevirtClient
	templateRequestInformer cache.SharedIndexInformer
	vmInformer              cache.SharedIndexInformer
	snapshotInformer        cache.SharedIndexInformer
	templateInformer        cache.SharedIndexInformer
	recorder                record.EventRecorder
	queue                   workqueue.TypedRateLimitingInterface[string]
}

func NewTemplateRequestController(
	clientset kubecli.KubevirtClient,
	templateRequestInformer cache.SharedIndexInformer,
	vmInformer cache.SharedIndexInformer,
	snapshotInformer cache.SharedIndexInformer,
	templateInformer cache.SharedIndexInformer,
	recorder record.EventRecorder,
) *TemplateRequestController {
	ctrl := &TemplateRequestController{
		clientset:               clientset,
		templateRequestInformer: templateRequestInformer,
		vmInformer:              vmInformer,
		snapshotInformer:        snapshotInformer,
		templateInformer:        templateInformer,
		recorder:                recorder,
		queue: workqueue.NewTypedRateLimitingQueue(
			workqueue.DefaultTypedControllerRateLimiter[string](),
		),
	}

	log.Log.Info("Setting up event handlers for VirtualMachineTemplateRequest")

	// Handle VirtualMachineTemplateRequest events
	if _, err := templateRequestInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ctrl.enqueueTemplateRequest(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			ctrl.enqueueTemplateRequest(new)
		},
		DeleteFunc: func(obj interface{}) {
			ctrl.enqueueTemplateRequest(obj)
		},
	}); err != nil {
		log.Log.Errorf("Failed to add VirtualMachineTemplateRequest event handler: %v", err)
	}

	// Handle VirtualMachineSnapshot events to re-enqueue owning VirtualMachineTemplateRequest
	if _, err := snapshotInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ctrl.enqueueTemplateRequestForSnapshot(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			ctrl.enqueueTemplateRequestForSnapshot(new)
		},
		DeleteFunc: func(obj interface{}) {
			ctrl.enqueueTemplateRequestForSnapshot(obj)
		},
	}); err != nil {
		log.Log.Errorf("Failed to add VirtualMachineSnapshot event handler: %v", err)
	}

	// Handle VirtualMachineTemplate events to re-enqueue owning VirtualMachineTemplateRequest
	if _, err := templateInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			ctrl.enqueueTemplateRequestForTemplate(obj)
		},
		UpdateFunc: func(old, new interface{}) {
			ctrl.enqueueTemplateRequestForTemplate(new)
		},
		DeleteFunc: func(obj interface{}) {
			ctrl.enqueueTemplateRequestForTemplate(obj)
		},
	}); err != nil {
		log.Log.Errorf("Failed to add VirtualMachineTemplate event handler: %v", err)
	}

	return ctrl
}

func (c *TemplateRequestController) Run(ctx context.Context, workers int) error {
	defer c.queue.ShutDown()

	log.Log.Info("Starting VirtualMachineTemplateRequest controller")

	// Wait for the caches to be synced before starting workers
	log.Log.Info("Waiting for informer caches to sync")
	if !cache.WaitForCacheSync(ctx.Done(),
		c.templateRequestInformer.HasSynced,
		c.vmInformer.HasSynced,
		c.snapshotInformer.HasSynced,
		c.templateInformer.HasSynced) {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	log.Log.Info("Starting workers")
	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.runWorker, time.Second)
	}

	log.Log.Info("Started workers")
	<-ctx.Done()
	log.Log.Info("Shutting down workers")

	return nil
}

func (c *TemplateRequestController) runWorker(ctx context.Context) {
	for c.processNextWorkItem(ctx) {
	}
}

func (c *TemplateRequestController) processNextWorkItem(ctx context.Context) bool {
	obj, shutdown := c.queue.Get()
	if shutdown {
		return false
	}

	err := func(obj string) error {
		defer c.queue.Done(obj)

		if err := c.syncHandler(ctx, obj); err != nil {
			// Check if it's a requeue error
			if requeueErr, ok := err.(*RequeueError); ok {
				log.Log.V(LogLevelVerbose).Infof("Requeuing '%s' after %v: %s", obj, requeueErr.delay, requeueErr.message)
				c.queue.AddAfter(obj, requeueErr.delay)
				return nil
			}
			// For other errors, use rate limiting
			c.queue.AddRateLimited(obj)
			return fmt.Errorf("error syncing '%s': %s, requeuing", obj, err.Error())
		}

		c.queue.Forget(obj)
		log.Log.V(LogLevelVerbose).Infof("Successfully synced '%s'", obj)
		return nil
	}(obj)
	if err != nil {
		log.Log.V(1).Info(err.Error())
		return true
	}

	return true
}

func (c *TemplateRequestController) syncHandler(ctx context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("invalid resource key: %s", key)
	}

	// Get the VirtualMachineTemplateRequest resource with this namespace/name
	obj, exists, err := c.templateRequestInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		log.Log.V(LogLevelInfo).Infof("VirtualMachineTemplateRequest '%s' no longer exists", key)
		return nil
	}

	templateRequest := obj.(*templatev1alpha1.VirtualMachineTemplateRequest)
	templateRequestCopy := templateRequest.DeepCopy()

	err = c.Sync(ctx, templateRequestCopy)
	if err != nil {
		c.recorder.Event(templateRequestCopy, corev1.EventTypeWarning, FailedSyncReason, err.Error())
		return err
	}

	c.recorder.Event(templateRequestCopy, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	log.Log.V(LogLevelVerbose).Infof("Successfully processed VirtualMachineTemplateRequest %s/%s", namespace, name)
	return nil
}

func (c *TemplateRequestController) Sync(
	ctx context.Context,
	templateRequest *templatev1alpha1.VirtualMachineTemplateRequest,
) error {
	// Initialize status if not set
	if templateRequest.Status.Phase == "" {
		templateRequest.Status.Phase = templatev1alpha1.VirtualMachineTemplateRequestPhasePending
		templateRequest.Status.ObservedGeneration = templateRequest.Generation
		return c.updateStatus(ctx, templateRequest)
	}

	// Set phase to InProgress if still pending
	if templateRequest.Status.Phase == templatev1alpha1.VirtualMachineTemplateRequestPhasePending {
		templateRequest.Status.Phase = templatev1alpha1.VirtualMachineTemplateRequestPhaseInProgress
		if err := c.updateStatus(ctx, templateRequest); err != nil {
			return err
		}
	}

	// Get the source VirtualMachine
	vm, err := c.getSourceVM(templateRequest)
	if err != nil {
		return c.handleError(ctx, templateRequest, SourceVMNotFound, err)
	}

	// Handle snapshot creation and tracking
	if err := c.handleSnapshot(ctx, templateRequest, vm); err != nil {
		return c.handleError(ctx, templateRequest, FailedCreateSnapshot, err)
	}

	// Handle template creation
	if err := c.handleTemplate(ctx, templateRequest, vm); err != nil {
		return c.handleError(ctx, templateRequest, FailedCreateTemplate, err)
	}

	// Mark as succeeded if both snapshot and template are ready
	if c.isSnapshotReady(templateRequest) && c.isTemplateReady(templateRequest) {
		templateRequest.Status.Phase = templatev1alpha1.VirtualMachineTemplateRequestPhaseSucceeded
		c.setCondition(
			templateRequest,
			templatev1alpha1.VirtualMachineTemplateRequestConditionReady,
			metav1.ConditionTrue,
			"Completed",
			"Template request completed successfully",
		)
	}

	// Update status first
	if err := c.updateStatus(ctx, templateRequest); err != nil {
		return err
	}

	// If we're still in progress and resources are not ready, requeue after delay
	if templateRequest.Status.Phase == templatev1alpha1.VirtualMachineTemplateRequestPhaseInProgress {
		if !c.isSnapshotReady(templateRequest) {
			return NewRequeueError("Waiting for VirtualMachineSnapshot to become ready", requeueDelay)
		}
		if !c.isTemplateReady(templateRequest) {
			return NewRequeueError("Waiting for VirtualMachineTemplate to become ready", requeueDelay)
		}
	}

	return nil
}

func (c *TemplateRequestController) getSourceVM(
	templateRequest *templatev1alpha1.VirtualMachineTemplateRequest,
) (*virtv1.VirtualMachine, error) {
	namespace := templateRequest.Spec.Source.Namespace
	if namespace == "" {
		namespace = templateRequest.Namespace
	}

	key := fmt.Sprintf("%s/%s", namespace, templateRequest.Spec.Source.Name)
	obj, exists, err := c.vmInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("VirtualMachine %s not found", key)
	}

	vm := obj.(*virtv1.VirtualMachine)
	return vm, nil
}

func (c *TemplateRequestController) handleSnapshot(
	ctx context.Context,
	templateRequest *templatev1alpha1.VirtualMachineTemplateRequest,
	vm *virtv1.VirtualMachine,
) error {
	// Check if snapshot already exists
	if templateRequest.Status.Snapshot != nil {
		return c.checkSnapshotStatus(templateRequest)
	}

	// Create snapshot
	snapshotName := fmt.Sprintf("%s-snapshot", templateRequest.Name)
	snapshot := &snapshotv1beta1.VirtualMachineSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      snapshotName,
			Namespace: templateRequest.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(
					templateRequest,
					templatev1alpha1.SchemeGroupVersion.WithKind("VirtualMachineTemplateRequest"),
				),
			},
		},
		Spec: snapshotv1beta1.VirtualMachineSnapshotSpec{
			Source: corev1.TypedLocalObjectReference{
				APIGroup: pointer.P("kubevirt.io"),
				Kind:     "VirtualMachine",
				Name:     vm.Name,
			},
		},
	}

	createdSnapshot, err := c.clientset.VirtualMachineSnapshot(templateRequest.Namespace).Create(
		ctx, snapshot, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create VirtualMachineSnapshot: %v", err)
	}

	// Handle the case where snapshot already exists
	if errors.IsAlreadyExists(err) {
		// Try to get the existing snapshot
		existingSnapshot, getErr := c.clientset.VirtualMachineSnapshot(templateRequest.Namespace).Get(ctx, snapshotName, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("failed to get existing VirtualMachineSnapshot: %v", getErr)
		}
		createdSnapshot = existingSnapshot
	}

	// Validate created snapshot has required fields
	if createdSnapshot.Name == "" {
		return fmt.Errorf("created VirtualMachineSnapshot has empty name")
	}
	if createdSnapshot.Namespace == "" {
		return fmt.Errorf("created VirtualMachineSnapshot has empty namespace")
	}

	// Update status with snapshot reference
	templateRequest.Status.Snapshot = &corev1.TypedObjectReference{
		Kind:      "VirtualMachineSnapshot",
		Name:      createdSnapshot.Name,
		Namespace: &createdSnapshot.Namespace,
	}

	c.setCondition(
		templateRequest,
		templatev1alpha1.VirtualMachineTemplateRequestConditionSnapshotReady,
		metav1.ConditionFalse,
		"Creating",
		"Snapshot is being created",
	)

	// Immediately update status to persist the snapshot reference
	if err := c.updateStatus(ctx, templateRequest); err != nil {
		return fmt.Errorf("failed to update status with snapshot reference: %v", err)
	}

	c.recorder.Event(templateRequest, corev1.EventTypeNormal, SnapshotCreated, "VirtualMachineSnapshot created")

	return nil
}

func (c *TemplateRequestController) checkSnapshotStatus(
	templateRequest *templatev1alpha1.VirtualMachineTemplateRequest,
) error {
	if templateRequest.Status.Snapshot == nil {
		return nil
	}

	// Validate snapshot reference has required fields
	if templateRequest.Status.Snapshot.Name == "" {
		log.Log.V(LogLevelVerbose).Infof("VirtualMachineTemplateRequest %s/%s has empty snapshot name in status, clearing reference",
			templateRequest.Namespace, templateRequest.Name)
		templateRequest.Status.Snapshot = nil
		c.setCondition(
			templateRequest,
			templatev1alpha1.VirtualMachineTemplateRequestConditionSnapshotReady,
			metav1.ConditionFalse,
			"InvalidReference",
			"Snapshot reference had empty name, cleared reference",
		)
		return nil
	}

	// Defensive check for namespace
	namespace := templateRequest.Namespace
	if templateRequest.Status.Snapshot.Namespace != nil && *templateRequest.Status.Snapshot.Namespace != "" {
		namespace = *templateRequest.Status.Snapshot.Namespace
	}

	key := fmt.Sprintf("%s/%s", namespace, templateRequest.Status.Snapshot.Name)
	log.Log.V(LogLevelVerbose).Infof("Looking up VirtualMachineSnapshot with key: %s", key)

	obj, exists, err := c.snapshotInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		log.Log.V(LogLevelInfo).Infof("VirtualMachineSnapshot %s not found in informer cache, but referenced from status. Snapshot status: name=%s, namespace=%v",
			key, templateRequest.Status.Snapshot.Name, templateRequest.Status.Snapshot.Namespace)
		c.setCondition(
			templateRequest,
			templatev1alpha1.VirtualMachineTemplateRequestConditionSnapshotReady,
			metav1.ConditionFalse,
			"NotFound",
			"Snapshot not found",
		)
		return nil
	}

	snapshot := obj.(*snapshotv1beta1.VirtualMachineSnapshot)

	// Check if snapshot is ready by examining both ReadyToUse field and conditions
	isReady := false
	readyReason := "InProgress"
	readyMessage := "Snapshot creation in progress"

	// Check ReadyToUse field
	if snapshot.Status != nil && snapshot.Status.ReadyToUse != nil && *snapshot.Status.ReadyToUse {
		isReady = true
		readyReason = "Ready"
		readyMessage = "Snapshot is ready"
	}

	// Also check conditions - this is the more reliable indicator
	if snapshot.Status != nil && len(snapshot.Status.Conditions) > 0 {
		for _, condition := range snapshot.Status.Conditions {
			if condition.Type == snapshotv1beta1.ConditionReady {
				if condition.Status == corev1.ConditionTrue {
					isReady = true
					readyReason = "Ready"
					readyMessage = fmt.Sprintf("Snapshot condition Ready is True: %s", condition.Message)
				} else {
					// If Ready condition exists but is False, use its message
					readyReason = condition.Reason
					if condition.Message != "" {
						readyMessage = condition.Message
					} else {
						readyMessage = "Snapshot not ready"
					}
				}
				break
			}
		}
	}

	if isReady {
		c.setCondition(
			templateRequest,
			templatev1alpha1.VirtualMachineTemplateRequestConditionSnapshotReady,
			metav1.ConditionTrue,
			readyReason,
			readyMessage,
		)
	} else {
		c.setCondition(
			templateRequest,
			templatev1alpha1.VirtualMachineTemplateRequestConditionSnapshotReady,
			metav1.ConditionFalse,
			readyReason,
			readyMessage,
		)
	}

	return nil
}

func (c *TemplateRequestController) handleTemplate(
	ctx context.Context,
	templateRequest *templatev1alpha1.VirtualMachineTemplateRequest,
	vm *virtv1.VirtualMachine,
) error {
	// Check if template already exists
	if templateRequest.Status.Template != nil {
		return c.checkTemplateStatus(templateRequest)
	}

	// Only create template if snapshot is ready
	if !c.isSnapshotReady(templateRequest) {
		return nil
	}

	// Create template from VM
	templateName := fmt.Sprintf("%s-template", templateRequest.Name)
	template := &templatev1alpha1.VirtualMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      templateName,
			Namespace: templateRequest.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(
					templateRequest,
					templatev1alpha1.SchemeGroupVersion.WithKind("VirtualMachineTemplateRequest"),
				),
			},
		},
		Spec: templatev1alpha1.VirtualMachineTemplateSpec{},
	}

	// Set the VirtualMachine data from the source VM
	vmCopy := vm.DeepCopy()
	// Clear fields that shouldn't be in the template
	vmCopy.ObjectMeta = metav1.ObjectMeta{
		Name: "${VM_NAME}",
	}
	vmCopy.Status = virtv1.VirtualMachineStatus{}

	// Ensure the VM object has proper type metadata for template processing
	vmCopy.TypeMeta = metav1.TypeMeta{
		APIVersion: virtv1.SchemeGroupVersion.String(),
		Kind:       "VirtualMachine",
	}

	// Rewrite DataVolumeTemplates to use VolumeSnapshots from the VirtualMachineSnapshot
	if err := c.rewriteDataVolumeTemplatesWithSnapshots(ctx, templateRequest, vmCopy); err != nil {
		return fmt.Errorf("failed to rewrite DataVolumeTemplates with snapshots: %v", err)
	}

	vmRaw, err := runtime.Encode(unstructured.UnstructuredJSONScheme, vmCopy)
	if err != nil {
		return fmt.Errorf("failed to encode VirtualMachine: %v", err)
	}

	template.Spec.VirtualMachine = runtime.RawExtension{Raw: vmRaw}

	// Add default parameters if not specified
	if template.Spec.Parameters == nil {
		template.Spec.Parameters = []templatev1alpha1.Parameter{
			{
				Name:        "VM_NAME",
				Description: "Name of the Virtual Machine",
				From:        fmt.Sprintf("vm-%s-[a-z0-9]{16}", template.Name),
				Generate:    "expression",
			},
		}
	}

	createdTemplate, err := c.clientset.GeneratedKubeVirtClient().TemplateV1alpha1().
		VirtualMachineTemplates(templateRequest.Namespace).Create(ctx, template, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create VirtualMachineTemplate: %v", err)
	}

	// Handle the case where template already exists
	if errors.IsAlreadyExists(err) {
		// Try to get the existing template
		existingTemplate, getErr := c.clientset.GeneratedKubeVirtClient().TemplateV1alpha1().
			VirtualMachineTemplates(templateRequest.Namespace).Get(ctx, templateName, metav1.GetOptions{})
		if getErr != nil {
			return fmt.Errorf("failed to get existing VirtualMachineTemplate: %v", getErr)
		}
		createdTemplate = existingTemplate
	}

	// Validate created template has required fields
	if createdTemplate.Name == "" {
		return fmt.Errorf("created VirtualMachineTemplate has empty name")
	}
	if createdTemplate.Namespace == "" {
		return fmt.Errorf("created VirtualMachineTemplate has empty namespace")
	}

	// Update status with template reference
	templateRequest.Status.Template = &corev1.TypedObjectReference{
		APIGroup:  pointer.P("template.kubevirt.io"),
		Kind:      "VirtualMachineTemplate",
		Name:      createdTemplate.Name,
		Namespace: pointer.P(createdTemplate.Namespace),
	}

	c.setCondition(
		templateRequest,
		templatev1alpha1.VirtualMachineTemplateRequestConditionTemplateReady,
		metav1.ConditionTrue,
		"Created",
		"Template created successfully",
	)

	// Immediately update status to persist the template reference
	if err := c.updateStatus(ctx, templateRequest); err != nil {
		return fmt.Errorf("failed to update status with template reference: %v", err)
	}

	c.recorder.Event(templateRequest, corev1.EventTypeNormal, TemplateCreated, "VirtualMachineTemplate created")

	return nil
}

func (c *TemplateRequestController) checkTemplateStatus(
	templateRequest *templatev1alpha1.VirtualMachineTemplateRequest,
) error {
	if templateRequest.Status.Template == nil {
		return nil
	}

	// Validate template reference has required fields
	if templateRequest.Status.Template.Name == "" {
		log.Log.V(LogLevelVerbose).Infof("VirtualMachineTemplateRequest %s/%s has empty template name in status, clearing reference",
			templateRequest.Namespace, templateRequest.Name)
		templateRequest.Status.Template = nil
		c.setCondition(
			templateRequest,
			templatev1alpha1.VirtualMachineTemplateRequestConditionTemplateReady,
			metav1.ConditionFalse,
			"InvalidReference",
			"Template reference had empty name, cleared reference",
		)
		return nil
	}

	// Defensive check for namespace
	namespace := templateRequest.Namespace
	if templateRequest.Status.Template.Namespace != nil && *templateRequest.Status.Template.Namespace != "" {
		namespace = *templateRequest.Status.Template.Namespace
	}

	key := fmt.Sprintf("%s/%s", namespace, templateRequest.Status.Template.Name)
	log.Log.V(LogLevelVerbose).Infof("Looking up VirtualMachineTemplate with key: %s", key)

	_, exists, err := c.templateInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		log.Log.V(LogLevelInfo).Infof("VirtualMachineTemplate %s not found in informer cache, but referenced from status. Template status: name=%s, namespace=%v",
			key, templateRequest.Status.Template.Name, templateRequest.Status.Template.Namespace)
		c.setCondition(
			templateRequest,
			templatev1alpha1.VirtualMachineTemplateRequestConditionTemplateReady,
			metav1.ConditionFalse,
			"NotFound",
			"Template not found",
		)
		return NewRequeueError(fmt.Sprintf("template %s not found within informer but referenced from status of request", key), requeueDelay)
	}

	c.setCondition(
		templateRequest,
		templatev1alpha1.VirtualMachineTemplateRequestConditionTemplateReady,
		metav1.ConditionTrue,
		"Ready",
		"Template is ready",
	)
	return nil
}

func (c *TemplateRequestController) isSnapshotReady(
	templateRequest *templatev1alpha1.VirtualMachineTemplateRequest,
) bool {
	for _, condition := range templateRequest.Status.Conditions {
		if condition.Type == templatev1alpha1.VirtualMachineTemplateRequestConditionSnapshotReady {
			return condition.Status == metav1.ConditionTrue
		}
	}
	return false
}

func (c *TemplateRequestController) isTemplateReady(
	templateRequest *templatev1alpha1.VirtualMachineTemplateRequest,
) bool {
	for _, condition := range templateRequest.Status.Conditions {
		if condition.Type == templatev1alpha1.VirtualMachineTemplateRequestConditionTemplateReady {
			return condition.Status == metav1.ConditionTrue
		}
	}
	return false
}

func (c *TemplateRequestController) setCondition(
	templateRequest *templatev1alpha1.VirtualMachineTemplateRequest,
	conditionType templatev1alpha1.VirtualMachineTemplateRequestConditionType,
	status metav1.ConditionStatus,
	reason, message string,
) {
	now := metav1.Now()

	// Find existing condition
	for i, condition := range templateRequest.Status.Conditions {
		if condition.Type == conditionType {
			if condition.Status != status {
				templateRequest.Status.Conditions[i].Status = status
				templateRequest.Status.Conditions[i].LastTransitionTime = now
				templateRequest.Status.Conditions[i].Reason = reason
				templateRequest.Status.Conditions[i].Message = message
			}
			return
		}
	}

	// Add new condition
	templateRequest.Status.Conditions = append(templateRequest.Status.Conditions,
		templatev1alpha1.VirtualMachineTemplateRequestCondition{
			Type:               conditionType,
			Status:             status,
			LastTransitionTime: now,
			Reason:             reason,
			Message:            message,
		})
}

func (c *TemplateRequestController) handleError(
	ctx context.Context,
	templateRequest *templatev1alpha1.VirtualMachineTemplateRequest,
	reason string,
	err error,
) error {
	templateRequest.Status.Phase = templatev1alpha1.VirtualMachineTemplateRequestPhaseFailed
	c.setCondition(
		templateRequest,
		templatev1alpha1.VirtualMachineTemplateRequestConditionReady,
		metav1.ConditionFalse,
		reason,
		err.Error(),
	)

	if updateErr := c.updateStatus(ctx, templateRequest); updateErr != nil {
		log.Log.Object(templateRequest).Errorf("Failed to update status: %v", updateErr)
	}

	return common.NewSyncError(err, reason)
}

func (c *TemplateRequestController) updateStatus(
	ctx context.Context,
	templateRequest *templatev1alpha1.VirtualMachineTemplateRequest,
) error {
	// Check if status has actually changed to avoid unnecessary updates
	key := fmt.Sprintf("%s/%s", templateRequest.Namespace, templateRequest.Name)
	current, exists, err := c.templateRequestInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("VirtualMachineTemplateRequest %s not found in cache", key)
	}

	currentRequest := current.(*templatev1alpha1.VirtualMachineTemplateRequest)
	if equality.Semantic.DeepEqual(currentRequest.Status, templateRequest.Status) {
		return nil
	}

	// Use retry logic to handle conflicts
	return c.updateStatusWithRetry(ctx, templateRequest)
}

func (c *TemplateRequestController) updateStatusWithRetry(
	ctx context.Context,
	templateRequest *templatev1alpha1.VirtualMachineTemplateRequest,
) error {
	client := c.clientset.GeneratedKubeVirtClient().TemplateV1alpha1().VirtualMachineTemplateRequests(templateRequest.Namespace)

	// Retry up to 5 times with exponential backoff
	return wait.ExponentialBackoffWithContext(ctx, wait.Backoff{
		Steps:    5,
		Duration: 100 * time.Millisecond,
		Factor:   2.0,
		Jitter:   0.1,
	}, func(ctx context.Context) (bool, error) {
		// Get the latest version from the API server
		latest, err := client.Get(ctx, templateRequest.Name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// Object was deleted, nothing to update
				return true, nil
			}
			// Retry on other errors
			return false, err
		}

		// Check if status has changed since we started
		if equality.Semantic.DeepEqual(latest.Status, templateRequest.Status) {
			// Status is already up to date
			return true, nil
		}

		// Update the status on the latest version
		latest.Status = templateRequest.Status
		latest.Status.ObservedGeneration = latest.Generation

		_, err = client.UpdateStatus(ctx, latest, metav1.UpdateOptions{})
		if err != nil {
			if errors.IsConflict(err) {
				// Conflict error, retry
				log.Log.V(LogLevelVerbose).Infof("Conflict updating VirtualMachineTemplateRequest %s/%s status, retrying",
					templateRequest.Namespace, templateRequest.Name)
				return false, nil
			}
			// Other errors should not be retried
			c.recorder.Event(
				templateRequest,
				corev1.EventTypeWarning,
				FailedUpdateStatus,
				fmt.Sprintf("Failed to update status: %v", err),
			)
			return false, fmt.Errorf("failed to update VirtualMachineTemplateRequest status: %v", err)
		}

		// Success
		return true, nil
	})
}

func (c *TemplateRequestController) enqueueTemplateRequest(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err != nil {
		log.Log.Errorf("couldn't get key for object %+v: %v", obj, err)
		return
	}
	c.queue.Add(key)
}

func (c *TemplateRequestController) enqueueTemplateRequestForSnapshot(obj interface{}) {
	snapshot, ok := obj.(*snapshotv1beta1.VirtualMachineSnapshot)
	if !ok {
		log.Log.Errorf("expected VirtualMachineSnapshot, got %T", obj)
		return
	}

	// Find the owning VirtualMachineTemplateRequest
	ownerRef := c.getOwnerRef(snapshot.OwnerReferences, "VirtualMachineTemplateRequest")
	if ownerRef != nil {
		key := fmt.Sprintf("%s/%s", snapshot.Namespace, ownerRef.Name)
		log.Log.V(LogLevelVerbose).Infof("Enqueueing VirtualMachineTemplateRequest %s due to VirtualMachineSnapshot %s change", key, snapshot.Name)
		c.queue.Add(key)
	}
}

func (c *TemplateRequestController) enqueueTemplateRequestForTemplate(obj interface{}) {
	template, ok := obj.(*templatev1alpha1.VirtualMachineTemplate)
	if !ok {
		log.Log.Errorf("expected VirtualMachineTemplate, got %T", obj)
		return
	}

	// Find the owning VirtualMachineTemplateRequest
	ownerRef := c.getOwnerRef(template.OwnerReferences, "VirtualMachineTemplateRequest")
	if ownerRef != nil {
		key := fmt.Sprintf("%s/%s", template.Namespace, ownerRef.Name)
		log.Log.V(LogLevelVerbose).Infof("Enqueueing VirtualMachineTemplateRequest %s due to VirtualMachineTemplate %s change", key, template.Name)
		c.queue.Add(key)
	}
}

func (c *TemplateRequestController) rewriteDataVolumeTemplatesWithSnapshots(
	ctx context.Context,
	templateRequest *templatev1alpha1.VirtualMachineTemplateRequest,
	vmCopy *virtv1.VirtualMachine,
) error {
	// Check if we have a snapshot reference
	if templateRequest.Status.Snapshot == nil {
		log.Log.V(LogLevelVerbose).Infof("No snapshot reference found for template request %s/%s, skipping DataVolumeTemplate rewrite",
			templateRequest.Namespace, templateRequest.Name)
		return nil
	}

	// Get the VirtualMachineSnapshot
	snapshotNamespace := templateRequest.Namespace
	if templateRequest.Status.Snapshot.Namespace != nil {
		snapshotNamespace = *templateRequest.Status.Snapshot.Namespace
	}

	key := fmt.Sprintf("%s/%s", snapshotNamespace, templateRequest.Status.Snapshot.Name)
	obj, exists, err := c.snapshotInformer.GetStore().GetByKey(key)
	if err != nil {
		return fmt.Errorf("failed to get VirtualMachineSnapshot from informer: %v", err)
	}
	if !exists {
		log.Log.V(LogLevelInfo).Infof("VirtualMachineSnapshot %s not found in informer, skipping DataVolumeTemplate rewrite", key)
		return nil
	}

	vmSnapshot := obj.(*snapshotv1beta1.VirtualMachineSnapshot)

	// Get the VirtualMachineSnapshotContent
	if vmSnapshot.Status == nil || vmSnapshot.Status.VirtualMachineSnapshotContentName == nil {
		log.Log.V(LogLevelInfo).Infof("VirtualMachineSnapshot %s has no content reference, skipping DataVolumeTemplate rewrite", key)
		return nil
	}

	contentName := *vmSnapshot.Status.VirtualMachineSnapshotContentName
	contentKey := fmt.Sprintf("%s/%s", snapshotNamespace, contentName)

	// Get the content from API server since we don't have an informer for it
	content, err := c.clientset.VirtualMachineSnapshotContent(snapshotNamespace).Get(ctx, contentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get VirtualMachineSnapshotContent %s: %v", contentKey, err)
	}

	// Check if the content has VolumeBackups
	if len(content.Spec.VolumeBackups) == 0 {
		log.Log.V(LogLevelVerbose).Infof("VirtualMachineSnapshotContent %s has no VolumeBackups, skipping DataVolumeTemplate rewrite", contentKey)
		return nil
	}

	// Create a map of PVC name to VolumeSnapshot name for quick lookup
	pvcToVolumeSnapshot := make(map[string]string)
	for _, volumeBackup := range content.Spec.VolumeBackups {
		if volumeBackup.VolumeSnapshotName != nil {
			pvcToVolumeSnapshot[volumeBackup.PersistentVolumeClaim.Name] = *volumeBackup.VolumeSnapshotName
		}
	}

	// Rewrite DataVolumeTemplates in the VM spec
	if len(vmCopy.Spec.DataVolumeTemplates) > 0 {
		// Keep track of name changes to update volume references
		dvtNameMapping := make(map[string]string)

		for i, dvt := range vmCopy.Spec.DataVolumeTemplates {
			// Check if this DataVolumeTemplate has a corresponding VolumeSnapshot
			if volumeSnapshotName, exists := pvcToVolumeSnapshot[dvt.Name]; exists {
				log.Log.V(LogLevelInfo).Infof("Rewriting DataVolumeTemplate %s to use VolumeSnapshot %s", dvt.Name, volumeSnapshotName)

				// Create a new DataVolumeTemplate that uses the VolumeSnapshot as source
				newDVT := dvt.DeepCopy()

				// Prefix the name with a template parameter to avoid conflicts
				originalName := newDVT.Name
				newDVT.Name = fmt.Sprintf("${VM_NAME}-%s", originalName)
				dvtNameMapping[originalName] = newDVT.Name

				newDVT.Spec.Source = &cdiv1.DataVolumeSource{
					Snapshot: &cdiv1.DataVolumeSourceSnapshot{
						Namespace: snapshotNamespace,
						Name:      volumeSnapshotName,
					},
				}

				// Clear other source fields to avoid conflicts
				newDVT.Spec.SourceRef = nil

				vmCopy.Spec.DataVolumeTemplates[i] = *newDVT
			}
		}

		// Update volume references in the VM template spec to match the new DataVolumeTemplate names
		if vmCopy.Spec.Template != nil && vmCopy.Spec.Template.Spec.Volumes != nil {
			for i, volume := range vmCopy.Spec.Template.Spec.Volumes {
				if volume.DataVolume != nil {
					if newName, exists := dvtNameMapping[volume.DataVolume.Name]; exists {
						log.Log.V(LogLevelVerbose).Infof("Updating volume %s DataVolume reference from %s to %s",
							volume.Name, volume.DataVolume.Name, newName)
						vmCopy.Spec.Template.Spec.Volumes[i].DataVolume.Name = newName
					}
				}
			}
		}
	}

	log.Log.V(LogLevelInfo).Infof("Successfully rewrote DataVolumeTemplates for VM template based on VirtualMachineSnapshotContent %s", contentKey)
	return nil
}

func (c *TemplateRequestController) getOwnerRef(ownerRefs []metav1.OwnerReference, kind string) *metav1.OwnerReference {
	for _, ownerRef := range ownerRefs {
		if ownerRef.Kind == kind {
			return &ownerRef
		}
	}
	return nil
}
