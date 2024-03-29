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

package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"

	api "github.com/NVIDIA/k8s-kata-manager/api/v1alpha1/config"
	utils "github.com/NVIDIA/k8s-kata-manager/internal/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// Auths struct contains an embedded RegistriesStruct of name auths
type Auths struct {
	Registries RegistriesStruct `json:"auths"`
}

// RegistriesStruct is a map of registries to their credentials
type RegistriesStruct map[string]RegistryCredentials

// RegistryCredentials defines the fields stored per registry in an docker config secret
type RegistryCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Auth     string `json:"auth"`
}

func (k *k8scli) GetCredentials(ctx context.Context, rc api.RuntimeClass) (*auth.Credential, error) {
	if rc.Artifacts.PullSecret == "" {
		return nil, nil
	}

	auths := Auths{}
	secret, err := k.Get(ctx, rc.Artifacts.PullSecret, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting secret: %w", err)
	}
	if err := json.Unmarshal(secret.Data[".dockerconfigjson"], &auths); err != nil {
		return nil, fmt.Errorf("error decoding secret: %w", err)
	}
	Registry, err := utils.ParseRegistry(rc.Artifacts.URL)
	if err != nil {
		return nil, fmt.Errorf("error parsing registry: %w", err)
	}

	creds := &auth.Credential{
		Username: auths.Registries[Registry].Username,
		Password: auths.Registries[Registry].Password,
	}
	return creds, nil
}
