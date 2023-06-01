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

import "k8s.io/klog/v2"

// Config defines the configuration for the kata-manager
type Config struct {
	ArtifactsDir   string         `json:"artifactsDir,omitempty"    yaml:"artifactsDir,omitempty"`
	RuntimeClasses []RuntimeClass `json:"runtimeClasses,omitempty"  yaml:"runtimeClasses,omitempty"`
}

// RuntimeClass defines the configuration for a kata RuntimeClass
type RuntimeClass struct {
	Name         string            `json:"name"                   yaml:"name"`
	NodeSelector map[string]string `json:"nodeSelector,omitempty" yaml:"nodeSelector,omitempty"`
	Artifacts    Artifacts         `json:"artifacts"              yaml:"artifacts"`
}

// Artifacts defines the path to an OCI artifact (payload) containing all artifacts
// associated with a kata RuntimeClass (e.g. kernel, guest image, initrd, kata configuration)
type Artifacts struct {
	URL        string `json:"url"                  yaml:"url"`
	PullSecret string `json:"pullSecret,omitempty" yaml:"pullSecret,omitempty"`
}

// NewDefaultConfig returns a new default config.
func NewDefaultConfig() *Config {
	return &Config{
		ArtifactsDir: DefaultKataArtifactsDir,
	}
}

// SanitizeConfig sanitizes the config struct and removes any invalid runtime class entries
func SanitizeConfig(c *Config) {
	i := 0
	for idx, rc := range c.RuntimeClasses {
		if rc.Name == "" {
			klog.Warningf("empty RuntimeClass name, skipping entry at index %d", idx)
			continue
		}
		if rc.Artifacts.URL == "" {
			klog.Warningf("empty artifacts url for runtime class %s, skipping entry at index %d", rc.Name, idx)
			continue
		}
		c.RuntimeClasses[i] = rc
		i++
	}

	c.RuntimeClasses = c.RuntimeClasses[:i]
}
