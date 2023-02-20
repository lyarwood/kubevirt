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
 * Copyright 2023 Red Hat, Inc.
 *
 */
package instancetype_test

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	"sigs.k8s.io/yaml"

	. "kubevirt.io/kubevirt/pkg/virtctl/create/instancetype"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

const (
	create     = "create"
	namespaced = "--namespaced"
)

var _ = Describe("create", func() {
	Context("instancetype without arguments", func() {
		DescribeTable("should fail because of required cpu and memory", func(namespaced string) {
			_, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespaced)()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("required flag(s) \"cpu\", \"memory\" not set"))
		},
			Entry("VirtualMachineInstancetype", namespaced),
			Entry("VirtualMachineClusterInstancetype", ""),
		)
	})

	Context("instancetype with arguments", func() {
		var instancetypeSpec *instancetypev1alpha2.VirtualMachineInstancetypeSpec

		DescribeTable("should succeed with defined cpu and memory", func(namespaced string) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespaced,
				setFlag(CPUFlag, "2"),
				setFlag(MemoryFlag, "256Mi"),
			)()
			Expect(err).ToNot(HaveOccurred())

			instancetypeSpec = getInstancetypeSpec(namespaced, bytes)
			Expect(instancetypeSpec.CPU.Guest).To(Equal(uint32(2)))
			Expect(instancetypeSpec.Memory.Guest).To(Equal(resource.MustParse("256Mi")))
		},
			Entry("VirtualMachineInstancetype", namespaced),
			Entry("VirtualMachineClusterInstancetype", ""),
		)

		DescribeTable("should succeed with defined gpus", func(namespaced string) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespaced,
				setFlag(CPUFlag, "1"),
				setFlag(MemoryFlag, "128Mi"),
				setFlag(GPUFlag, "name:gpu1,devicename:nvidia"),
			)()
			Expect(err).ToNot(HaveOccurred())

			instancetypeSpec = getInstancetypeSpec(namespaced, bytes)
			Expect(instancetypeSpec.GPUs).To(HaveLen(1))
			Expect(instancetypeSpec.GPUs[0].Name).To(Equal("gpu1"))
			Expect(instancetypeSpec.GPUs[0].DeviceName).To(Equal("nvidia"))
		},
			Entry("VirtualMachineInstancetype", namespaced),
			Entry("VirtualMachineClusterInstancetype", ""),
		)

		DescribeTable("should succeed with defined hostDevices", func(namespaced string) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespaced,
				setFlag(CPUFlag, "1"),
				setFlag(MemoryFlag, "128Mi"),
				setFlag(HostDeviceFlag, "name:device1,devicename:intel"),
			)()
			Expect(err).ToNot(HaveOccurred())

			instancetypeSpec = getInstancetypeSpec(namespaced, bytes)
			Expect(instancetypeSpec.HostDevices).To(HaveLen(1))
			Expect(instancetypeSpec.HostDevices[0].Name).To(Equal("device1"))
			Expect(instancetypeSpec.HostDevices[0].DeviceName).To(Equal("intel"))
		},
			Entry("VirtualMachineInstancetype", namespaced),
			Entry("VirtualMachineClusterInstancetype", ""),
		)

		DescribeTable("should succeed with valid IOThreadsPolicy", func(namespaced, param string, policy v1.IOThreadsPolicy) {
			bytes, err := clientcmd.NewRepeatableVirtctlCommandWithOut(create, Instancetype, namespaced,
				setFlag(CPUFlag, "1"),
				setFlag(MemoryFlag, "128Mi"),
				setFlag(IOThreadsPolicyFlag, param),
			)()
			Expect(err).ToNot(HaveOccurred())

			instancetypeSpec := getInstancetypeSpec(namespaced, bytes)
			Expect(*instancetypeSpec.IOThreadsPolicy).To(Equal(policy))

		},
			Entry("VirtualMachineInstacetype set to auto", namespaced, "auto", v1.IOThreadsPolicyAuto),
			Entry("VirtualMachineInstacetype set to shared", namespaced, "shared", v1.IOThreadsPolicyShared),

			Entry("VirtualMachineClusterInstacetype set to auto", "", "auto", v1.IOThreadsPolicyAuto),
			Entry("VirtualMachineClusterInstacetype set to shared", "", "shared", v1.IOThreadsPolicyShared),
		)

		DescribeTable("invalid cpu and memory", func(cpu, memory, errMsg string) {
			err := clientcmd.NewRepeatableVirtctlCommand(create, Instancetype, namespaced,
				setFlag(CPUFlag, cpu),
				setFlag(MemoryFlag, memory),
			)()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(errMsg))
		},
			Entry("Ivalid cpu value", "two", "256Mi", "invalid argument \"two\" for \"--cpu\" flag: strconv.ParseUint: parsing \"two\": invalid syntax"),
			Entry("Ivalid cpu value", "-2", "256Mi", "invalid argument \"-2\" for \"--cpu\" flag: strconv.ParseUint: parsing \"-2\": invalid syntax"),
			Entry("Invalid memory value", "2", "256My", "quantities must match the regular expression '^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$'"),
		)

		DescribeTable("Invalid arguments", func(namespaced, flag, params, errMsg string) {
			err := clientcmd.NewRepeatableVirtctlCommand(create, Instancetype, namespaced,
				setFlag(CPUFlag, "1"),
				setFlag(MemoryFlag, "128Mi"),
				setFlag(flag, params),
			)()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(errMsg))
		},
			Entry("VirtualMachineInstacetype gpu missing name", namespaced, GPUFlag, "devicename:nvidia", fmt.Sprintf("failed to parse \"--gpu\" flag: %+s", NameErr)),
			Entry("VirtualMachineInstacetype gpu missing deviceName", namespaced, GPUFlag, "name:gpu1", fmt.Sprintf("failed to parse \"--gpu\" flag: %+s", DeviceNameErr)),
			Entry("VirtualMachineInstacetype hostdevice missing name", namespaced, HostDeviceFlag, "devicename:intel", fmt.Sprintf("failed to parse \"--hostdevice\" flag: %+s", NameErr)),
			Entry("VirtualMachineInstacetype hostdevice missing deviceName", namespaced, HostDeviceFlag, "name:device1", fmt.Sprintf("failed to parse \"--hostdevice\" flag: %+s", DeviceNameErr)),
			Entry("VirtualMachineInstacetype to IOThreadsPolicy", namespaced, IOThreadsPolicyFlag, "invalid-policy", fmt.Sprintf("failed to parse \"--iothreadspolicy\" flag: %+s", IOThreadErr)),

			Entry("VirtualMachineClusterInstacetype gpu missing name", "", GPUFlag, "devicename:nvidia", fmt.Sprintf("failed to parse \"--gpu\" flag: %+s", NameErr)),
			Entry("VirtualMachineClusterInstacetype gpu missing deviceName", "", GPUFlag, "name:gpu1", fmt.Sprintf("failed to parse \"--gpu\" flag: %+s", DeviceNameErr)),
			Entry("VirtualMachineClusterInstacetype hostdevice missing name", "", HostDeviceFlag, "devicename:intel", fmt.Sprintf("failed to parse \"--hostdevice\" flag: %+s", NameErr)),
			Entry("VirtualMachineClusterInstacetype hostdevice missing deviceName", "", HostDeviceFlag, "name:device1", fmt.Sprintf("failed to parse \"--hostdevice\" flag: %+s", DeviceNameErr)),
			Entry("VirtualMachineClusterInstacetype to IOThreadsPolicy", "", IOThreadsPolicyFlag, "invalid-policy", fmt.Sprintf("failed to parse \"--iothreadspolicy\" flag: %+s", IOThreadErr)),
		)
	})
})

