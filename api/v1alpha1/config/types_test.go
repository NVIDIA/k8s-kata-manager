/*
 * Copyright (c), NVIDIA CORPORATION.  All rights reserved.
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
 */

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeConfig(t *testing.T) {
	testCases := []struct {
		description    string
		inputConfig    *Config
		expectedConfig *Config
	}{
		{
			description: "no runtime classes",
			inputConfig: &Config{
				ArtifactsDir: DefaultKataArtifactsDir,
			},
			expectedConfig: &Config{
				ArtifactsDir: DefaultKataArtifactsDir,
			},
		},
		{
			description: "multiple valid runtime class entries",
			inputConfig: &Config{
				ArtifactsDir: DefaultKataArtifactsDir,
				RuntimeClasses: []RuntimeClass{
					{
						Name: "nvidia-rc-1",
						Artifacts: Artifacts{
							URL: "/path/to/artifact-1:tag",
						},
					},
					{
						Name: "nvidia-rc-2",
						Artifacts: Artifacts{
							URL: "/path/to/artifact-2:tag",
						},
					},
				},
			},
			expectedConfig: &Config{
				ArtifactsDir: DefaultKataArtifactsDir,
				RuntimeClasses: []RuntimeClass{
					{
						Name: "nvidia-rc-1",
						Artifacts: Artifacts{
							URL: "/path/to/artifact-1:tag",
						},
					},
					{
						Name: "nvidia-rc-2",
						Artifacts: Artifacts{
							URL: "/path/to/artifact-2:tag",
						},
					},
				},
			},
		},
		{
			description: "invalid runtime class name sanitized",
			inputConfig: &Config{
				ArtifactsDir: DefaultKataArtifactsDir,
				RuntimeClasses: []RuntimeClass{
					{
						Name: "",
						Artifacts: Artifacts{
							URL: "/path/to/artifact-1:tag",
						},
					},
					{
						Name: "nvidia-rc-2",
						Artifacts: Artifacts{
							URL: "/path/to/artifact-2:tag",
						},
					},
				},
			},
			expectedConfig: &Config{
				ArtifactsDir: DefaultKataArtifactsDir,
				RuntimeClasses: []RuntimeClass{
					{
						Name: "nvidia-rc-2",
						Artifacts: Artifacts{
							URL: "/path/to/artifact-2:tag",
						},
					},
				},
			},
		},
		{
			description: "invalid artifact url sanitized",
			inputConfig: &Config{
				ArtifactsDir: DefaultKataArtifactsDir,
				RuntimeClasses: []RuntimeClass{
					{
						Name: "nvidia-rc-1",
						Artifacts: Artifacts{
							URL: "/path/to/artifact-1:tag",
						},
					},
					{
						Name: "nvidia-rc-2",
						Artifacts: Artifacts{
							URL: "",
						},
					},
				},
			},
			expectedConfig: &Config{
				ArtifactsDir: DefaultKataArtifactsDir,
				RuntimeClasses: []RuntimeClass{
					{
						Name: "nvidia-rc-1",
						Artifacts: Artifacts{
							URL: "/path/to/artifact-1:tag",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			SanitizeConfig(tc.inputConfig)
			require.Equal(t, tc.inputConfig, tc.expectedConfig)
		})
	}
}
