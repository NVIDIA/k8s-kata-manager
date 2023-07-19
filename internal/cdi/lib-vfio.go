/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
**/

package cdi

import (
	"fmt"

	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/spec"
	"github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	"github.com/container-orchestrated-devices/container-device-interface/specs-go"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvlib/device"
	"k8s.io/klog/v2"
)

type vfiolib nvcdilib

var _ nvcdi.Interface = (*vfiolib)(nil)

// GetSpec returns the complete CDI spec
func (l *vfiolib) GetSpec() (spec.Interface, error) {
	deviceSpecs, err := l.GetAllDeviceSpecs()
	if err != nil {
		return nil, err
	}

	edits, err := l.GetCommonEdits()
	if err != nil {
		return nil, err
	}

	return spec.New(
		spec.WithDeviceSpecs(deviceSpecs),
		spec.WithEdits(*edits.ContainerEdits),
		spec.WithVendor(l.vendor),
		spec.WithClass(l.class),
	)
}

// GetAllDeviceSpecs returns the device specs for all available devices.
func (l *vfiolib) GetAllDeviceSpecs() ([]specs.Device, error) {
	var deviceSpecs []specs.Device

	devices, err := l.nvpcilib.GetGPUs()
	if err != nil {
		return nil, fmt.Errorf("failed getting NVIDIA GPUs: %v", err)
	}

	for idx, dev := range devices {
		if dev.Driver == "vfio-pci" {
			klog.Infof("Found NVIDIA device: address=%s, driver=%s, iommu_group=%d, deviceId=%x",
				dev.Address, dev.Driver, dev.IommuGroup, dev.Device)
			deviceSpecs = append(deviceSpecs, specs.Device{
				Name: fmt.Sprintf("%d", idx),
				ContainerEdits: specs.ContainerEdits{
					DeviceNodes: []*specs.DeviceNode{
						{
							Path: fmt.Sprintf("/dev/vfio/%d", dev.IommuGroup),
						},
					},
				},
			})
		}
	}

	return deviceSpecs, nil
}

// GetCommonEdits returns common edits for ALL devices.
// Note, currently there are no common edits.
func (l *vfiolib) GetCommonEdits() (*cdi.ContainerEdits, error) {
	return &cdi.ContainerEdits{ContainerEdits: &specs.ContainerEdits{}}, nil
}

// GetGPUDeviceEdits should not be called for vfiolib
func (l *vfiolib) GetGPUDeviceEdits(device.Device) (*cdi.ContainerEdits, error) {
	return nil, fmt.Errorf("Unexpected call to vfiolib.GetGPUDeviceEdits()")
}

// GetGPUDeviceSpecs should not be called for vfiolib
func (l *vfiolib) GetGPUDeviceSpecs(int, device.Device) (*specs.Device, error) {
	return nil, fmt.Errorf("Unexpected call to vfiolib.GetGPUDeviceSpecs()")
}

// GetMIGDeviceEdits should not be called for vfiolib
func (l *vfiolib) GetMIGDeviceEdits(device.Device, device.MigDevice) (*cdi.ContainerEdits, error) {
	return nil, fmt.Errorf("Unexpected call to vfiolib.GetMIGDeviceEdits()")
}

// GetMIGDeviceSpecs should not be called for vfiolib
func (l *vfiolib) GetMIGDeviceSpecs(int, device.Device, int, device.MigDevice) (*specs.Device, error) {
	return nil, fmt.Errorf("Unexpected call to vfiolib.GetMIGDeviceSpecs()")
}
