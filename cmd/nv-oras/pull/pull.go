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
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type command struct {
	logger *logrus.Logger
}

type options struct {
	// reference contains the tag or digest
	reference  string
	registry   string
	repository string
	tag        string

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
		UsageText: "nv-oras pull [flags] <name>{:<tag>|@<digest>}",
		Before: func(c *cli.Context) error {
			err := m.validateArgs(c, &opts)
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
			Usage:       "Output directory (default '.')",
			Value:       ".",
			Destination: &opts.output,
			EnvVars:     []string{"NVORAS_PULL_OUTPUT"},
		},
		&cli.StringFlag{
			Name:        "username",
			Usage:       "registry username",
			Value:       "",
			Destination: &opts.username,
			EnvVars:     []string{"NVORAS_PULL_USERNAME"},
		},
		// TODO: make this secure
		&cli.StringFlag{
			Name:        "password",
			Usage:       "registry password",
			Value:       "",
			Destination: &opts.password,
			EnvVars:     []string{"NVORAS_PULL_PASSWORD"},
		},
	}

	return &c
}

func (m command) validateArgs(c *cli.Context, opts *options) error {
	if c.Args().Len() != 1 {
		return fmt.Errorf("unexpected number of positional arguments")
	}

	ref := c.Args().Get(0)
	reg, repo, tag, err := parseReference(ref)
	if err != nil {
		return fmt.Errorf("failed to parse reference: %v", err)
	}
	opts.reference = ref
	opts.registry = reg
	opts.repository = repo
	opts.tag = tag

	return nil
}

func (m command) validateFlags(c *cli.Context, opts *options) error {
	return nil
}

// parseReference parses the raw input in format <path>[:<tag>|@<digest>]
func parseReference(ref string) (registry string, repository string, tag string, err error) {
	refSplit := strings.Split(ref, "/")
	if len(refSplit) == 0 {
		err = fmt.Errorf("unable to parse the registry")
		return
	}
	registry = refSplit[0]

	if idx := strings.LastIndex(ref, "@"); idx != -1 {
		repository = ref[:idx]
		tag = ref[idx+1:]
	} else if idx := strings.LastIndex(ref, ":"); idx != 1 {
		repository = ref[:idx]
		tag = ref[idx+1:]
	} else {
		err = fmt.Errorf("unable to parse tag or digest")
	}

	return
}

func (m command) run(c *cli.Context, opts *options) error {
	// Create a file store
	fs, err := file.New(opts.output)
	if err != nil {
		return err
	}
	defer fs.Close()

	// Connect to a remote repository
	ctx := context.Background()
	repo, err := remote.NewRepository(opts.repository)
	if err != nil {
		return err
	}

	if opts.username != "" && opts.password != "" {
		repo.Client = &auth.Client{
			Client: retry.DefaultClient,
			Cache:  auth.DefaultCache,
			Credential: auth.StaticCredential(opts.registry, auth.Credential{
				Username: opts.username,
				Password: opts.password,
			}),
		}
	}

	// Copy from the remote repository to the file store
	manifestDescriptor, err := oras.Copy(ctx, repo, opts.tag, fs, opts.tag, oras.DefaultCopyOptions)
	if err != nil {
		return err
	}
	m.logger.Infof("Successfully pulled %s", opts.reference)
	m.logger.Debugf("Manifest descriptor: %v", manifestDescriptor)
	return nil
}
