/**
# Copyright (c) 2024, NVIDIA CORPORATION.  All rights reserved.
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

package crio

import (
	"fmt"
	"os/exec"

	"github.com/pelletier/go-toml"
	"k8s.io/klog/v2"
)

const (
	defaultRuntimeType = "oci"
)

type builder struct {
	path            string
	runtimeType     string
	useLegacyConfig bool
	podAnnotations  []string
}

// Option defines a function that can be used to configure the config builder
type Option func(*builder)

// WithPath sets the path for the config builder
func WithPath(path string) Option {
	return func(b *builder) {
		b.path = path
	}
}

// WithRuntimeType sets the runtime type for the config builder
func WithRuntimeType(runtimeType string) Option {
	return func(b *builder) {
		b.runtimeType = runtimeType
	}
}

// WithUseLegacyConfig sets the useLegacyConfig flag for the config builder
func WithUseLegacyConfig(useLegacyConfig bool) Option {
	return func(b *builder) {
		b.useLegacyConfig = useLegacyConfig
	}
}

// WithPodAnnotations sets the container annotations for the config builder
func WithPodAnnotations(podAnnotations ...string) Option {
	return func(b *builder) {
		b.podAnnotations = podAnnotations
	}
}

func (b *builder) build() (*Config, error) {
	if b.path == "" {
		return &Config{}, fmt.Errorf("config path is empty")
	}

	if b.runtimeType == "" {
		b.runtimeType = defaultRuntimeType
	}

	config, err := loadConfig(b.path)
	if err != nil {
		return &Config{}, fmt.Errorf("failed to load config: %w", err)
	}
	config.RuntimeType = b.runtimeType
	config.UseDefaultRuntimeName = !b.useLegacyConfig
	config.PodAnnotations = b.podAnnotations
	config.Path = b.path

	return config, nil
}

// loadConfig loads the crio config from disk
func loadConfig(config string) (*Config, error) {
	klog.Infof("Loading config: %v", config)

	var args []string
	args = append(args, "chroot", "/host", "crio", "status", "config")

	klog.Infof("Getting crio config")

	// TODO: Can we harden this so that there is less risk of command injection
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error getting crio config: %w", err)
	}
	tomlConfig, err := toml.LoadBytes(output)
	if err != nil {
		return nil, err
	}

	klog.Infof("Successfully loaded config")

	cfg := Config{
		Tree: tomlConfig,
	}
	return &cfg, nil
}
