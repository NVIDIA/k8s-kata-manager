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
	"os"
	"os/exec"

	"github.com/pelletier/go-toml"
	"k8s.io/klog/v2"

	api "github.com/NVIDIA/k8s-kata-manager/api/v1alpha1/config"

	"github.com/NVIDIA/k8s-kata-manager/internal/runtime"
)

// Config represents the crio config
type Config struct {
	*toml.Tree
	RuntimeType           string
	UseDefaultRuntimeName bool
	PodAnnotations        []string
	Path                  string
}

func Setup(o *runtime.Options) (runtime.Runtime, error) {
	crioConfig, err := New(
		WithPath(o.Path),
		WithPodAnnotations(o.PodAnnotations...),
		WithRuntimeType(o.RuntimeType),
	)
	return crioConfig, err
}

// New creates a crio config with the specified options
func New(opts ...Option) (*Config, error) {
	b := &builder{}
	for _, opt := range opts {
		opt(b)
	}

	return b.build()
}

// AddRuntime adds a runtime to the crio config
func (c *Config) AddRuntime(runtimeName string, path string, setAsDefault bool) error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	config := *c.Tree

	// By default we extract the runtime options from the crun settings; if this does not exist we get the options from the default runtime specified in the config.
	runtimeNamesForConfig := []string{api.DefaultCrioRuntime}
	if defaultRuntimeName, ok := config.GetPath([]string{"crio", "runtime", "default_runtime"}).(string); ok && defaultRuntimeName != "" {
		runtimeNamesForConfig = append(runtimeNamesForConfig, defaultRuntimeName)
	}
	for _, r := range runtimeNamesForConfig {
		if options, ok := config.GetPath([]string{"crio", "runtime", "runtimes", r}).(*toml.Tree); ok {
			options, _ = toml.Load(options.String())
			config.SetPath([]string{"crio", "runtime", "runtimes", runtimeName}, options)
			break
		}
	}

	config.SetPath([]string{"crio", "runtime", "runtimes", runtimeName, "runtime_path"}, path)
	config.SetPath([]string{"crio", "runtime", "runtimes", runtimeName, "runtime_type"}, "vm")
	config.SetPath([]string{"crio", "runtime", "runtimes", runtimeName, "privileged_without_host_devices"}, "true")

	if setAsDefault {
		config.SetPath([]string{"crio", "runtime", "default_runtime"}, runtimeName)
	}

	*c.Tree = config
	return nil
}

// DefaultRuntime returns the default runtime for the crio config
func (c *Config) DefaultRuntime() string {
	if c == nil || c.Tree == nil {
		return ""
	}
	if runtime, ok := c.GetPath([]string{"crio", "runtime", "default_runtime"}).(string); ok {
		return runtime
	}
	return ""
}

// RemoveRuntime removes a runtime from the crio config
func (c *Config) RemoveRuntime(name string) error {
	if c == nil {
		return nil
	}

	config := *c.Tree
	if runtime, ok := config.GetPath([]string{"crio", "runtime", "default_runtime"}).(string); ok {
		if runtime == name {
			err := config.DeletePath([]string{"crio", "runtime", "default_runtime"})
			if err != nil {
				return err
			}

		}
	}

	runtimeClassPath := []string{"crio", "runtime", "runtimes", name}
	err := config.DeletePath(runtimeClassPath)
	if err != nil {
		return err
	}
	for i := 0; i < len(runtimeClassPath); i++ {
		remainingPath := runtimeClassPath[:len(runtimeClassPath)-i]
		if entry, ok := config.GetPath(remainingPath).(*toml.Tree); ok {
			if len(entry.Keys()) != 0 {
				break
			}
			err := config.DeletePath(remainingPath)
			if err != nil {
				return err
			}
		}
	}

	*c.Tree = config
	return nil
}

// Save writes the config to the specified path
func (c *Config) Save() (int64, error) {
	config := c.Tree
	output, err := config.Marshal()
	if err != nil {
		return 0, fmt.Errorf("unable to convert to TOML: %w", err)
	}
	f, err := os.Create(c.Path)
	if err != nil {
		return 0, fmt.Errorf("unable to open '%s' for writing: %w", c.Path, err)
	}
	defer f.Close()

	n, err := f.Write(output)
	if err != nil {
		return 0, fmt.Errorf("unable to write output: %w", err)
	}

	return int64(n), err
}

func (c *Config) Restart() error {
	var args []string
	args = append(args, "chroot", "/host", "systemctl", "restart", "crio")

	klog.Infof("Restarting crio using systemd")

	// TODO: Can we harden this so that there is less risk of command injection
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error restarting crio using systemd: %w", err)
	}

	return nil
}
