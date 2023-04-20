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
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
)

func TestNewCommand(t *testing.T) {
	type args struct {
		logger *logrus.Logger
	}
	tests := []struct {
		name string
		args args
		want *cli.Command
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewCommand(tt.args.logger); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_command_build(t *testing.T) {
	type fields struct {
		logger *logrus.Logger
	}
	tests := []struct {
		name   string
		fields fields
		want   *cli.Command
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := command{
				logger: tt.fields.logger,
			}
			if got := m.build(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("command.build() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseReference(t *testing.T) {
	type args struct {
		ref string
	}
	tests := []struct {
		name           string
		args           args
		wantRegistry   string
		wantRepository string
		wantTag        string
		wantErr        bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRegistry, gotRepository, gotTag, err := parseReference(tt.args.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseReference() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotRegistry != tt.wantRegistry {
				t.Errorf("parseReference() gotRegistry = %v, want %v", gotRegistry, tt.wantRegistry)
			}
			if gotRepository != tt.wantRepository {
				t.Errorf("parseReference() gotRepository = %v, want %v", gotRepository, tt.wantRepository)
			}
			if gotTag != tt.wantTag {
				t.Errorf("parseReference() gotTag = %v, want %v", gotTag, tt.wantTag)
			}
		})
	}
}

func Test_command_run(t *testing.T) {
	type fields struct {
		logger *logrus.Logger
	}
	type args struct {
		c    *cli.Context
		opts *options
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := command{
				logger: tt.fields.logger,
			}
			if err := m.run(tt.args.c, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("command.run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_command_validateArgs(t *testing.T) {
	type fields struct {
		logger *logrus.Logger
	}
	type args struct {
		c    *cli.Context
		opts *options
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := command{
				logger: tt.fields.logger,
			}
			if err := m.validateArgs(tt.args.c, tt.args.opts); (err != nil) != tt.wantErr {
				t.Errorf("command.validateArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
