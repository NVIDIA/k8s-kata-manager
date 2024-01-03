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

package containerd

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/NVIDIA/k8s-kata-manager/internal/containerd"
)

const (
	defaultContainerdSocketFilePath = "/run/containerd/containerd.sock"
)

type command struct {
	logger *logrus.Logger
}

// NewCommand constructs a pull command with the specified logger
func NewCommand(logger *logrus.Logger) *cli.Command {
	c := command{
		logger: logger,
	}
	return c.build()
}

// build creates the CLI command
func (m command) build() *cli.Command {
	// Create the 'containerd' command
	c := cli.Command{
		Name:      "containerd",
		Usage:     "containerd restarts containerd in a safe way",
		UsageText: "kata-manager containerd <socket-path>",
		Before: func(c *cli.Context) error {
			err := m.validateArgs(c)
			if err != nil {
				return fmt.Errorf("failed to parse arguments: %w", err)
			}
			err = m.validateFlags(c)
			if err != nil {
				return fmt.Errorf("failed to parse flags: %w", err)
			}
			return nil
		},
		Action: m.run,
	}

	return &c
}

func (m command) validateArgs(c *cli.Context) error {
	if c.Args().Len() != 1 {
		return fmt.Errorf("unexpected number of positional arguments")
	}
	return nil
}

func (m command) validateFlags(_ *cli.Context) error { return nil }

func (m command) run(c *cli.Context) error {
	containerdSocketFilePath := c.Args().Get(0)
	var err error
	if containerdSocketFilePath == "" {
		err = containerd.RestartContainerd(defaultContainerdSocketFilePath)
	} else {
		err = containerd.RestartContainerd(containerdSocketFilePath)
	}
	if err != nil {
		m.logger.Errorf("failed to restart containerd: %v", err)
		return err
	}
	return nil
}
