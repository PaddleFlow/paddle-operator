// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/paddleflow/paddle-operator/controllers/extensions/executor"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"os"
	//+kubebuilder:scaffold:imports
)

var (
	syncJobOption executor.SyncJobOptions
)

var cmd = &cobra.Command{
	Use:   "exec",
	Short: "execute cache manage command",
}

var SyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "sync metadata to cache runtime engine",
	Run: func(cmd *cobra.Command, args []string) {

	},
}

var WarmupCmd = &cobra.Command{
	Use:   "warmup",
	Short: "sync metadata to cache runtime engine",
	Run: func(cmd *cobra.Command, args []string) {

	},
}

var RmrCmd = &cobra.Command{
	Use:   "rmr",
	Short: "sync metadata to cache runtime engine",
	Run: func(cmd *cobra.Command, args []string) {

	},
}

// RunCmd DaemonSet
var RunCmd = &cobra.Command{
	Use:   "rmr",
	Short: "sync metadata to cache runtime engine",
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {

	//
	SyncCmd.Flags().StringVarP(&syncJobOption.Source, "SRC", "", "", "")

	cmd.AddCommand(SyncCmd)
}

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
