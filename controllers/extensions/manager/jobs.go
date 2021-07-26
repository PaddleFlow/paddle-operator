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

package manager

import (
	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
	"github.com/paddleflow/paddle-operator/controllers/extensions/driver"
)

type Job interface {

}

type BaseJob struct {
	driver.Driver
}

func NewBaseJob(options common.RootCmdOptions) *BaseJob {

	return &BaseJob{

	}
}

type SyncJob struct {
	BaseJob
}

func NewSyncJob(rootOpt common.RootCmdOptions, syncOpt v1alpha1.SyncJobOptions) {

}


type WarmupJob struct {
	BaseJob
}

type RmrJob struct {
	BaseJob
}

type ClearJob struct {
	BaseJob
}