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
	"testing"

	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/require"
)

func TestConfig_AddRuntime(t *testing.T) {
	const (
		runtime        = "kata"
		runtimeType    = "io.containerd.kata.v2"
		kataConfigPath = "/opt/nvidia-gpu-operator/artifacts/runtimeclasses/kata-qemu-nvidia-gpu-snp/configuration-kata-qemu-nvidia-gpu-snp.toml"
	)

	testcases := []struct {
		name           string
		version        int64
		runtimeName    string
		initialConfig  map[string]interface{}
		expectedConfig map[string]interface{}
	}{
		{
			name:        "version 2 config",
			version:     2,
			runtimeName: runtime,
			initialConfig: map[string]interface{}{
				"version": int64(2),
				"plugins": map[string]interface{}{
					"io.containerd.grpc.v1.cri": map[string]interface{}{
						"containerd": map[string]interface{}{
							"runtimes": map[string]interface{}{},
						},
					},
				},
			},
			expectedConfig: map[string]interface{}{
				"version": int64(2),
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
		{
			name:        "version 3 config",
			version:     3,
			runtimeName: runtime,
			initialConfig: map[string]interface{}{
				"version": int64(3),
				"plugins": map[string]interface{}{
					"io.containerd.cri.v1.runtime": map[string]interface{}{
						"containerd": map[string]interface{}{
							"runtimes": map[string]interface{}{},
						},
					},
				},
			},
			expectedConfig: map[string]interface{}{
				"version": int64(3),
				"plugins": map[string]interface{}{
					"io.containerd.cri.v1.runtime": map[string]interface{}{
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
		{
			name:          "empty config defaults to v2",
			version:       2,
			runtimeName:   runtime,
			initialConfig: map[string]interface{}{
				// No version field, no existing structure - should default to v2
			},
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
		t.Run(tc.name, func(t *testing.T) {
			config, err := toml.TreeFromMap(tc.initialConfig)
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
		})
	}
}

func TestConfig_DefaultRuntime(t *testing.T) {
	const (
		runtime     = "kata-qemu"
		runtimeType = "io.containerd.kata.v2"
	)

	testcases := []struct {
		name           string
		config         map[string]interface{}
		expectedResult string
	}{
		{
			name: "version 2 with default runtime",
			config: map[string]interface{}{
				"version": int64(2),
				"plugins": map[string]interface{}{
					"io.containerd.grpc.v1.cri": map[string]interface{}{
						"containerd": map[string]interface{}{
							"default_runtime_name": runtime,
						},
					},
				},
			},
			expectedResult: runtime,
		},
		{
			name: "version 3 with default runtime",
			config: map[string]interface{}{
				"version": int64(3),
				"plugins": map[string]interface{}{
					"io.containerd.cri.v1.runtime": map[string]interface{}{
						"containerd": map[string]interface{}{
							"default_runtime_name": runtime,
						},
					},
				},
			},
			expectedResult: runtime,
		},
		{
			name: "version 2 without default runtime",
			config: map[string]interface{}{
				"version": int64(2),
				"plugins": map[string]interface{}{
					"io.containerd.grpc.v1.cri": map[string]interface{}{
						"containerd": map[string]interface{}{},
					},
				},
			},
			expectedResult: "",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := toml.TreeFromMap(tc.config)
			require.NoError(t, err)
			ctrdConfig := Config{
				Tree:        config,
				RuntimeType: runtimeType,
			}

			result := ctrdConfig.DefaultRuntime()
			require.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestConfig_RemoveRuntime(t *testing.T) {
	const (
		runtime        = "kata"
		runtimeType    = "io.containerd.kata.v2"
		kataConfigPath = "/opt/nvidia-gpu-operator/artifacts/runtimeclasses/kata-qemu-nvidia-gpu-snp/configuration-kata-qemu-nvidia-gpu-snp.toml"
	)

	testcases := []struct {
		name            string
		initialConfig   map[string]interface{}
		runtimeToRemove string
		expectedConfig  map[string]interface{}
	}{
		{
			name: "version 2 remove runtime",
			initialConfig: map[string]interface{}{
				"version": int64(2),
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
			runtimeToRemove: runtime,
			// When only version is left, it gets removed too
			expectedConfig: map[string]interface{}{},
		},
		{
			name: "version 3 remove runtime",
			initialConfig: map[string]interface{}{
				"version": int64(3),
				"plugins": map[string]interface{}{
					"io.containerd.cri.v1.runtime": map[string]interface{}{
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
			runtimeToRemove: runtime,
			// When only version is left, it gets removed too
			expectedConfig: map[string]interface{}{},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := toml.TreeFromMap(tc.initialConfig)
			require.NoError(t, err)
			ctrdConfig := Config{
				Tree:        config,
				RuntimeType: runtimeType,
			}

			err = ctrdConfig.RemoveRuntime(tc.runtimeToRemove)
			require.NoError(t, err)

			expected, err := toml.TreeFromMap(tc.expectedConfig)
			require.NoError(t, err)
			require.Equal(t, expected.String(), config.String())
		})
	}
}
