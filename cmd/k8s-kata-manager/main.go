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
	"syscall"
	"time"

	api "github.com/NVIDIA/k8s-kata-manager/api/v1alpha1/config"
	k8scli "github.com/NVIDIA/k8s-kata-manager/internal/client-go"
	"github.com/NVIDIA/k8s-kata-manager/internal/containerd"
	"github.com/NVIDIA/k8s-kata-manager/internal/oras"
	version "github.com/NVIDIA/k8s-kata-manager/internal/version"
	"golang.org/x/sys/unix"

	cli "github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"

	klog "k8s.io/klog/v2"
	yaml "sigs.k8s.io/yaml"
)

const (
	defaultContainerdConfigFilePath = "/etc/containerd/config.toml"
	defaultContainerdSocketFilePath = "/run/containerd/containerd.sock"
)

var waitingForSignal = make(chan bool, 1)
var signalReceived = make(chan bool, 1)

var (
	pidFile = filepath.Join(api.DefaultKataArtifactsDir, "k8s-kata-manager.pid")
)

// Worker is the interface for k8s-kata-manager daemon
type Worker interface {
	Run() error
	Stop()
}

type worker struct {
	Config         *api.Config
	Namespace      string
	ConfigFilePath string

	ContainerdConfig string
	ContainerdSocket string
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
		&cli.StringFlag{
			Name:        "containerd-config",
			Usage:       "Path to the containerd config file",
			Value:       defaultContainerdConfigFilePath,
			Destination: &worker.ContainerdConfig,
			EnvVars:     []string{"CONTAINERD_CONFIG"},
		},
		&cli.StringFlag{
			Name:        "containerd-socket",
			Usage:       "Path to the containerd socket file",
			Value:       defaultContainerdSocketFilePath,
			Destination: &worker.ContainerdSocket,
			EnvVars:     []string{"CONTAINERD_SOCKET"},
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

	api.SanitizeConfig(c)
	w.Config = c

	return nil
}

func (w *worker) Run(clictxt *cli.Context) error {
	defer func() {
		klog.Info("Exiting")
	}()

	klog.Infof("K8s-kata-manager Worker %s", version.Get())
	klog.Infof("NodeName: '%s'", k8scli.NodeName())
	klog.Infof("Kubernetes namespace: '%s'", w.Namespace)

	klog.Info("Parsing configuration file")
	if err := w.configure(w.ConfigFilePath); err != nil {
		return err
	}

	configYAML, err := yaml.Marshal(w.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to yaml: %v", err)
	}
	klog.Infof("Running with configuration:\n%v", string(configYAML))

	if w.Config.ArtifactsDir != api.DefaultKataArtifactsDir {
		pidFile = filepath.Join(w.Config.ArtifactsDir, "k8s-kata-manager.pid")
	}

	//TODO move to subcommand or internal.pkg
	k8scli := k8scli.NewClient(w.Namespace)

	if err := initialize(); err != nil {
		return fmt.Errorf("unable to initialize: %v", err)
	}
	defer shutdown()

	ctrdConfig, err := containerd.New(
		containerd.WithPath(w.ContainerdConfig),
		containerd.WithPodAnnotations("io.katacontainers.*"),
		containerd.WithRuntimeType("io.containerd.kata.v2"),
	)
	if err != nil {
		klog.Errorf("error creating containerd.config client : %s", err)
		return err
	}

	for _, rc := range w.Config.RuntimeClasses {
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

		kataConfigCandidates, err := filepath.Glob(filepath.Join(rcDir, "*.toml"))
		if err != nil {
			return fmt.Errorf("error searching for kata config file: %v", err)
		}
		if len(kataConfigCandidates) == 0 {
			return fmt.Errorf("no kata config file found for runtime class %s", rc.Name)
		}
		kataConfigPath := kataConfigCandidates[0]

		err = ctrdConfig.AddRuntime(
			rc.Name,
			kataConfigPath,
			false,
		)
		if err != nil {
			return fmt.Errorf("unable to update config: %v", err)
		}

	}

	n, err := ctrdConfig.Save(w.ContainerdConfig)
	if err != nil {
		return fmt.Errorf("unable to flush config: %v", err)
	}

	if n == 0 {
		klog.Infof("Removed empty config from %v", w.ContainerdConfig)
	} else {
		klog.Infof("Wrote updated config to %v", w.ContainerdConfig)
	}

	klog.Infof("Restarting containerd")
	if err := restartContainerd(w.ContainerdSocket); err != nil {
		return fmt.Errorf("unable to restart containerd: %v", err)
	}
	klog.Info("containerd successfully restarted")

	if err := waitForSignal(); err != nil {
		return fmt.Errorf("unable to wait for signal: %v", err)
	}

	if err := w.CleanUp(); err != nil {
		return fmt.Errorf("unable to revert config: %v", err)
	}

	return nil
}

// RevertConfig reverts the containerd config to remove the nvidia-container-runtime
func (w *worker) CleanUp() error {
	ctrdConfig, err := containerd.New(
		containerd.WithPath(w.ContainerdConfig),
	)
	if err != nil {
		klog.Errorf("error creating containerd.config client : %s", err)
		return err
	}
	for _, rc := range w.Config.RuntimeClasses {
		err := ctrdConfig.RemoveRuntime(rc.Name)
		if err != nil {
			return fmt.Errorf("unable to revert config for runtime class '%v': %v", rc, err)
		}
	}
	n, err := ctrdConfig.Save(w.ContainerdConfig)
	if err != nil {
		return fmt.Errorf("unable to flush config: %v", err)
	}

	if n == 0 {
		klog.Infof("Removed empty config from %v", w.ContainerdConfig)
	} else {
		klog.Infof("Wrote updated config to %v", w.ContainerdConfig)
	}
	if err := restartContainerd(w.ContainerdSocket); err != nil {
		return fmt.Errorf("unable to restart containerd: %v", err)
	}
	return nil
}

func initialize() error {
	klog.Infof("Initializing")

	f, err := os.Create(pidFile)
	if err != nil {
		return fmt.Errorf("unable to create pidfile: %v", err)
	}

	err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err != nil {
		klog.Warningf("Unable to get exclusive lock on '%v'", pidFile)
		klog.Warningf("This normally means an instance of the NVIDIA k8s-kata-manager Container is already running, aborting")
		return fmt.Errorf("unable to get flock on pidfile: %v", err)
	}

	_, err = f.WriteString(fmt.Sprintf("%v\n", os.Getpid()))
	if err != nil {
		return fmt.Errorf("unable to write PID to pidfile: %v", err)
	}

	return nil
}

func restartContainerd(containerdSocket string) error {

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
		err := containerd.RestartContainerd(containerdSocket)
		if err != nil {
			klog.Errorf("error restarting containerd: %v", err)
			done <- err
		}
		// Since we are restarintg Containerd we need to
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

func waitForSignal() error {
	klog.Infof("Waiting for signal")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGPIPE, syscall.SIGTERM)
	go func() {
		<-sigs
		select {
		case <-waitingForSignal:
			signalReceived <- true
		default:
			klog.Infof("Signal received, exiting early")
			shutdown()
			os.Exit(0)
		}
	}()

	waitingForSignal <- true
	<-signalReceived
	return nil
}

func shutdown() {
	klog.Infof("Shutting Down")

	if err := os.Remove(pidFile); err != nil {
		klog.Warningf("Unable to remove pidfile: %v", err)
	}
}
