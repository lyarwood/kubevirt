package virtctl

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	"kubevirt.io/client-go/kubecli"
	"sigs.k8s.io/yaml"

	. "kubevirt.io/kubevirt/pkg/virtctl/create/instancetype"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/util"
)

const namespaced = "--namespaced"

var _ = Describe("create instancetype", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("should create valid instancetype manifest", func() {
		DescribeTable("when CPU and Memory defined", func(namespaced string) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespaced,
				setFlag(CPUFlag, "2"),
				setFlag(MemoryFlag, "256Mi"),
			)()
			Expect(err).ToNot(HaveOccurred())
			instancetypeSpec := getInstancetypeSpec(namespaced, bytes, virtClient)

			Expect(instancetypeSpec.CPU.Guest).To(Equal(uint32(2)))
			Expect(instancetypeSpec.Memory.Guest).To(Equal(resource.MustParse("256Mi")))
		},
			Entry("VirtualMachineInstancetype", namespaced),
			Entry("VirtualMachineClusterInstancetype", ""),
		)

		DescribeTable("when GPUs defined", func(namespaced string) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespaced,
				setFlag(CPUFlag, "2"),
				setFlag(MemoryFlag, "256Mi"),
				setFlag(GPUFlag, "name:gpu1,devicename:nvidia/gpu1"),
			)()
			Expect(err).ToNot(HaveOccurred())
			instancetypeSpec := getInstancetypeSpec(namespaced, bytes, virtClient)

			Expect(instancetypeSpec.GPUs[0].Name).To(Equal("gpu1"))
			Expect(instancetypeSpec.GPUs[0].DeviceName).To(Equal("nvidia/gpu1"))
		},
			Entry("VirtualMachineInstancetype", namespaced),
			Entry("VirtualMachineClusterInstancetype", ""),
		)

		DescribeTable("when hostDevice defined", func(namespaced string) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespaced,
				setFlag(CPUFlag, "2"),
				setFlag(MemoryFlag, "256Mi"),
				setFlag(HostDeviceFlag, "name:device1,devicename:hostdevice1"),
			)()
			Expect(err).ToNot(HaveOccurred())
			instancetypeSpec := getInstancetypeSpec(namespaced, bytes, virtClient)

			Expect(instancetypeSpec.HostDevices[0].Name).To(Equal("device1"))
			Expect(instancetypeSpec.HostDevices[0].DeviceName).To(Equal("hostdevice1"))
		},
			Entry("VirtualMachineInstancetype", namespaced),
			Entry("VirtualMachineClusterInstancetype", ""),
		)

		DescribeTable("when IOThreadsPolicy defined", func(namespaced, policyStr string, policy v1.IOThreadsPolicy) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespaced,
				setFlag(CPUFlag, "2"),
				setFlag(MemoryFlag, "256Mi"),
				setFlag(IOThreadsPolicyFlag, policyStr),
			)()
			Expect(err).ToNot(HaveOccurred())
			instancetypeSpec := getInstancetypeSpec(namespaced, bytes, virtClient)

			Expect(*instancetypeSpec.IOThreadsPolicy).To(Equal(policy))
		},
			Entry("VirtualMachineInstancetype", namespaced, "auto", v1.IOThreadsPolicyAuto),
			Entry("VirtualMachineClusterInstancetype", "", "shared", v1.IOThreadsPolicyShared),
		)
	})
})

func getInstancetypeSpec(namespaced string, bytes []byte, virtClient kubecli.KubevirtClient) *instancetypev1alpha2.VirtualMachineInstancetypeSpec {
	if namespaced == "" {
		clusterInstancetype, err := virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), unmarshalClusterInstanceType(bytes), metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		return &clusterInstancetype.Spec
	}

	instancetype, err := virtClient.VirtualMachineInstancetype(util.NamespaceTestDefault).Create(context.Background(), unmarshalInstanceType(bytes), metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	return &instancetype.Spec
}

func unmarshalInstanceType(bytes []byte) *instancetypev1alpha2.VirtualMachineInstancetype {
	instancetype := &instancetypev1alpha2.VirtualMachineInstancetype{}
	Expect(yaml.Unmarshal(bytes, instancetype)).ToNot(HaveOccurred())
	return instancetype
}

func unmarshalClusterInstanceType(bytes []byte) *instancetypev1alpha2.VirtualMachineClusterInstancetype {
	clusterInstancetype := &instancetypev1alpha2.VirtualMachineClusterInstancetype{}
	Expect(yaml.Unmarshal(bytes, clusterInstancetype)).ToNot(HaveOccurred())
	return clusterInstancetype
}
