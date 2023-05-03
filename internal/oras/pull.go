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

package oras

import (
	"context"
	"fmt"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type artifact struct {
	registry   string
	repository string
	tag        string

	output string
}

func NewArtifact(ref string, output string) (*artifact, error) {
	var registry, repository, tag string

	refSplit := strings.Split(ref, "/")
	if len(refSplit) == 0 {
		return nil, fmt.Errorf("unable to parse the registry")
	}
	registry = refSplit[0]

	if idx := strings.LastIndex(ref, "@"); idx != -1 {
		repository = ref[:idx]
		tag = ref[idx+1:]
	} else if idx := strings.LastIndex(ref, ":"); idx != 1 {
		repository = ref[:idx]
		tag = ref[idx+1:]
	} else {
		return nil, fmt.Errorf("unable to parse tag or digest")
	}

	return &artifact{
		registry:   registry,
		repository: repository,
		tag:        tag,
		output:     output,
	}, nil
}

func (a artifact) Pull(creds auth.Credential) (ocispec.Descriptor, error) {
	// Create a file store
	fs, err := file.New(a.output)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	defer fs.Close()

	// Connect to a remote repository
	ctx := context.Background()
	repo, err := remote.NewRepository(a.repository)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	if creds.Username != "" && creds.Password != "" {
		repo.Client = &auth.Client{
			Client: retry.DefaultClient,
			Cache:  auth.DefaultCache,
			Credential: auth.StaticCredential(a.registry, auth.Credential{
				Username: creds.Username,
				Password: creds.Password,
			}),
		}
	}

	// Copy from the remote repository to the file store
	return oras.Copy(ctx, repo, a.tag, fs, a.tag, oras.DefaultCopyOptions)
}
