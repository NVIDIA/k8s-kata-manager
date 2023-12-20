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

package pull

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"oras.land/oras-go/v2/registry/remote/auth"

	"github.com/NVIDIA/k8s-kata-manager/internal/oras"
)

type command struct {
	logger *logrus.Logger
}

type options struct {
	output   string
	username string
	password string
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
	opts := options{}

	// Create the 'pull' command
	c := cli.Command{
		Name:      "pull",
		Usage:     "Pull files from a remote registry",
		UsageText: "kata-manager pull [flags] <name>{:<tag>|@<digest>}",
		Before: func(c *cli.Context) error {
			err := m.validateArgs(c)
			if err != nil {
				return fmt.Errorf("failed to parse arguments: %v", err)
			}
			err = m.validateFlags(c, &opts)
			if err != nil {
				return fmt.Errorf("failed to parse flags: %v", err)
			}
			return nil
		},
		Action: func(c *cli.Context) error {
			return m.run(c, &opts)
		},
	}

	c.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "output",
			Aliases:     []string{"o"},
			Usage:       "Output directory (default '.')",
			Value:       ".",
			Destination: &opts.output,
			EnvVars:     []string{"NVORAS_PULL_OUTPUT"},
		},
		&cli.StringFlag{
			Name:        "username",
			Aliases:     []string{"u"},
			Usage:       "registry username",
			Value:       "",
			Destination: &opts.username,
			EnvVars:     []string{"NVORAS_PULL_USERNAME"},
		},
		// TODO: make this secure
		&cli.StringFlag{
			Name:        "password",
			Aliases:     []string{"p"},
			Usage:       "registry password",
			Value:       "",
			Destination: &opts.password,
			EnvVars:     []string{"NVORAS_PULL_PASSWORD"},
		},
	}

	return &c
}

func (m command) validateArgs(c *cli.Context) error {
	if c.Args().Len() != 1 {
		return fmt.Errorf("unexpected number of positional arguments")
	}

	ref := c.Args().Get(0)
	refSplit := strings.Split(ref, "/")
	if len(refSplit) == 0 {
		return fmt.Errorf("unable to parse the registry")
	}

	if idx := strings.LastIndexAny(ref, "@:"); idx == -1 || (idx != -1 && ref[idx] != '@' && ref[idx] != ':') {
		return fmt.Errorf("unable to parse tag or digest")
	}

	return nil
}

func (m command) validateFlags(_ *cli.Context, _ *options) error { return nil }

func (m command) run(c *cli.Context, opts *options) error {

	ref := c.Args().Get(0)
	art, err := oras.NewArtifact(ref, opts.output)
	if err != nil {
		return fmt.Errorf("failed to create oras artifact: %v", err)
	}
	m.logger.Infof("Artifact: %v", art)

	creds := &auth.Credential{
		Username: opts.username,
		Password: opts.password,
	}

	m.logger.Infof("Pulling %s...\n", ref)
	manifest, err := art.Pull(creds)
	if err != nil {
		return fmt.Errorf("failed to pull %s: %v", ref, err)
	}

	m.logger.Infof("Successfully pulled %s", ref)
	m.logger.Debugf("Manifest descriptor: %v", manifest)
	return nil
}
