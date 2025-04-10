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
	"os/signal"
	"syscall"
	"time"

	"github.com/pelletier/go-toml"
	"k8s.io/klog/v2"

	"github.com/NVIDIA/k8s-kata-manager/internal/runtime"
)

// Config represents the containerd config
type Config struct {
	*toml.Tree
	RuntimeType           string
	UseDefaultRuntimeName bool
	PodAnnotations        []string
	Path                  string
	Socket                string
}

func Setup(o *runtime.Options) (runtime.Runtime, error) {
	ctrdConfig, err := New(
		WithPath(o.Path),
		WithPodAnnotations(o.PodAnnotations...),
		WithRuntimeType(o.RuntimeType),
		WithSocket(o.Socket),
	)
	return ctrdConfig, err
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

	cfgPath := config.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name})
	if kata, ok := cfgPath.(*toml.Tree); ok {
		kata, err := toml.Load(kata.String())
		if err != nil {
			return fmt.Errorf("failed to load kata config: %w", err)
		}
		config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name}, kata)
	}

	if config.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name}) == nil {
		config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name, "runtime_type"}, c.RuntimeType)
		config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name, "privileged_without_host_devices"}, true)
	}

	config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name, "pod_annotations"}, c.PodAnnotations)

	config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "runtimes", name, "options", "ConfigPath"}, path)

	if setAsDefault {
		config.SetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"}, name)
	}

	*c.Tree = config
	return nil
}

// DefaultRuntime returns the default runtime for the containerd config
func (c *Config) DefaultRuntime() string {
	if runtime, ok := c.GetPath([]string{"plugins", "io.containerd.grpc.v1.cri", "containerd", "default_runtime_name"}).(string); ok {
		return runtime
	}
	return ""
}

// RemoveRuntime removes a runtime from the containerd config
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
func (c *Config) Save() (int64, error) {
	config := c.Tree
	output, err := config.ToTomlString()
	if err != nil {
		return 0, fmt.Errorf("unable to convert to TOML: %w", err)
	}

	if c.Path == "" {
		fmt.Fprintf(os.Stdout, "%s\n", output)
		return int64(len(output)), nil
	}

	if len(output) == 0 {
		err := os.Remove(c.Path)
		if err != nil {
			return 0, fmt.Errorf("unable to remove empty file: %w", err)
		}
		return 0, nil
	}

	f, err := os.Create(c.Path)
	if err != nil {
		return 0, fmt.Errorf("unable to open '%s' for writing: %w", c.Path, err)
	}
	defer f.Close()

	n, err := f.WriteString(output)
	if err != nil {
		return 0, fmt.Errorf("unable to write output: %w", err)
	}

	return int64(n), err
}

func (c *Config) Restart() error {

	// Create a channel to receive signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGHUP)

	// Set up a timer to ignore the signal for 5 seconds
	ignoreTimer := time.NewTimer(5 * time.Second)

	// Create a channel to signal when the function has finished executing
	done := make(chan error)

	// Start the function in a goroutine
	go func() {
		// Execute your function here
		err := RestartContainerd(c.Socket)
		if err != nil {
			klog.Errorf("error restarting containerd: %v", err)
			done <- err
		}
		// Since we are restarting containerd we need to
		// Ignore the SIGTERM signal for 5 seconds
		<-ignoreTimer.C
		// Signal that the function has finished executing
		done <- nil
	}()

	// Wait for the function to finish executing or for the signal to be received
	select {
	case err := <-done:
		if err != nil {
			return err
		}
	case s := <-sigs:
		fmt.Printf("Received signal %v", s)
		// Reset the timer to ignore the signal for another 5 seconds
		ignoreTimer.Reset(5 * time.Second)
	}

	return nil
}
