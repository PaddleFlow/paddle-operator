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
	"os"

	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
	"github.com/spf13/cobra"
)

var (
	rmrJobOptions v1alpha1.RmrJobOptions
	syncJobOptions v1alpha1.SyncJobOptions
	clearJobOptions v1alpha1.ClearJobOptions
	warmupJobOptions v1alpha1.WarmupJobOptions

	rootCmdOptions common.RootCmdOptions
	runtimeServerOptions common.RuntimeServerOptions
)

var rootCmd = &cobra.Command{
	Use:   "run",
	Short: "run server or job command",
}

var serverCmd = &cobra.Command{
	Use: "server",
	Short: "run server",
	Run: func(cmd *cobra.Command, args []string) {
		println("server ...")
	},
}

var syncJobCmd = &cobra.Command{
	Use:   common.SyncJobCmd,
	Short: "sync data or metadata from source to the cache engine",
	Run: func(cmd *cobra.Command, args []string) {
		println("sync job ...")
	},
}

var warmupJobCmd = &cobra.Command{
	Use:   common.WarmupJobCmd,
	Short: "warm up data from remote storage to local host",
	Run: func(cmd *cobra.Command, args []string) {
		println("warmup job ...")
	},
}

var rmrJobCmd = &cobra.Command{
	Use:   common.RmrJobCmd,
	Short: "remove data from cache engine storage backend",
	Run: func(cmd *cobra.Command, args []string) {
		println("rmr job ...")
	},
}

var clearJobCmd = &cobra.Command{
	Use: common.ClearJobCmd,
	Short: "clear cache data from local host",
	Run: func(cmd *cobra.Command, args []string) {
		println("clear job ...")
	},
}

func initSyncJobOptions() {
	// initialize options for root command
	rootCmd.PersistentFlags().StringVar(&rootCmdOptions.Driver, "driver", "juicefs", "specify the cache engine")

	// initialize options for runtime server

	// initialize options for sync job command
	syncJobCmd.Flags().StringVar(&syncJobOptions.Source, "source", "", "data source that need sync to cache engine")
	syncJobCmd.Flags().StringVar(&syncJobOptions.Destination, "destination", "", "destination path data should sync to")
	syncJobCmd.Flags().StringVar(&syncJobOptions.Start, "start", "", "the first KEY to sync")
	syncJobCmd.Flags().StringVar(&syncJobOptions.End, "end", "", "the last KEY to sync")
	syncJobCmd.Flags().IntVar(&syncJobOptions.Threads, "threads", 10, "number of concurrent threads (default: 10)")
	syncJobCmd.Flags().IntVar(&syncJobOptions.HttpPort, "http-port", 6070, "HTTP PORT to listen to (default: 6070)")
	syncJobCmd.Flags().BoolVar(&syncJobOptions.Update, "update", false, "update existing file if the source is newer (default: false)")
	syncJobCmd.Flags().BoolVar(&syncJobOptions.ForceUpdate, "force-update", false, "always update existing file (default: false)")
	syncJobCmd.Flags().BoolVar(&syncJobOptions.Perms, "perms", false, "preserve permissions (default: false)")
	syncJobCmd.Flags().BoolVar(&syncJobOptions.Dirs, "dirs", false, "Sync directories or holders (default: false)")
	syncJobCmd.Flags().BoolVar(&syncJobOptions.Dirs, "dry", false, "Don't copy file (default: false)")
	syncJobCmd.Flags().BoolVar(&syncJobOptions.DeleteSrc, "delete-src", false, "delete objects from source after synced (default: false)")
	syncJobCmd.Flags().BoolVar(&syncJobOptions.DeleteDst, "delete-dst", false, "delete extraneous objects from destination (default: false)")
	syncJobCmd.Flags().StringVar(&syncJobOptions.Exclude, "exclude", "", "exclude keys containing PATTERN (POSIX regular expressions)")
	syncJobCmd.Flags().StringVar(&syncJobOptions.Include, "include", "", "only include keys containing PATTERN (POSIX regular expressions)")
	syncJobCmd.Flags().StringVar(&syncJobOptions.Manager, "manager", "", "manager address")
	syncJobCmd.Flags().StringVar(&syncJobOptions.Worker, "worker", "", "hosts (seperated by comma) to launch worker")
	syncJobCmd.Flags().IntVar(&syncJobOptions.BWLimit, "bwlimit", 0, "limit bandwidth in Mbps (0 means unlimited) (default: 0)")
	syncJobCmd.Flags().BoolVar(&syncJobOptions.NoHttps, "no-https", false, "do not use HTTPS (default: false)")

	// initialize options for warmup job command
	warmupJobCmd.Flags().StringSliceVar(&warmupJobOptions.Path, "path", nil, "A list of paths need to build cache")
	warmupJobCmd.Flags().StringVar(&warmupJobOptions.File, "file", "", "file containing a list of paths")
	warmupJobCmd.Flags().IntVar(&warmupJobOptions.Threads, "threads", 50, "number of concurrent workers (default: 50)")


}

func init() {
	initSyncJobOptions()

	rootCmd.AddCommand(serverCmd, syncJobCmd, warmupJobCmd, rmrJobCmd, clearJobCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
