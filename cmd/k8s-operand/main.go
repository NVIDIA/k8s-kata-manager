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
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	api "github.com/NVIDIA/k8s-kata-manager/api/v1alpha1/config"
	k8scli "github.com/NVIDIA/k8s-kata-manager/internal/client-go"
	"github.com/NVIDIA/k8s-kata-manager/internal/containerd"
	"github.com/NVIDIA/k8s-kata-manager/internal/oras"
	version "github.com/NVIDIA/k8s-kata-manager/internal/version"

	cli "github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"

	klog "k8s.io/klog/v2"
	yaml "sigs.k8s.io/yaml"
)

const (
	defaultContainerdConfigFilePath = "/etc/containerd/config.toml"
)

// Worker is the interface for k8s-kata-manager daemon
type Worker interface {
	Run() error
	Stop()
}

type worker struct {
	stop chan struct{} // channel for signaling stop

	Config         *api.Config
	Namespace      string
	ConfigFilePath string
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
				Name:    "log-level",
				Usage:   "Set the logging level",
				Aliases: []string{"l"},
				Value:   1}),
		&cli.StringFlag{
			Name:        "config-file",
			Value:       "/etc/kubernetes/kata-manager/config.yaml", // Default value
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
	c := api.NewDefaultConfig()

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

func (w *worker) Run(clictxt *cli.Context) error {
	defer func() {
		klog.Info("Exiting")
	}()

	klog.Info("Starting OS watcher.")
	sigs := newOSWatcher(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	klog.Infof("K8s-kata-manager Worker %s", version.Get())
	klog.Infof("NodeName: '%s'", k8scli.NodeName())
	klog.Infof("Kubernetes namespace: '%s'", w.Namespace)

	if err := w.configure(w.ConfigFilePath); err != nil {
		return err
	}

	//TODO move to subcommand or internal.pkg
	k8scli := k8scli.NewClient(w.Namespace)

	for _, rc := range w.Config.RuntimeClass {
		creds, err := k8scli.GetCredentials(rc, w.Namespace)
		if err != nil {
			klog.Errorf("error getting credentials: %s", err)
			return err
		}
		rcDir := filepath.Join(w.Config.ArtifactsDir, rc.Name)
		if _, err := os.Stat(rcDir); os.IsNotExist(err) {
			err := os.Mkdir(rcDir, 0755)
			if err != nil {
				klog.Errorf("error creating artifact directory: %s", err)
				return err
			}
		}
		a, err := oras.NewArtifact(rc.Artifacts.URL, rcDir)
		if err != nil {
			klog.Errorf("error creating artifact: %s", err)
			return err
		}
		_, err = a.Pull(creds)
		if err != nil {
			klog.Errorf("error pulling artifact: %s", err)
			return err
		}

		ctrdConfig, err := containerd.New(
			containerd.WithPath(defaultContainerdConfigFilePath),
			containerd.WithPodAnnotations("io.katacontainers.*"),
		)
		if err != nil {
			klog.Errorf("error creating containerd.config client : %s", err)
			return err
		}

		var setAsDefault bool
		if strings.EqualFold(rc.SetAsDefault, "true") {
			setAsDefault = true
		}

		runtime := fmt.Sprintf("kata-qemu-%s", rc.Name)
		ctrdConfig.RuntimeType = fmt.Sprintf("io.containerd.%s.v2", runtime)
		err = ctrdConfig.AddRuntime(
			runtime,
			rcDir,
			setAsDefault,
		)
		if err != nil {
			return fmt.Errorf("unable to update config: %v", err)
		}

		n, err := ctrdConfig.Save(defaultContainerdConfigFilePath)
		if err != nil {
			return fmt.Errorf("unable to flush config: %v", err)
		}

		if n == 0 {
			klog.Infof("Removed empty config from %v", defaultContainerdConfigFilePath)
		} else {
			klog.Infof("Wrote updated config to %v", defaultContainerdConfigFilePath)
		}
		// TODO Reload containerd
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
