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
	"fmt"
	"github.com/go-logr/logr"
	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
	"github.com/paddleflow/paddle-operator/controllers/extensions/driver"
	zapOpt "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type Job struct {
	driver.Driver
	Log logr.Logger

	rootOpt   *common.RootCmdOptions
}

func newJob(rootOpt *common.RootCmdOptions) (*Job, error) {
	driverName := v1alpha1.DriverName(rootOpt.Driver)
	csiDriver, err :=  driver.GetDriver(driverName)
	if err != nil {
		return nil, err
	}

	// configure zap log and create a logger
	zapLog := zap.New(func(o *zap.Options) {
		o.Development = rootOpt.Development
	}, func(o *zap.Options) {
		o.ZapOpts = append(o.ZapOpts, zapOpt.AddCaller())
	},
		func(o *zap.Options) {
			if !rootOpt.Development {
				encCfg := zapOpt.NewProductionEncoderConfig()
				encCfg.EncodeLevel = zapcore.CapitalLevelEncoder
				encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
				o.Encoder = zapcore.NewConsoleEncoder(encCfg)
			}
		})

	job := &Job{
		Log: zapLog,
		rootOpt: rootOpt,
		Driver: csiDriver,
	}

	return job, nil
}

func (j *Job) Run() error {
	panic("Not implement")
}

type SyncJob struct {
	*Job
	syncOpt *v1alpha1.SyncJobOptions
}

func NewSyncJob(rootOpt *common.RootCmdOptions,
	syncOpt *v1alpha1.SyncJobOptions) (*SyncJob, error) {
	job, err := newJob(rootOpt)
	if err != nil {
		return nil, err
	}
	// valid options

	syncJob := &SyncJob{
		Job: job,
		syncOpt: syncOpt,
	}
	return syncJob, nil
}

func (s *SyncJob) Run() error {
	if err := s.DoSyncJob(s.syncOpt); err != nil {
		s.Log.Error(err, "error occur when do sync job")
		return err
	}
	s.Log.V(1).Info("sync job finish")
	return nil
}

type WarmupJob struct {
	*Job
	warmupOpt *v1alpha1.WarmupJobOptions
}

func NewWarmupJob(rootOpt *common.RootCmdOptions,
	warmupOpt *v1alpha1.WarmupJobOptions) (*WarmupJob, error) {
	job, err := newJob(rootOpt)
	if err != nil {
		return nil, err
	}
	// valid options
	if len(warmupOpt.Paths) == 0 {
		return nil, fmt.Errorf("paths not set")
	}

	warmupJob := &WarmupJob{
		Job: job,
		warmupOpt: warmupOpt,
	}
	return warmupJob, nil
}

func (w *WarmupJob) Run() error {
	if err := w.DoWarmupJob(w.warmupOpt); err != nil {
		w.Log.Error(err, "error occur when do warmup job")
	}
	w.Log.V(1).Info("warmup job finish")
	return nil
}

type RmrJob struct {
	*Job
	rmrOpt *v1alpha1.RmrJobOptions
}

func NewRmrJob(rootOpt *common.RootCmdOptions,
	rmrOpt *v1alpha1.RmrJobOptions) (*RmrJob, error) {
	job, err := newJob(rootOpt)
	if err != nil {
		return nil, err
	}
	// valid options
	if len(rmrOpt.Paths) == 0 {
		return nil, fmt.Errorf("paths not set")
	}

	rmrJob := &RmrJob{
		Job: job,
		rmrOpt: rmrOpt,
	}
	return rmrJob, nil
}

func (r *RmrJob) Run() error {
	if err := r.DoRmrJob(r.rmrOpt); err != nil {
		r.Log.Error(err, "error occur when do rmr job")
	}
	r.Log.V(1).Info("rmr job finish")
	return nil
}

type ClearJob struct {
	*Job
	clearOpt *v1alpha1.ClearJobOptions
}

func NewClearJob(rootOpt *common.RootCmdOptions,
	clearOpt *v1alpha1.ClearJobOptions) (*ClearJob, error) {
	job, err := newJob(rootOpt)
	if err != nil {
		return nil, err
	}
	// valid options
	if len(clearOpt.Paths) == 0 {
		return nil, fmt.Errorf("paths not set")
	}

	rmrJob := &ClearJob{
		Job: job,
		clearOpt: clearOpt,
	}
	return rmrJob, nil
}

func (c *ClearJob) Run() error {
	if err := c.DoClearJob(c.clearOpt); err != nil {
		c.Log.Error(err, "error occur when do clear job")
	}
	c.Log.V(1).Info("clear job finish")
	return nil
}