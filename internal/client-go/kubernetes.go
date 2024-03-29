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
	"os"
	"strings"

	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

var nodeName string

type k8scli struct {
	corev1.SecretInterface

	namespace string
}

func NewClient(namespace string) k8scli {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	k := k8scli{
		clientset.CoreV1().Secrets(namespace),
		namespace}
	return k
}

// NodeName returns the name of the k8s node we're running on.
func NodeName() string {
	if nodeName == "" {
		nodeName = os.Getenv("NODE_NAME")
	}
	return nodeName
}

// GetKubernetesNamespace returns the kubernetes namespace we're running under,
// or an empty string if the namespace cannot be determined.
func GetKubernetesNamespace() string {
	const kubernetesNamespaceFilePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	if _, err := os.Stat(kubernetesNamespaceFilePath); err == nil {
		data, err := os.ReadFile(kubernetesNamespaceFilePath)
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	return os.Getenv("KUBERNETES_NAMESPACE")
}
