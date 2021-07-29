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
	// MediumTypeMEM use
	MediumTypeMEM v1alpha1.MediumType = "MEM"
	// MediumTypeSSD use
	MediumTypeSSD v1alpha1.MediumType = "SSD"
	// MediumTypeHDD use
	MediumTypeHDD v1alpha1.MediumType = "HDD"
)

const (
	// SampleSetNone After create SampleSet CR, before create PV/PVC
	SampleSetNone   v1alpha1.SampleSetPhase = ""
	// SampleSetBound After create PV/PVC, before create runtime daemon set
	SampleSetBound  v1alpha1.SampleSetPhase = "Bound"
	// SampleSetMount After create runtime daemon set, before data sync finish
	SampleSetMount  v1alpha1.SampleSetPhase = "Mount"
	// SampleSetPartialReady means
	SampleSetPartialReady v1alpha1.SampleSetPhase = "PartialReady"
	// SampleSetReady After data sync finish and SampleSet is ready to be use
	SampleSetReady  v1alpha1.SampleSetPhase = "Ready"
	// SampleSetFailed Not bound to runtime, can be deleted
	SampleSetFailed v1alpha1.SampleSetPhase = "Failed"
)

const (
	JobTypeRmr    v1alpha1.SampleJobType = "rmr"
	JobTypeSync   v1alpha1.SampleJobType = "sync"
	JobTypeClear  v1alpha1.SampleJobType = "clear"
	JobTypeWarmup v1alpha1.SampleJobType = "warmup"
)

const (
	CmdServer = "server"
	// CmdSync defined the command to sync data and metadata
	// from source to destination for job or cronjob
	CmdSync   = "sync"
	CmdWarmup = "warmup"
	CmdRmr    = "rmr"
	CmdClear  = "clear"
)

const EventCreate = "SuccessfulCreate"

const (
	ErrorDriverNotExist  = "ErrorDriverNotExist"
	ErrorSecretNotExist  = "ErrorSecretNotExist"
	ErrorPVAlreadyExist  = "ErrorPersistentVolumeAlreadyExist"
	ErrorCreatePV        = "ErrorCreatePersistentVolume"
	ErrorPVCAlreadyExist = "ErrorPersistentVolumeClaimAlreadyExist"
	ErrorCreatePVC       = "ErrorCreatePersistentVolumeClaim"
	ErrorSSAlreadyExist  = "ErrorStatefulSetAlreadyExist"
	ErrorCreateService   = "ErrorCreateService"
	ErrorCreateRuntime   = "ErrorCreateRuntime"
	ErrorDeleteRuntime   = "ErrorDeleteRuntime"
	ErrorDeleteService   = "ErrorDeleteService"
	ErrorDeletePVC       = "ErrorDeletePersistentVolumeClaim"
	ErrorDeletePV        = "ErrorDeletePersistentVolume"
)

const ResourceStorage = "10Pi"

const (
	PaddleLabel = "paddlepaddle.org"
	PaddleOperatorLabel = "paddle-operator"
)

const (
	IndexerKeyEvent = "eventIndexerKey"
)

const (
	RuntimeContainerName = "runtime"
	RuntimeCacheMountPath = "/cache"
	RuntimeDateMountPath = "/mnt"
)

const (
	RuntimeServicePort = 7716
	RuntimeServiceName = "service"
)

const (
	PathUploadPrefix  = "/upload"
	PathServerRoot    = "/runtime"
	PathCacheStatus   = "/cacheStatus"
	PathSyncResult    = "/syncResult"
	PathClearResult   = "/clearResult"
	PathRmrResult     = "/rmrResult"
	PathWarmupResult  = "/warmupResult"
	PathSyncOptions   = "/syncOptions"
	PathClearOptions  = "/clearOptions"
	PathRmrOptions    = "/rmrOptions"
	PathWarmupOptions = "/warmupOptions"
	FilePathCacheInfo = "/cacheInfo.json"
)

const (
	JobStatusRunning JobStatus = "running"
	JobStatusSuccess JobStatus = "success"
	JobStatusFail    JobStatus = "fail"
)