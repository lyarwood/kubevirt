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
 */

package apply_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("instancetype.Spec.Devices", func() {
	var (
		vmi            *virtv1.VirtualMachineInstance
		preferenceSpec *v1beta1.VirtualMachinePreferenceSpec
		vmiApplier     = apply.NewVMIApplier()
		field          = k8sfield.NewPath("spec", "template", "spec")
	)

	BeforeEach(func() {
		vmi = libvmi.New()
	})

	DescribeTable("should apply devices to VMI", func(
		instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
		getDevices func(spec *virtv1.VirtualMachineInstanceSpec) any,
	) {
		Expect(vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)).To(Succeed())

		if len(instancetypeSpec.GPUs) > 0 {
			Expect(getDevices(&vmi.Spec)).To(Equal(instancetypeSpec.GPUs))
		} else if len(instancetypeSpec.HostDevices) > 0 {
			Expect(getDevices(&vmi.Spec)).To(Equal(instancetypeSpec.HostDevices))
		}
	},
		Entry("when applying GPUs",
			&v1beta1.VirtualMachineInstancetypeSpec{
				GPUs: []virtv1.GPU{
					{
						Name:       "barfoo",
						DeviceName: "vendor.com/gpu_name",
					},
				},
			},
			func(spec *virtv1.VirtualMachineInstanceSpec) any {
				return spec.Domain.Devices.GPUs
			},
		),
		Entry("when applying HostDevices",
			&v1beta1.VirtualMachineInstancetypeSpec{
				HostDevices: []virtv1.HostDevice{
					{
						Name:       "foobar",
						DeviceName: "vendor.com/device_name",
					},
				},
			},
			func(spec *virtv1.VirtualMachineInstanceSpec) any {
				return spec.Domain.Devices.HostDevices
			},
		),
	)

	DescribeTable("should detect device conflict", func(
		instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
		setVmiDevices func(spec *virtv1.VirtualMachineInstanceSpec),
		expectedConflict string,
	) {
		setVmiDevices(&vmi.Spec)

		conflicts := vmiApplier.ApplyToVMI(field, instancetypeSpec, preferenceSpec, &vmi.Spec, &vmi.ObjectMeta)
		Expect(conflicts).To(HaveLen(1))
		Expect(conflicts[0].String()).To(Equal(expectedConflict))
	},
		Entry("when GPUs already exist in VMI",
			&v1beta1.VirtualMachineInstancetypeSpec{
				GPUs: []virtv1.GPU{{Name: "barfoo", DeviceName: "vendor.com/gpu_name"}},
			},
			func(spec *virtv1.VirtualMachineInstanceSpec) {
				spec.Domain.Devices.GPUs = []virtv1.GPU{{Name: "foobar", DeviceName: "vendor.com/gpu_name"}}
			},
			"spec.template.spec.domain.devices.gpus",
		),
		Entry("when HostDevices already exist in VMI",
			&v1beta1.VirtualMachineInstancetypeSpec{
				HostDevices: []virtv1.HostDevice{{Name: "foobar", DeviceName: "vendor.com/device_name"}},
			},
			func(spec *virtv1.VirtualMachineInstanceSpec) {
				spec.Domain.Devices.HostDevices = []virtv1.HostDevice{{Name: "barfoo", DeviceName: "vendor.com/device_name"}}
			},
			"spec.template.spec.domain.devices.hostDevices",
		),
	)
})
