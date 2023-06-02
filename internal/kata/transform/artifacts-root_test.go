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

package transform

import (
	"fmt"
	"testing"

	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/require"
)

func TestArtifactsRootTransform(t *testing.T) {
	testCases := []struct {
		targetRoot     string
		inputConfig    map[string]interface{}
		expectedConfig map[string]interface{}
	}{
		{
			targetRoot: "/new/artifacts/root/",
			inputConfig: map[string]interface{}{
				"hypervisor": map[string]interface{}{
					"qemu": map[string]interface{}{
						"path":   "/opt/kata/bin/qemu-system-x86_64",
						"kernel": "/opt/kata/share/vmlinuz.container",
						"image":  "/opt/kata/share/kata-vm.image",
					},
				},
				"runtime": map[string]interface{}{
					"enable_debug": true,
				},
			},
			expectedConfig: map[string]interface{}{
				"hypervisor": map[string]interface{}{
					"qemu": map[string]interface{}{
						"path":   "/opt/kata/bin/qemu-system-x86_64",
						"kernel": "/new/artifacts/root/vmlinuz.container",
						"image":  "/new/artifacts/root/kata-vm.image",
					},
				},
				"runtime": map[string]interface{}{
					"enable_debug": true,
				},
			},
		},
		{
			targetRoot: "/new/artifacts/root/",
			inputConfig: map[string]interface{}{
				"hypervisor": map[string]interface{}{
					"qemu": map[string]interface{}{
						"path":   "/opt/kata/bin/qemu-system-x86_64",
						"kernel": "/opt/kata/share/vmlinuz.container",
						"image":  "/opt/kata/share/kata-vm.image",
						"initrd": "/opt/kata/share/initrd",
					},
				},
				"runtime": map[string]interface{}{
					"enable_debug": true,
				},
			},
			expectedConfig: map[string]interface{}{
				"hypervisor": map[string]interface{}{
					"qemu": map[string]interface{}{
						"path":   "/opt/kata/bin/qemu-system-x86_64",
						"kernel": "/new/artifacts/root/vmlinuz.container",
						"image":  "/new/artifacts/root/kata-vm.image",
						"initrd": "/new/artifacts/root/initrd",
					},
				},
				"runtime": map[string]interface{}{
					"enable_debug": true,
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			input, err := toml.TreeFromMap(tc.inputConfig)
			require.NoError(t, err)

			transform := NewArtifactsRootTransformer(tc.targetRoot)
			err = transform.Transform(input)
			require.NoError(t, err)

			expected, err := toml.TreeFromMap(tc.expectedConfig)
			require.NoError(t, err)

			require.Equal(t, expected.String(), input.String())
		})
	}
}
