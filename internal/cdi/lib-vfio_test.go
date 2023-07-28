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
	"testing"

	"github.com/container-orchestrated-devices/container-device-interface/specs-go"
	"github.com/stretchr/testify/require"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvpci"
)

func TestGetAllDeviceSpecs(t *testing.T) {
	testCases := []struct {
		description         string
		lib                 *vfiolib
		expectedDeviceSpecs []specs.Device
	}{
		{
			description: "no NVIDIA devices",
			lib: &vfiolib{
				nvpcilib: &nvpciInterfaceMock{
					GetGPUsFunc: func() ([]*nvpci.NvidiaPCIDevice, error) {
						devices := []*nvpci.NvidiaPCIDevice{}
						return devices, nil
					},
				},
			},
			expectedDeviceSpecs: nil,
		},
		{
			description: "one NVIDIA device, not bound to vfio-pci",
			lib: &vfiolib{
				nvpcilib: &nvpciInterfaceMock{
					GetGPUsFunc: func() ([]*nvpci.NvidiaPCIDevice, error) {
						devices := []*nvpci.NvidiaPCIDevice{
							{
								Address:    "000:3B:00.0",
								Device:     0x2331,
								IommuGroup: -1,
								Driver:     "nvidia",
							},
						}
						return devices, nil
					},
				},
			},
			expectedDeviceSpecs: nil,
		},
		{
			description: "one NVIDIA device, bound to vfio-pci",
			lib: &vfiolib{
				nvpcilib: &nvpciInterfaceMock{
					GetGPUsFunc: func() ([]*nvpci.NvidiaPCIDevice, error) {
						devices := []*nvpci.NvidiaPCIDevice{
							{
								Address:    "000:3B:00.0",
								Device:     0x2331,
								IommuGroup: 60,
								Driver:     "vfio-pci",
							},
						}
						return devices, nil
					},
				},
			},
			expectedDeviceSpecs: []specs.Device{
				{
					Name: "0",
					ContainerEdits: specs.ContainerEdits{
						DeviceNodes: []*specs.DeviceNode{
							{
								Path: "/dev/vfio/60",
							},
						},
					},
				},
			},
		},
		{
			description: "multiple NVIDIA devices, one bound to vfio-pci",
			lib: &vfiolib{
				nvpcilib: &nvpciInterfaceMock{
					GetGPUsFunc: func() ([]*nvpci.NvidiaPCIDevice, error) {
						devices := []*nvpci.NvidiaPCIDevice{
							{
								Address:    "000:3B:00.0",
								Device:     0x2331,
								IommuGroup: 60,
								Driver:     "vfio-pci",
							},
							{
								Address:    "000:86:00.0",
								Device:     0x2331,
								IommuGroup: -1,
								Driver:     "",
							},
						}
						return devices, nil
					},
				},
			},
			expectedDeviceSpecs: []specs.Device{
				{
					Name: "0",
					ContainerEdits: specs.ContainerEdits{
						DeviceNodes: []*specs.DeviceNode{
							{
								Path: "/dev/vfio/60",
							},
						},
					},
				},
			},
		},
		{
			description: "multiple NVIDIA devices, all bound to vfio-pci",
			lib: &vfiolib{
				nvpcilib: &nvpciInterfaceMock{
					GetGPUsFunc: func() ([]*nvpci.NvidiaPCIDevice, error) {
						devices := []*nvpci.NvidiaPCIDevice{
							{
								Address:    "000:3B:00.0",
								Device:     0x2331,
								IommuGroup: 60,
								Driver:     "vfio-pci",
							},
							{
								Address:    "000:86:00.0",
								Device:     0x2331,
								IommuGroup: 90,
								Driver:     "vfio-pci",
							},
						}
						return devices, nil
					},
				},
			},
			expectedDeviceSpecs: []specs.Device{
				{
					Name: "0",
					ContainerEdits: specs.ContainerEdits{
						DeviceNodes: []*specs.DeviceNode{
							{
								Path: "/dev/vfio/60",
							},
						},
					},
				},
				{
					Name: "1",
					ContainerEdits: specs.ContainerEdits{
						DeviceNodes: []*specs.DeviceNode{
							{
								Path: "/dev/vfio/90",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			deviceSpecs, err := tc.lib.GetAllDeviceSpecs()
			require.NoError(t, err)
			require.Equal(t, tc.expectedDeviceSpecs, deviceSpecs)
		})
	}
}
