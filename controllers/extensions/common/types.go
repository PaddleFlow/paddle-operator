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

package common

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync/atomic"
)

type ReconcileContext struct {
	//
	client.Client

	//
	Ctx context.Context

	//
	Req *ctrl.Request

	//
	Log logr.Logger

	//
	Scheme *runtime.Scheme

	//
	Recorder record.EventRecorder
}

type RequestContext struct {
	//
	Req *ctrl.Request

	//
	SampleSet *v1alpha1.SampleSet

	//
	Secret *v1.Secret

	//
	PV *v1.PersistentVolume

	Service *v1.Service
}

type JobsName atomic.Value

// RootCmdOptions the
type RootCmdOptions struct {
	// container storage interface driver name,
	// the value of it should in v1alpha1.DriverName
	Driver string
	// configures the logger to use a Zap development config
	Development bool
}

type ServerOptions struct {
	ServerPort int      `json:"serverPort,omitempty"`
	ServerDir  string   `json:"serverDir,omitempty"`
	CacheDirs  []string `json:"cacheDirs,omitempty"`
	DataDir    string   `json:"dataDir,omitempty"`
}

type JobStatus string

type JobResult struct {
	Status  JobStatus `json:"status,omitempty"`
	Message string    `json:"message,omitempty"`
}
