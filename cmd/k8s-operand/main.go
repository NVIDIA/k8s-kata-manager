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

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	k8scli "github.com/NVIDIA/k8s-kata-manager/internal/client-go"

	cli "github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"

	"k8s.io/client-go/pkg/version"
	klog "k8s.io/klog/v2"
	yaml "sigs.k8s.io/yaml"
)

// Worker is the interface for k8s-kata-manager daemon
type Worker interface {
	Run() error
	Stop()
}

type worker struct {
	kubernetesNamespace string
	stop                chan struct{} // channel for signaling stop

	Config         *Config
	Namespace      string
	ConfigFilePath string
}

type Config struct {
	ArtifactsDir string `yaml:"artifactsDir"`
	RuntimeClass []struct {
		Name         string            `yaml:"name"`
		NodeSelector map[string]string `yaml:"nodeSelector"`
		Artifacts    struct {
			URL        string `yaml:"url"`
			PullSecret string `yaml:"pullSecret"`
		} `yaml:"artifacts"`
	} `yaml:"runtimeClass"`
}

func newDefaultConfig() *Config {
	return &Config{
		ArtifactsDir: "/opt/nvidia-gpu-operator/artifacts/runtimeclasses",
	}
}

// newWorker returns a new worker struct
func newWorker() *worker {
	return &worker{}
}

func main() {
	worker := newWorker()

	// Create the top-level CLI
	c := cli.NewApp()
	c.Name = "k8s-kata-manager"
	c.Usage = "Tool for managing and NVIDIA OCI artifacts"
	c.Version = "0.1.0"
	c.Action = func(ctx *cli.Context) error {
		return worker.Run(ctx)
	}

	// Setup the flags for this command
	c.Flags = []cli.Flag{
		altsrc.NewIntFlag(
			&cli.IntFlag{
				Name:    "loglevel",
				Usage:   "Set the logging level",
				Aliases: []string{"l"},
				Value:   1}),
		&cli.StringFlag{
			Name:        "configFile",
			Aliases:     []string{"c"},
			Usage:       "Path to the configuration file",
			Destination: &worker.ConfigFilePath,
			EnvVars:     []string{"CONFIG_FILE"},
		},
		&cli.StringFlag{
			Name:        "namespace",
			Aliases:     []string{"n"},
			Usage:       "Namespace to use for the k8s-kata-manager",
			Destination: &worker.Namespace,
			EnvVars:     []string{"POD_NAMESPACE"},
		},
	}

	c.Before = func(c *cli.Context) error {
		// Check if a namespace was specified
		if worker.Namespace == "" {
			klog.Warning("No namespace specified, using current namespace")
			worker.Namespace = k8scli.GetKubernetesNamespace()
		}
		// set klog log level
		fs := flag.NewFlagSet("", flag.PanicOnError)
		klog.InitFlags(fs)
		return fs.Set("v", strconv.Itoa(c.Int("loglevel")))
	}

	err := c.Run(os.Args)
	if err != nil {
		klog.Errorf("%v", err)
		os.Exit(1)
	}
}

func (w *worker) configure(filepath string) error {
	c := newDefaultConfig()

	// Try to read and parse config file
	if filepath != "" {
		data, err := os.ReadFile(filepath)
		if err != nil {
			if os.IsNotExist(err) {
				klog.Infof("config file %q not found, using defaults", filepath)
			} else {
				return fmt.Errorf("error reading config file: %s", err)
			}
		} else {
			err = yaml.Unmarshal(data, c)
			if err != nil {
				return fmt.Errorf("failed to parse config file: %s", err)
			}

			klog.Infof("configuration file %q parsed", filepath)
		}
	} else {
		klog.Info("no config file specified, using defaults")
	}

	w.Config = c

	return nil
}

func newOSWatcher(sigs ...os.Signal) chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, sigs...)

	return sigChan
}

func (w *worker) Run(c *cli.Context) error {
	defer func() {
		klog.Info("Exiting")
	}()

	klog.Info("Starting OS watcher.")
	sigs := newOSWatcher(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	klog.Infof("K8s-kata-manager Worker %s", version.Get())
	klog.Infof("NodeName: '%s'", k8scli.NodeName())
	klog.Infof("Kubernetes namespace: '%s'", w.kubernetesNamespace)

	if err := w.configure(w.ConfigFilePath); err != nil {
		return err
	}

	for _, rc := range w.Config.RuntimeClass {
		klog.Infof("RuntimeClass: '%s'", rc.Name)
		// TODO create an Artifact, pull and configure containerd
	}

	// TODO: clean up on exit
	klog.Info("Watching for signals")
	for {
		select {
		// Watch for any signals from the OS. On SIGHUP trigger a reload of the config.
		// On all other signals, exit the loop and exit the program.
		case s := <-sigs:
			switch s {
			case syscall.SIGHUP:
				klog.Info("Received SIGHUP, restarting.")
				return nil
			default:
				klog.Infof("Received signal %v, shutting down.", s)
				return nil
			}
		case <-w.stop:
			klog.Infof("shutting down k8s-kata-manager-worker")
			return nil
		}
	}
}

// Stop k8s-kata-manager
func (w *worker) Stop() {
	select {
	case w.stop <- struct{}{}:
	default:
	}
}
