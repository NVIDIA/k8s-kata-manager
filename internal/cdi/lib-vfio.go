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

	"github.com/NVIDIA/go-nvlib/pkg/nvlib/device"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi"
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi/spec"
	"k8s.io/klog/v2"
	"tags.cncf.io/container-device-interface/pkg/cdi"
	"tags.cncf.io/container-device-interface/specs-go"
)

type vfiolib nvcdilib

const (
	VfioPath        = "/dev/vfio"
	VfioDevicesPath = "/dev/vfio/devices"
	VfioDriver      = "vfio-pci"
)

var _ nvcdi.Interface = (*vfiolib)(nil)

// GetSpec returns the complete CDI spec
func (l *vfiolib) GetSpec(...string) (spec.Interface, error) {
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
		return nil, fmt.Errorf("failed getting NVIDIA GPUs: %w", err)
	}

	path := VfioPath
	devName := ""

	for idx, dev := range devices {
		if dev.Driver == VfioDriver {
			klog.Infof("Found NVIDIA device: address=%s, driver=%s, iommu_group=%d, deviceId=%x",
				dev.Address, dev.Driver, dev.IommuGroup, dev.Device)

			if dev.IommuFD != "" {
				path = VfioDevicesPath
				devName = dev.IommuFD
			} else {
				devName = fmt.Sprintf("%d", dev.IommuGroup)
			}
			cedits := specs.ContainerEdits{
				DeviceNodes: []*specs.DeviceNode{
					{
						Path: fmt.Sprintf("%s/%s", path, devName),
					},
				},
			}
			// Add the same device multiple times with keys for meant for
			// various use cases:
			// key=idx: use case where cdi annotations are manually put
			//   on pod spec e.g. 0,1,2 etc
			// key=IommuGroup e.g. 65 for /dev/vfio/65 in non-iommufd setup
			//   and legacy device plugin case
			// key=IommuFD e.g. vfio0 for /dev/vfio/devices/vfio0 for
			//   iommufd support
			deviceSpecs = append(deviceSpecs, specs.Device{
				Name:           fmt.Sprintf("%d", idx),
				ContainerEdits: cedits,
			})
			deviceSpecs = append(deviceSpecs, specs.Device{
				Name:           fmt.Sprintf("%d", dev.IommuGroup),
				ContainerEdits: cedits,
			})
			if dev.IommuFD != "" {
				deviceSpecs = append(deviceSpecs, specs.Device{
					Name:           fmt.Sprintf("%s", dev.IommuFD),
					ContainerEdits: cedits,
				})
			}
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
	return nil, fmt.Errorf("unexpected call to vfiolib.GetGPUDeviceEdits()")
}

// GetGPUDeviceSpecs should not be called for vfiolib
func (l *vfiolib) GetGPUDeviceSpecs(int, device.Device) ([]specs.Device, error) {
	return nil, fmt.Errorf("unexpected call to vfiolib.GetGPUDeviceSpecs()")
}

// GetMIGDeviceEdits should not be called for vfiolib
func (l *vfiolib) GetMIGDeviceEdits(device.Device, device.MigDevice) (*cdi.ContainerEdits, error) {
	return nil, fmt.Errorf("unexpected call to vfiolib.GetMIGDeviceEdits()")
}

// GetMIGDeviceSpecs should not be called for vfiolib
func (l *vfiolib) GetMIGDeviceSpecs(int, device.Device, int, device.MigDevice) ([]specs.Device, error) {
	return nil, fmt.Errorf("unexpected call to vfiolib.GetMIGDeviceSpecs()")
}

// GetDeviceSpecsByID should not be called for vfiolib.
func (l *vfiolib) GetDeviceSpecsByID(...string) ([]specs.Device, error) {
	return nil, fmt.Errorf("unexpected call to vfiolib.GetDeviceSpecsByID()")
}
