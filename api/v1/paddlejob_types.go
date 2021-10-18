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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	KIND = "PaddleJob"
)

const (
	// LABEL KEYS
	ResourceName = "paddle-res-name"
	ResourceType = "paddle-res-type"
	// Annotation KEY
	ResourceAnnotation = "paddle-resource"
)

// PaddleJobMode defines the avaiable mode of a job
type PaddleJobMode string

const (
	PaddleJobModePS PaddleJobMode = "PS"

	PaddleJobModeCollective PaddleJobMode = "Collective"

	PaddleJobModeSingle PaddleJobMode = "Single"

	PaddleJobModeHeter PaddleJobMode = "Heter"
)

// PaddleJobPhase defines the phase of the job.
type PaddleJobPhase string

const (
	Starting    PaddleJobPhase = "Starting"
	Pending     PaddleJobPhase = "Pending"
	Scaling     PaddleJobPhase = "Scaling"
	Aborting    PaddleJobPhase = "Aborting"
	Aborted     PaddleJobPhase = "Aborted"
	Running     PaddleJobPhase = "Running"
	Restarting  PaddleJobPhase = "Restarting"
	Completing  PaddleJobPhase = "Completing"
	Completed   PaddleJobPhase = "Completed"
	Terminating PaddleJobPhase = "Terminating"
	Terminated  PaddleJobPhase = "Terminated"
	Failed      PaddleJobPhase = "Failed"
	Succeed     PaddleJobPhase = "Succeed"
	Unknown     PaddleJobPhase = "Unknown"
)

type CleanPodPolicy string

const (
	// CleanAlways policy will always clean pods
	CleanAlways CleanPodPolicy = "Always"
	// CleanNever policy will nerver clean pods
	CleanNever CleanPodPolicy = "Never"
	// CleanOnFailure policy will clean pods only on job failed
	CleanOnFailure CleanPodPolicy = "OnFailure"
	// CleanOnCompletion policy will clean pods only on job completed
	CleanOnCompletion CleanPodPolicy = "OnCompletion"
)

// ElasticStatus defines the status of elastic process
type ElasticStatus string

const (
	ElasticStatusNone  ElasticStatus = "NONE"
	ElasticStatusDoing ElasticStatus = "DOING"
	ElasticStatusDone  ElasticStatus = "DONE"
	ElasticStatusError ElasticStatus = "ERROR"
)

type Intranet string

const (
	PodIP       Intranet = "PodIP"
	Service     Intranet = "Service"
	HostNetwork Intranet = "Host"
)

// SchedulingPolicy embed schedule policy of volcano
type SchedulingPolicy struct {
	MinAvailable  *int32 `json:"minAvailable,omitempty"`
	Queue         string `json:"queue,omitempty"`
	PriorityClass string `json:"priorityClass,omitempty"`
	// pointer may cause deepcopy error
	// api/v1/zz_generated.deepcopy.go:230:8: cannot use new(map["k8s.io/api/core/v1".ResourceName]resource.Quantity) (type *map["k8s.io/api/core/v1".ResourceName]resource.Quantity) as type *"k8s.io/api/core/v1".ResourceList in assignment
	MinResources corev1.ResourceList `json:"minResources,omitempty"`
}

// PaddleJobSpec defines the desired state of PaddleJob
type PaddleJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// CleanPodPolicy defines whether to clean pod after job finished
	CleanPodPolicy CleanPodPolicy `json:"cleanPodPolicy,omitempty"`

	// SchedulingPolicy defines the policy related to scheduling, for volcano
	// +optional
	SchedulingPolicy *SchedulingPolicy `json:"schedulingPolicy,omitempty"`

	// Intranet defines the communication mode inter pods : PodIP, Service or Host
	Intranet Intranet `json:"intranet,omitempty"`

	// WithGloo indicate whether enable gloo, 0/1/2 for disable/enable for worker/enable for server
	WithGloo *int `json:"withGloo,omitempty"`

	// Elastic indicate the elastic level
	Elastic *int `json:"elastic,omitempty"`

	// Tasks defines the resources in list, the order determinate the running order
	Tasks []*ResourceSpec `json:"tasks,omitempty"`
}

type ResourceSpec struct {
	// Replicas replica
	Replicas int `json:"replicas"`

	// Requests set the minimal replicas of server to be run
	Requests *int `json:"requests,omitempty"`

	// Requests set the maximal replicas of server to be run, elastic is auto enbale if limits is set larger than 0
	Limits *int `json:"limits,omitempty"`

	// Template specifies the podspec of a server
	Template corev1.PodTemplateSpec `json:"template,omitempty"`

	// Name name the resource, validated by IsDNS1123Subdomain
	Name string `json:"name,omitempty"`
}

// PaddleJobStatus defines the observed state of PaddleJob
type PaddleJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The phase of PaddleJob.
	Phase PaddleJobPhase `json:"phase,omitempty"`

	// Mode indicates in which the PaddleJob run with : PS/Collective/Single
	// PS mode is enabled when ps is set
	// Single/Collective is enabled if ps is missing
	Mode PaddleJobMode `json:"mode,omitempty"`

	// ResourceStatues of tasks
	Tasks map[string]*ResourceStatus `json:"tasks,omitempty"`

	// Elastic
	Elastic *ElasticStatus `json:"elastic,omitempty"`

	// StartTime indicate when the job started
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime indicate when the job completed/failed
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	ObservedGeneration int `json:"observedGeneration,omitempty"`
}

type ResourceStatus struct {
	// Pending
	Pending int `json:"pending,omitempty"`
	// Starting
	Starting int `json:"starting,omitempty"`
	// Running
	Running int `json:"running,omitempty"`
	// Failed
	Failed int `json:"failed,omitempty"`
	// Success
	Succeeded int `json:"succeeded,omitempty"`
	// Unknown
	Unknown int `json:"unknown,omitempty"`
	// A list of pointer to pods
	Refs []corev1.ObjectReference `json:"refs,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:shortName=pdj
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Mode",type=string,JSONPath=`.status.mode`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// PaddleJob is the Schema for the paddlejobs API
type PaddleJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PaddleJobSpec   `json:"spec,omitempty"`
	Status PaddleJobStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PaddleJobList contains a list of PaddleJob
type PaddleJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PaddleJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PaddleJob{}, &PaddleJobList{})
}
