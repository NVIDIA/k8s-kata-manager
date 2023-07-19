/**
# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
**/

package cdi

import (
	"github.com/NVIDIA/nvidia-container-toolkit/pkg/nvcdi"
	"gitlab.com/nvidia/cloud-native/go-nvlib/pkg/nvpci"
)

type nvcdilib struct {
	vendor string
	class  string

	nvpcilib nvpci.Interface
}

// New creates a new instance of this library
func New(opts ...Option) (nvcdi.Interface, error) {
	l := &nvcdilib{}
	for _, opt := range opts {
		opt(l)
	}

	if l.vendor == "" {
		l.vendor = "nvidia.com"
	}
	if l.class == "" {
		l.class = "pgpu"
	}
	if l.nvpcilib == nil {
		l.nvpcilib = nvpci.New()
	}

	return (*vfiolib)(l), nil
}

type Option func(*nvcdilib)

// WithNvpciLib sets the nvpci library for the library
func WithNvpciLib(nvpcilib nvpci.Interface) Option {
	return func(l *nvcdilib) {
		l.nvpcilib = nvpcilib
	}
}

// WithVendor sets the vendor for the library
func WithVendor(vendor string) Option {
	return func(o *nvcdilib) {
		o.vendor = vendor
	}
}

// WithClass sets the class for the library
func WithClass(class string) Option {
	return func(o *nvcdilib) {
		o.class = class
	}
}
