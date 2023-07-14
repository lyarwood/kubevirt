package instancetype

import (
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/instancetype"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/tests/util"
)

type MockUpgrader struct {
	UpgradeFn func(*appsv1.ControllerRevision) (*appsv1.ControllerRevision, error)
}

func (u *MockUpgrader) Upgrade(original *appsv1.ControllerRevision) (*appsv1.ControllerRevision, error) {
	return u.UpgradeFn(original)
}

var _ instancetype.UpgraderInterface = &MockUpgrader{}

func newMockUpgrader() *MockUpgrader {
	return &MockUpgrader{
		UpgradeFn: func(*appsv1.ControllerRevision) (*appsv1.ControllerRevision, error) {
			return nil, nil
		},
	}
}

var _ = Describe("UpgradeController", func() {
	var (
		controller *UpgradeController

		vmInformer        cache.SharedIndexInformer
		crInformer        cache.SharedIndexInformer
		crUpgradeInformer cache.SharedIndexInformer
		crUpgradeSource   *framework.FakeControllerSource
		recorder          *record.FakeRecorder
		mockQueue         *testutils.MockWorkQueue

		client *kubevirtfake.Clientset
		ctrl   *gomock.Controller
		err    error
		stop   chan struct{}
	)

	const (
		crUpgradeName = "controllerrevisionupgrade"
		crName        = "controllerrevision"
	)

	syncCaches := func(stop chan struct{}) {
		go vmInformer.Run(stop)
		go crInformer.Run(stop)
		go crUpgradeInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop, vmInformer.HasSynced, crInformer.HasSynced, crUpgradeInformer.HasSynced)).To(BeTrue())
	}

	BeforeEach(func() {
		vmInformer, _ = testutils.NewFakeInformerFor(&virtv1.VirtualMachine{})
		crInformer, _ = testutils.NewFakeInformerFor(&appsv1.ControllerRevision{})
		crUpgradeInformer, crUpgradeSource = testutils.NewFakeInformerFor(&instancetypev1beta1.ControllerRevisionUpgrade{})

		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		ctrl = gomock.NewController(GinkgoT())
		virtClient := kubecli.NewMockKubevirtClient(ctrl)

		client = kubevirtfake.NewSimpleClientset()
		virtClient.EXPECT().ControllerRevisionUpgrade(util.NamespaceTestDefault).Return(client.InstancetypeV1beta1().ControllerRevisionUpgrades(util.NamespaceTestDefault)).AnyTimes()

		client.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})

		controller, err = NewUpgradeController(virtClient, recorder, vmInformer, crInformer, crUpgradeInformer)
		Expect(err).ToNot(HaveOccurred())

		// Overwrite the upgrader within the controller with a mocked version for testing
		controller.upgrader = newMockUpgrader()

		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue

		stop = make(chan struct{})
		syncCaches(stop)
	})

	addCR := func(cr *appsv1.ControllerRevision) {
		Expect(crInformer.GetStore().Add(cr)).To(Succeed())
	}

	addCRUpgrade := func(crUpgrade *instancetypev1beta1.ControllerRevisionUpgrade) {
		mockQueue.ExpectAdds(1)
		crUpgradeSource.Add(crUpgrade)
		mockQueue.Wait()
	}

	addInProgressUpgrade := func() {
		inprogress := instancetypev1beta1.UpgradeInProgress
		addCRUpgrade(&instancetypev1beta1.ControllerRevisionUpgrade{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: util.NamespaceTestDefault,
				Name:      crUpgradeName,
			},
			Spec: &instancetypev1beta1.ControllerRevisionUpgradeSpec{
				TargetName: crName,
			},
			Status: &instancetypev1beta1.ControllerRevisionUpgradeStatus{
				Phase: &inprogress,
			},
		})
	}

	addUnsetUpgrade := func() {
		addCRUpgrade(&instancetypev1beta1.ControllerRevisionUpgrade{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: util.NamespaceTestDefault,
				Name:      crUpgradeName,
			},
			Spec: &instancetypev1beta1.ControllerRevisionUpgradeSpec{
				TargetName: crName,
			},
		})
	}

	expectUpgradePhase := func(phase instancetypev1beta1.ControllerRevisionUpgradePhase) {
		client.Fake.PrependReactor("update", instancetypeapi.PluralControllerRevisionUpgradeResourceName, func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())

			crUpgrade := update.GetObject().(*instancetypev1beta1.ControllerRevisionUpgrade)
			Expect(*crUpgrade.Status.Phase).To(Equal(phase))

			return true, update.GetObject(), nil
		})
	}

	expectUpdatePhaseToInProgress := func() {
		expectUpgradePhase(instancetypev1beta1.UpgradeInProgress)
	}

	expectUpdatePhaseToFailed := func() {
		expectUpgradePhase(instancetypev1beta1.UpgradeFailed)
	}

	expectUpdatePhaseToSucceeded := func() {
		expectUpgradePhase(instancetypev1beta1.UpgradeSucceeded)
	}

	assertExecutedWithoutReenqueue := func() {
		Expect(mockQueue.Len()).To(Equal(1))
		Expect(controller.Execute()).To(BeTrue())
		Expect(mockQueue.Len()).To(Equal(0))
		Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(0))
	}

	assertExecutedWithReenqueue := func() {
		Expect(mockQueue.Len()).To(Equal(1))
		Expect(controller.Execute()).To(BeTrue())
		Expect(mockQueue.Len()).To(Equal(0))
		Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(1))
	}

	Context("ControllerRevisionUpgrade", func() {
		BeforeEach(func() {
			addCR(&appsv1.ControllerRevision{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: util.NamespaceTestDefault,
					Name:      crName,
				},
			})
		})

		It("should reenqueue on failure to find ControllerRevisionUpgrade", func() {
			mockQueue.Add("non-existing-crUpgrade-key")
			assertExecutedWithReenqueue()
		})

		It("should be ignored if phase already successful", func() {
			succeeded := instancetypev1beta1.UpgradeSucceeded
			addCRUpgrade(&instancetypev1beta1.ControllerRevisionUpgrade{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: util.NamespaceTestDefault,
					Name:      crUpgradeName,
				},
				Spec: &instancetypev1beta1.ControllerRevisionUpgradeSpec{
					TargetName: crName,
				},
				Status: &instancetypev1beta1.ControllerRevisionUpgradeStatus{
					Phase: &succeeded,
				},
			})
			assertExecutedWithoutReenqueue()
		})

		It("should update new upgrade phase to in-progress", func() {
			addUnsetUpgrade()
			expectUpdatePhaseToInProgress()
			assertExecutedWithoutReenqueue()
		})

		It("mark completed upgrade as succeeded", func() {
			addInProgressUpgrade()
			expectUpdatePhaseToSucceeded()
			assertExecutedWithoutReenqueue()
		})

		It("should mark upgrade as failed when unable to find target ControllerRevision", func() {
			inprogress := instancetypev1beta1.UpgradeInProgress
			addCRUpgrade(&instancetypev1beta1.ControllerRevisionUpgrade{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: util.NamespaceTestDefault,
					Name:      crUpgradeName,
				},
				Spec: &instancetypev1beta1.ControllerRevisionUpgradeSpec{
					TargetName: "non-existing-cr",
				},
				Status: &instancetypev1beta1.ControllerRevisionUpgradeStatus{
					Phase: &inprogress,
				},
			})
			expectUpdatePhaseToFailed()
			assertExecutedWithReenqueue()
		})

		It("should mark failed upgrade as failed", func() {
			controller.upgrader = &MockUpgrader{
				UpgradeFn: func(original *appsv1.ControllerRevision) (*appsv1.ControllerRevision, error) {
					return nil, fmt.Errorf("failure")
				},
			}
			addInProgressUpgrade()
			expectUpdatePhaseToFailed()
			assertExecutedWithReenqueue()
		})
	})
})
