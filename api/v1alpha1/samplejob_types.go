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

package v1alpha1

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SampleJobType string

type SampleJobPhase string

// JuiceFSMountOptions describes the JuiceFS mount options which user can set
// All the mount options is list in https://github.com/juicedata/juicefs/blob/main/docs/en/command_reference.md
type JuiceFSMountOptions struct {
	// address to export metrics (default: "127.0.0.1:9567")
	// +optional
	Metrics string `json:"metrics,omitempty"`
	// attributes cache timeout in seconds (default: 1)
	// +optional
	AttrCache int `json:"attr-cache,omitempty"`
	// file entry cache timeout in seconds (default: 1)
	// +optional
	EntryCache int `json:"entry-cache,omitempty"`
	// dir entry cache timeout in seconds (default: 1)
	// +optional
	DirEntryCache int `json:"dir-entry-cache,omitempty"`
	// enable extended attributes (xattr) (default: false)
	// +optional
	EnableXattr bool `json:"enable-xattr,omitempty"`
	// the max number of seconds to download an object (default: 60)
	// +optional
	GetTimeout int `json:"get-timeout,omitempty"`
	// the max number of seconds to upload an object (default: 60)
	// +optional
	PutTimeout int `json:"put-timeout,omitempty"`
	// number of retries after network failure (default: 30)
	// +optional
	IoRetries int `json:"io-retries,omitempty"`
	// number of connections to upload (default: 20)
	// +optional
	MaxUploads int `json:"max-uploads,omitempty"`
	// total read/write buffering in MB (default: 300)
	// +optional
	BufferSize int `json:"buffer-size,omitempty"`
	// prefetch N blocks in parallel (default: 1)
	// +optional
	Prefetch int `json:"prefetch,omitempty"`
	// upload objects in background (default: false)
	// +optional
	WriteBack bool `json:"writeback,omitempty"`
	// directory paths of local cache, use colon to separate multiple paths
	// +optional
	CacheDir string `json:"cache-dir,omitempty"`
	// size of cached objects in MiB (default: 1024)
	// +optional
	CacheSize int `json:"cache-size,omitempty"`
	// min free space (ratio) (default: 0.1)
	// float64 is not supported https://github.com/kubernetes-sigs/controller-tools/issues/245
	// +optional
	FreeSpaceRatio string `json:"free-space-ratio,omitempty"`
	// cache only random/small read (default: false)
	// +optional
	CachePartialOnly bool `json:"cache-partial-only,omitempty"`
	// open files cache timeout in seconds (0 means disable this feature) (default: 0)
	// +optional
	OpenCache int `json:"open-cache,omitempty"`
	// mount a sub-directory as root
	// +optional
	SubDir string `json:"sub-dir,omitempty"`
}

// JuiceFSSyncOptions describes the JuiceFS sync options which user can set by SampleSet
type JuiceFSSyncOptions struct {
	// the first KEY to sync
	// +optional
	Start string `json:"start,omitempty"`
	// the last KEY to sync
	// +optional
	End string `json:"end,omitempty"`
	// number of concurrent threads (default: 10)
	// +optional
	Threads int `json:"threads,omitempty"`
	// HTTP PORT to listen to (default: 6070)
	// +optional
	HttpPort int `json:"http-port,omitempty"`
	// update existing file if the source is newer (default: false)
	// +optional
	Update bool `json:"update,omitempty"`
	// always update existing file (default: false)
	// +optional
	ForceUpdate bool `json:"force-update,omitempty"`
	// preserve permissions (default: false)
	// +optional
	Perms bool `json:"perms,omitempty"`
	// Sync directories or holders (default: false)
	// +optional
	Dirs bool `json:"dirs,omitempty"`
	// Don't copy file (default: false)
	// +optional
	Dry bool `json:"dry,omitempty"`
	// delete objects from source after synced (default: false)
	// +optional
	DeleteSrc bool `json:"delete-src,omitempty"`
	// delete extraneous objects from destination (default: false)
	// +optional
	DeleteDst bool `json:"delete-dst,omitempty"`
	// exclude keys containing PATTERN (POSIX regular expressions)
	// +optional
	Exclude string `json:"exclude,omitempty"`
	// only include keys containing PATTERN (POSIX regular expressions)
	// +optional
	Include string `json:"include,omitempty"`
	// manager address
	// +optional
	Manager string `json:"manager,omitempty"`
	// hosts (seperated by comma) to launch worker
	// +optional
	Worker string `json:"worker,omitempty"`
	// limit bandwidth in Mbps (0 means unlimited) (default: 0)
	// +optional
	BWLimit int `json:"bwlimit,omitempty"`
	// do not use HTTPS (default: false)
	NoHttps bool `json:"no-https,omitempty"`
}

