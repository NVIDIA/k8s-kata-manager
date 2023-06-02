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
	"path/filepath"

	"github.com/pelletier/go-toml"
)

var (
	defaultArtifactKeys = []string{"kernel", "image", "initrd"}
)

type artifactsRootTransformer struct {
	targetRoot string
}

// NewArtifactsRootTransformer creates a new Transformer which updates
// the root path for kata artifacts specified in the kata configuration file
func NewArtifactsRootTransformer(targetRoot string) Transformer {
	return artifactsRootTransformer{
		targetRoot: targetRoot,
	}
}

// Transform transforms the kata config in-place by updating the root path
// of the kata artifacts (e.g. kernel, image, initrd)
func (t artifactsRootTransformer) Transform(config *toml.Tree) error {
	kConfig := kataConfig{}
	err := config.Unmarshal(&kConfig)
	if err != nil {
		return fmt.Errorf("failed to unmarshal kata config: %v", err)
	}

	for hypervisor := range kConfig.Hypervisor {
		for _, key := range defaultArtifactKeys {
			value := config.GetPath([]string{"hypervisor", hypervisor, key})
			if value == nil {
				continue
			}
			newPath := filepath.Join(t.targetRoot, filepath.Base(value.(string)))
			config.SetPath([]string{"hypervisor", hypervisor, key}, newPath)
		}
	}

	return nil
}
