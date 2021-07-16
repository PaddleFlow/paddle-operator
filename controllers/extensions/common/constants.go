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
	// SampleSetPending Bound to runtime, can't be deleted
	SampleSetPending v1alpha1.SampleSetPhase = "Pending"
	// SampleSetBound to dataset, can't be released
	SampleSetBound v1alpha1.SampleSetPhase = "Bound"
	// SampleSetReady can't be deleted
	SampleSetReady v1alpha1.SampleSetPhase = "Ready"
	// SampleSetPartialReady Not bound to runtime, can be deleted
	SampleSetPartialReady v1alpha1.SampleSetPhase = "PartialReady"
	// SampleSetFailed Not bound to runtime, can be deleted
	SampleSetFailed v1alpha1.SampleSetPhase = "Failed"
	SampleSetNone   v1alpha1.SampleSetPhase = ""
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
	ErrorSecretNotExists = "ErrorSecretNotExists"

	ErrorCreatePersistentVolume = "ErrorCreatePersistentVolume"


)

const (
	StorageClassName = "paddle-operator"
)

const (
	PaddleLabel = "paddlepaddle.org"
	PaddleOperatorLabel = "paddle-operator"
)