/**
# Copyright (c) 2023, NVIDIA CORPORATION.  All rights reserved.
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

package containerd

import (
	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConfig_AddRuntime(t *testing.T) {
	const (
		runtime        = "kata"
		runtimeType    = "io.containerd.kata.v2"
		kataConfigPath = "/opt/nvidia-gpu-operator/artifacts/runtimeclasses/kata-qemu-nvidia-gpu-snp/configuration-kata-qemu-nvidia-gpu-snp.toml"
	)

	testcases := []struct {
		runtimeName    string
		expectedConfig map[string]interface{}
	}{
		{
			runtimeName: runtime,
			expectedConfig: map[string]interface{}{
				"plugins": map[string]interface{}{
					"io.containerd.grpc.v1.cri": map[string]interface{}{
						"containerd": map[string]interface{}{
							"runtimes": map[string]interface{}{
								"kata": map[string]interface{}{
									"pod_annotations":                 []string{"io.katacontainers.*"},
									"privileged_without_host_devices": true,
									"runtime_type":                    runtimeType,
									"options": map[string]interface{}{
										"ConfigPath": kataConfigPath,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		config, err := toml.TreeFromMap(map[string]interface{}{
			"plugins": map[string]interface{}{
				"io.containerd.grpc.v1.cri": map[string]interface{}{
					"containerd": map[string]interface{}{
						"runtimes": map[string]interface{}{},
					},
				},
			},
		})
		require.NoError(t, err)
		ctrdConfig := Config{
			Tree:        config,
			RuntimeType: runtimeType,
			PodAnnotations: []string{
				"io.katacontainers.*",
			},
		}

		err = ctrdConfig.AddRuntime(tc.runtimeName, kataConfigPath, false)
		require.NoError(t, err)

		expected, err := toml.TreeFromMap(tc.expectedConfig)
		require.NoError(t, err)
		require.Equal(t, expected.String(), config.String())
	}
}
