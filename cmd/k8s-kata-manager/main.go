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
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	fmt.Println("============================================================")
	fmt.Println("DEPRECATION NOTICE")
	fmt.Println("============================================================")
	fmt.Println("")
	fmt.Println("The k8s-kata-manager operand is deprecated and will be")
	fmt.Println("removed in a future release of the NVIDIA GPU Operator.")
	fmt.Println("")
	fmt.Println("This component no longer performs any operations.")
	fmt.Println("Please migrate to the recommended solution for running")
	fmt.Println("GPU workloads with Kata Containers.")
	fmt.Println("")
	fmt.Println("For more information, see:")
	fmt.Println("https://docs.nvidia.com/datacenter/cloud-native/gpu-operator/latest/")
	fmt.Println("============================================================")
	fmt.Println("")
	fmt.Println("Waiting for termination signal...")

	// Wait for termination signal to keep the pod running
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	fmt.Println("Received termination signal, exiting.")
}