// JuiceFSWarmupOptions describes the JuiceFS warmup options which user can set by SampleSet
type JuiceFSWarmupOptions struct {
	// the path of file that containing a list of data file paths
	// +optional
	File string `json:"file,omitempty"`
	// number of concurrent workers (default: 50)
	// +optional
	Threads int `json:"threads,omitempty"`
}

// SyncJobOptions the options for sync data to cache engine
type SyncJobOptions struct {
	// data source that need sync to cache engine, the format of it should be
	// [NAME://]BUCKET[.ENDPOINT][/PREFIX]
	// +optional
	Source string `json:"source,omitempty"`
	// the relative path in mount volume for data sync to, eg: /train
	// +option
	Destination string `json:"destination,omitempty"`
	// JuiceFS sync command options
	// +optional
	JuiceFSSyncOptions `json:",inline"`
}

type SampleStrategy struct {
	// +kubebuilder:validation:Enum=random;sequence
	// +kubebuilder:validation:Required
	// +kubebuilder:default=random
	Name string `json:"strategyName,omitempty"`
}

// WarmupJobOptions the options for warmup date to local host
type WarmupJobOptions struct {
	// A list of paths need to build cache
	// +kubebuilder:validation:MinItems=1
	// +optional
	Paths []string `json:"paths,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// +optional
	Partitions int32 `json:"partitions,omitempty"`
	// +optional
	Strategy SampleStrategy `json:",inline"`
	// JuiceFS warmup command options
	// +optional
	JuiceFSWarmupOptions `json:",inline"`
}

// RmrJobOptions the options for remove data from cache engine
type RmrJobOptions struct {
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:Required
	Paths []string `json:"paths,omitempty"`
}

// ClearJobOptions the options for clear cache from local host
type ClearJobOptions struct {
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:Required
	Paths []string `json:"paths,omitempty"`
}

type JobOptions struct {
	//
	// +optional
	SyncOptions *SyncJobOptions `json:"syncOptions,omitempty"`
	//
	// +optional
	WarmupOptions *WarmupJobOptions `json:"warmupOptions,omitempty"`
	//
	// +optional
	RmrOptions *RmrJobOptions `json:"rmrOptions,omitempty"`
	//
	// +optional
	ClearOptions *ClearJobOptions `json:"clearOptions,omitempty"`
}

// SampleJobSpec defines the desired state of SampleJob
type SampleJobSpec struct {
	// Job Type of SampleJob. One of the three types: `sync`, `warmup`, `rmr`, `clear`
	// +kubebuilder:validation:Enum=sync;warmup;rmr;clear
	// +kubebuilder:validation:Required
	Type SampleJobType `json:"type,omitempty"`
	// +kubebuilder:validation:MinLength=0
	// +kubebuilder:validation:Required
	SampleSet string `json:"sampleset,omitempty"`
	/// The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	// +optional
	Schedule string `json:"schedule,omitempty"`
	// +optional
	JobOptions `json:",inline,omitempty"`
}

// SampleJobStatus defines the observed state of SampleJob
type SampleJobStatus struct {
	//
	Phase SampleJobPhase `json:"phase,omitempty"`
	// The latest available observations of an object's current state. When a Job
	// fails, one of the conditions will have type "Failed" and status true. When
	// a Job is suspended, one of the conditions will have type "Suspended" and
	// status true; when the Job is resumed, the status of this condition will
	// become false. When a Job is completed, one of the conditions will have
	// type "Complete" and status true.
	// More info: https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=atomic
	Conditions []batchv1.JobCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
	// A list of pointers to currently running jobs.
	// +optional
	// +listType=atomic
	Active []corev1.ObjectReference `json:"active,omitempty"`
	// Information when was the last time the job was successfully scheduled.
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`
	// Information when was the last time the job successfully completed.
	// +optional
	LastSuccessfulTime *metav1.Time `json:"lastSuccessfulTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SampleJob is the Schema for the samplejobs API
type SampleJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SampleJobSpec   `json:"spec,omitempty"`
	Status SampleJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SampleJobList contains a list of SampleJob
type SampleJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SampleJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SampleJob{}, &SampleJobList{})
}
