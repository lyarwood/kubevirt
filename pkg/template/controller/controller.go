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
	RequeueDelay = 30 * time.Second
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
			return NewRequeueError("Waiting for VirtualMachineSnapshot to become ready", RequeueDelay)
		}
		if !c.isTemplateReady(templateRequest) {
			return NewRequeueError("Waiting for VirtualMachineTemplate to become ready", RequeueDelay)
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
	c.recorder.Event(templateRequest, corev1.EventTypeNormal, SnapshotCreated, "VirtualMachineSnapshot created")

	return nil
}

func (c *TemplateRequestController) checkSnapshotStatus(
	templateRequest *templatev1alpha1.VirtualMachineTemplateRequest,
) error {
	if templateRequest.Status.Snapshot == nil {
		return nil
	}

	key := fmt.Sprintf("%s/%s", *templateRequest.Status.Snapshot.Namespace, templateRequest.Status.Snapshot.Name)
	obj, exists, err := c.snapshotInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
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
				Required:    true,
			},
		}
	}

	createdTemplate, err := c.clientset.GeneratedKubeVirtClient().TemplateV1alpha1().
		VirtualMachineTemplates(templateRequest.Namespace).Create(ctx, template, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create VirtualMachineTemplate: %v", err)
	}

	// Update status with template reference
	templateRequest.Status.Template = &corev1.TypedObjectReference{
		Kind:      "VirtualMachineTemplate",
		Name:      createdTemplate.Name,
		Namespace: &createdTemplate.Namespace,
	}

	c.setCondition(
		templateRequest,
		templatev1alpha1.VirtualMachineTemplateRequestConditionTemplateReady,
		metav1.ConditionTrue,
		"Created",
		"Template created successfully",
	)
	c.recorder.Event(templateRequest, corev1.EventTypeNormal, TemplateCreated, "VirtualMachineTemplate created")

	return nil
}

func (c *TemplateRequestController) checkTemplateStatus(
	templateRequest *templatev1alpha1.VirtualMachineTemplateRequest,
) error {
	if templateRequest.Status.Template == nil {
		return nil
	}

	key := fmt.Sprintf("%s/%s", *templateRequest.Status.Template.Namespace, templateRequest.Status.Template.Name)
	_, exists, err := c.templateInformer.GetStore().GetByKey(key)
	if err != nil {
		return err
	}
	if !exists {
		c.setCondition(
			templateRequest,
			templatev1alpha1.VirtualMachineTemplateRequestConditionTemplateReady,
			metav1.ConditionFalse,
			"NotFound",
			"Template not found",
		)
		return fmt.Errorf("template %s not found", key)
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

func (c *TemplateRequestController) getOwnerRef(ownerRefs []metav1.OwnerReference, kind string) *metav1.OwnerReference {
	for _, ownerRef := range ownerRefs {
		if ownerRef.Kind == kind {
			return &ownerRef
		}
	}
	return nil
}
