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
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
)

// Config represents the containerd config
type Config struct {
	*toml.Tree
	RuntimeType           string
	UseDefaultRuntimeName bool
	PodAnnotations        []string
}

// New creates a containerd config with the specified options
func New(opts ...Option) (*Config, error) {
	b := &builder{}
	for _, opt := range opts {
		opt(b)
	}

	return b.build()
}

// AddRuntime adds a runtime to the containerd config
func (c *Config) AddRuntime(name string, path string, setAsDefault bool) error {
	if c == nil || c.Tree == nil {
		return fmt.Errorf("config is nil")
	}
	config := *c.Tree

	switch kata := config.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", "kata-qemu-nvidia-gpu"}).(type) {
	case *toml.Tree:
		kata, _ = toml.Load(kata.String())
		config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name}, kata)
	}

	if config.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name}) == nil {
		config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name, "runtime_type"}, c.RuntimeType)
		config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name, "privileged_without_host_devices"}, true)
	}
	annotations, err := c.getRuntimeAnnotations([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name, "pod_annotations"})
	if err != nil {
		return err
	}
	annotations = append(c.PodAnnotations, annotations...)
	config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name, "pod_annotations"}, annotations)

	config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name, "options", "ConfigPath"}, path)

	if setAsDefault {
		config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"}, name)
	}

	*c.Tree = config
	return nil
}

func (c *Config) getRuntimeAnnotations(path []string) ([]string, error) {
	if c == nil || c.Tree == nil {
		return nil, nil
	}

	config := *c.Tree
	if !config.HasPath(path) {
		return nil, nil
	}
	annotationsI, ok := config.GetPath(path).([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid annotations: %v", annotationsI)
	}

	var annotations []string
	for _, annotation := range annotationsI {
		a, ok := annotation.(string)
		if !ok {
			return nil, fmt.Errorf("invalid annotation: %v", annotation)
		}
		annotations = append(annotations, a)
	}

	return annotations, nil
}

// DefaultRuntime returns the default runtime for the cri-o config
func (c Config) DefaultRuntime() string {
	if runtime, ok := c.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"}).(string); ok {
		return runtime
	}
	return ""
}

// RemoveRuntime removes a runtime from the docker config
func (c *Config) RemoveRuntime(name string) error {
	if c == nil || c.Tree == nil {
		return nil
	}

	config := *c.Tree

	if err := config.DeletePath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name}); err != nil {
		return err
	}
	if runtime, ok := config.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"}).(string); ok {
		if runtime == name {
			if err := config.DeletePath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"}); err != nil {
				return err
			}
		}
	}

	runtimePath := []string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name}
	for i := 0; i < len(runtimePath); i++ {
		if runtimes, ok := config.GetPath(runtimePath[:len(runtimePath)-i]).(*toml.Tree); ok {
			if len(runtimes.Keys()) == 0 {
				if err := config.DeletePath(runtimePath[:len(runtimePath)-i]); err != nil {
					return err
				}

			}
		}
	}

	if len(config.Keys()) == 1 && config.Keys()[0] == "version" {
		if err := config.Delete("version"); err != nil {
			return err
		}
	}

	*c.Tree = config
	return nil
}

// Save writes the config to the specified path
func (c Config) Save(path string) (int64, error) {
	config := c.Tree
	output, err := config.ToTomlString()
	if err != nil {
		return 0, fmt.Errorf("unable to convert to TOML: %v", err)
	}

	if path == "" {
		os.Stdout.WriteString(fmt.Sprintf("%s\n", output))
		return int64(len(output)), nil
	}

	if len(output) == 0 {
		err := os.Remove(path)
		if err != nil {
			return 0, fmt.Errorf("unable to remove empty file: %v", err)
		}
		return 0, nil
	}

	f, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("unable to open '%v' for writing: %v", path, err)
	}
	defer f.Close()

	n, err := f.WriteString(output)
	if err != nil {
		return 0, fmt.Errorf("unable to write output: %v", err)
	}

	return int64(n), err
}
