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
	ErrorSecretNotExists = "ErrorSecretNotExists"

	ErrorPVAlreadyExists = "ErrorPersistentVolumeAlreadyExists"

	ErrorCreatePV = "ErrorCreatePersistentVolume"

	ErrorPVCAlreadyExists = "ErrorPersistentVolumeClaimAlreadyExists"

	ErrorCreatePVC = "ErrorCreatePersistentVolumeClaim"



)

const (
	ResourceStorage = "10Pi"
)

const (
	PaddleLabel = "paddlepaddle.org"
	PaddleOperatorLabel = "paddle-operator"
)