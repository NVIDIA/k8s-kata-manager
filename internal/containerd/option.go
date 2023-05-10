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

package containerd

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
	log "github.com/sirupsen/logrus"
)

const (
	defaultRuntimeType = "io.containerd.runc.v2"
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
		return &Config{}, fmt.Errorf("failed to load config: %v", err)
	}
	config.RuntimeType = b.runtimeType
	config.UseDefaultRuntimeName = !b.useLegacyConfig
	config.PodAnnotations = b.podAnnotations

	return config, nil
}

// loadConfig loads the containerd config from disk
func loadConfig(config string) (*Config, error) {
	log.Infof("Loading config: %v", config)

	info, err := os.Stat(config)
	if os.IsExist(err) && info.IsDir() {
		return nil, fmt.Errorf("config file is a directory")
	}

	configFile := config
	if os.IsNotExist(err) {
		configFile = "/dev/null"
		log.Infof("Config file does not exist, creating new one")
	}

	tomlConfig, err := toml.LoadFile(configFile)
	if err != nil {
		return nil, err
	}

	log.Infof("Successfully loaded config")

	cfg := Config{
		Tree: tomlConfig,
	}
	return &cfg, nil
}
