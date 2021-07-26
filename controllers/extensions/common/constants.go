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

import "github.com/paddleflow/paddle-operator/api/v1alpha1"

const (
	// Memory use
	Memory v1alpha1.MediumType = "MEM"
	// SSD use
	SSD v1alpha1.MediumType = "SSD"
	// HDD use
	HDD v1alpha1.MediumType = "HDD"
)

const (
	// SampleSetNone After create SampleSet CR, before create PV/PVC
	SampleSetNone   v1alpha1.SampleSetPhase = ""
	// SampleSetBound After create PV/PVC, before create runtime daemon set
	SampleSetBound v1alpha1.SampleSetPhase = "Bound"
	// SampleSetMount After create runtime daemon set, before data sync finish
	SampleSetMount v1alpha1.SampleSetPhase = "Mount"
	// SampleSetPartialReady means
	SampleSetPartialReady v1alpha1.SampleSetPhase = "PartialReady"
	// SampleSetReady After data sync finish and SampleSet is ready to be use
	SampleSetReady v1alpha1.SampleSetPhase = "Ready"
	// SampleSetFailed Not bound to runtime, can be deleted
	SampleSetFailed v1alpha1.SampleSetPhase = "Failed"
	// SampleSetScaling a
	SampleSetScaling v1alpha1.SampleSetPhase = "Scaling"

)

const (
	SampleJobNone v1alpha1.SampleJobPhase = ""
	// SampleJobPending Bound to runtime, can't be deleted
	SampleJobPending v1alpha1.SampleJobPhase = "Pending"
	// SampleJobExecuting Bound to dataset, can't be released
	SampleJobExecuting v1alpha1.SampleJobPhase = "Executing"
	// SampleJobComplete Complete Ready can't be deleted
	SampleJobComplete v1alpha1.SampleJobPhase = "Complete"
)

const (
	// RmrJob use
	RmrJob v1alpha1.SampleJobType = "rmr"
	// SyncJob a
	SyncJob v1alpha1.SampleJobType = "sync"
	// ClearJob a
	ClearJob v1alpha1.SampleJobType = "clear"
	// WarmupJob a
	WarmupJob v1alpha1.SampleJobType = "warmup"
)

const (
	NormalCreate = "SuccessfulCreate"
)

const (


	ErrorDriverNotExist = "ErrorDriverNotExist"

	ErrorSecretNotExist = "ErrorSecretNotExist"

	ErrorPVAlreadyExist = "ErrorPersistentVolumeAlreadyExist"

	ErrorCreatePV = "ErrorCreatePersistentVolume"

	ErrorPVCAlreadyExist = "ErrorPersistentVolumeClaimAlreadyExist"

	ErrorCreatePVC = "ErrorCreatePersistentVolumeClaim"

	ErrorSSAlreadyExist = "ErrorStatefulSetAlreadyExist"

	ErrorCreateRuntime = "ErrorCreateRuntime"

	ErrorDeleteRuntime = "ErrorDeleteRuntime"
	ErrorDeletePVC = "ErrorDeletePersistentVolumeClaim"
	ErrorDeletePV = "ErrorDeletePersistentVolume"
	ErrorDeleteSampleSet = "ErrorDeleteSampleSet"
)

const (
	ResourceStorage = "10Pi"
)

const (
	PaddleLabel = "paddlepaddle.org"
	PaddleOperatorLabel = "paddle-operator"
)

const (
	EventIndexerKey = "eventIndexerKey"
	NodeIndexerKey = "nodeIndexerKey"
	RuntimeIndexerKey = "runtimeIndexerKey"
)

const (
	RuntimeContainerName = "runtime"
	RuntimeCacheMountPath = "/cache"
	RuntimeDateMountPath = "/mnt"
)

const (
	ServerCmd = "server"

	// SyncJobCmd defined the command to sync data and metadata
	// from source to destination for job or cronjob
	SyncJobCmd = "sync"
	// WarmupJobCmd the
	WarmupJobCmd = "warmup"
	// RmrJobCmd
	RmrJobCmd = "rmr"
	// ClearJobCmd
	ClearJobCmd = "clear"
)