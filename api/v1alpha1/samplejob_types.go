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
	"github.com/paddleflow/paddle-operator/controllers/extensions/executor"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SampleJobType string

type SampleJobPhase string

type JobOptions struct {
	//
	// +optional
	SyncOptions *executor.SyncJobOptions `json:"syncOptions,omitempty"`
	//
	// +optional
	WarmupOptions *executor.WarmupJobOptions `json:"warmupOptions,omitempty"`
	//
	// +optional
	RmrOptions *executor.RmrJobOptions `json:"rmrOptions,omitempty"`
	//
	// +optional
	ClearOptions *executor.ClearJobOptions `json:"clearOptions,omitempty"`
}



// SampleJobSpec defines the desired state of SampleJob
type SampleJobSpec struct {
	// Job Type of SampleJob. One of the three types: `sync`, `warmup`, `rmr`, `clear`
	// +kubebuilder:validation:Enum=sync;warmup;rmr;clear
	// +required
	Type SampleJobType `json:"type,omitempty"`
	// +kubebuilder:validation:MinLength=0
	// +required
	SampleSet string `json:"sampleset,omitempty"`
	/// The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	// +optional
	Schedule string `json:"schedule,omitempty"`
	//
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