func setFlag(flag, parameter string) string {
	return fmt.Sprintf("--%s=%s", flag, parameter)
}

func getInstancetypeSpec(namespaced string, bytes []byte) *instancetypev1alpha2.VirtualMachineInstancetypeSpec {
	if namespaced == "" {
		return unmarshalClusterInstanceType(bytes)
	}

	return unmarshalInstanceType(bytes)
}

func unmarshalInstanceType(bytes []byte) *instancetypev1alpha2.VirtualMachineInstancetypeSpec {
	instancetype := &instancetypev1alpha2.VirtualMachineInstancetype{}
	Expect(yaml.Unmarshal(bytes, instancetype)).To(Succeed())
	Expect(strings.ToLower(instancetype.Kind)).To(Equal(apiinstancetype.SingularResourceName))
	Expect(instancetype.APIVersion).To(Equal(instancetypev1alpha2.SchemeGroupVersion.String()))

	return &instancetype.Spec
}

func unmarshalClusterInstanceType(bytes []byte) *instancetypev1alpha2.VirtualMachineInstancetypeSpec {
	clusterInstancetype := &instancetypev1alpha2.VirtualMachineClusterInstancetype{}
	Expect(yaml.Unmarshal(bytes, clusterInstancetype)).To(Succeed())
	Expect(strings.ToLower(clusterInstancetype.Kind)).To(Equal(apiinstancetype.ClusterSingularResourceName))
	Expect(clusterInstancetype.APIVersion).To(Equal(instancetypev1alpha2.SchemeGroupVersion.String()))

	return &clusterInstancetype.Spec
}
