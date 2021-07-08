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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SampleSetPhase indicates whether the loading is behaving
type SampleSetPhase string

// MediumType store medium type
type MediumType string

type RuntimeRole string

// CacheStateName is the name identifying various cacheStateName in a CacheStateNameList.
type CacheStateName string

// CacheStateList is a set of (resource name, quantity) pairs.
type CacheStateList map[CacheStateName]string

type SecretKeySelector struct {
	// The name of required secret
	// +required
	Name string `json:"name,omitempty"`

	// The required key in the secret
	// +optional
	Key string `json:"key,omitempty"`
}

type EncryptOptionSource struct {
	// The encryptInfo obtained from secret
	// +optional
	SecretKeyRef SecretKeySelector `json:"secretKeyRef,omitempty"`
}

type EncryptOption struct {
	// The name of encryptOption
	// +required
	Name string `json:"name,omitempty"`

	// The valueFrom of encryptOption
	// +optional
	ValueFrom EncryptOptionSource `json:"valueFrom,omitempty"`
}

// Mount describes a mounting. <br>
type Mount struct {
	// MountPoint is the mount point of source.
	// +kubebuilder:validation:MinLength=10
	// +required
	MountPoint string `json:"mountPoint,omitempty"`

	// The Mount Options. <br>
	// Refer to <a href="https://docs.alluxio.io/os/user/stable/en/reference/Properties-List.html">Mount Options</a>.  <br>
	// The option has Prefix 'fs.' And you can Learn more from
	// <a href="https://docs.alluxio.io/os/user/stable/en/ufs/S3.html">The Storage Integrations</a>
	// +optional
	Options map[string]string `json:"options,omitempty"`

	// The name of mount
	// +kubebuilder:validation:MinLength=0
	// +required
	Name string `json:"name,omitempty"`

	// The path of mount, if not set will be /{Name}
	// +optional
	Path string `json:"path,omitempty"`

	// Optional: Defaults to false (read-write).
	// +optional
	ReadOnly bool `json:"readOnly,omitempty"`

	// Optional: Defaults to false (shared).
	// +optional
	Shared bool `json:"shared,omitempty"`

	// The secret information
	// +optional
	EncryptOptions []EncryptOption `json:"encryptOptions,omitempty"`
}

// CacheableNodeAffinity defines constraints that limit what nodes this dataset can be cached to.
type CacheableNodeAffinity struct {
	// Required specifies hard node constraints that must be met.
	Required *corev1.NodeSelector `json:"required,omitempty"`
}

// Runtime describes a runtime to be used to support dataset
type Runtime struct {

	// Name of the runtime object
	Name string `json:"name,omitempty"`

	// Namespace of the runtime object
	Namespace string `json:"namespace,omitempty"`

	// Category the runtime object belongs to (e.g. Accelerate)
	//Category common.Category `json:"category,omitempty"`

	// Runtime object's type (e.g. Alluxio)
	Type string `json:"type,omitempty"`
}

// Level describes configurations a tier needs. <br>
// Refer to <a href="https://docs.alluxio.io/os/user/stable/en/core-services/Caching.html#configuring-tiered-storage">Configuring Tiered Storage</a> for more info
type Level struct {
	// Alias string `json:"alias,omitempty"`

	// Medium Type of the tier. One of the three types: `MEM`, `SSD`, `HDD`
	// +kubebuilder:validation:Enum=MEM;SSD;HDD
	// +required
	MediumType MediumType `json:"mediumtype"`

	// File paths to be used for the tier. Multiple paths are supported.
	// Multiple paths should be separated with comma. For example: "/mnt/cache1,/mnt/cache2".
	// +kubebuilder:validation:MinLength=1
	// +required
	Path string `json:"path,omitempty"`

	// Quota for the whole tier. (e.g. 100Gi)
	// Please note that if there're multiple paths used for this tierstore,
	// the quota will be equally divided into these paths. If you'd like to
	// set quota for each, path, see QuotaList for more information.
	// +optional
	Quota *resource.Quantity `json:"quota,omitempty"`

	// QuotaList are quotas used to set quota on multiple paths. Quotas should be separated with comma.
	// Quotas in this list will be set to paths with the same order in Path.
	// For example, with Path defined with "/mnt/cache1,/mnt/cache2" and QuotaList set to "100Gi, 50Gi",
	// then we get 100GiB cache storage under "/mnt/cache1" and 50GiB under "/mnt/cache2".
	// Also note that num of quotas must be consistent with the num of paths defined in Path.
	// +optional
	// +kubebuilder:validation:Pattern:="^(\\+|-)?(([0-9]+(\\.[0-9]*)?)|(\\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\\+|-)?(([0-9]+(\\.[0-9]*)?)|(\\.[0-9]+))))?,((\\+|-)?(([0-9]+(\\.[0-9]*)?)|(\\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\\+|-)?(([0-9]+(\\.[0-9]*)?)|(\\.[0-9]+))))?)+$"
	QuotaList string `json:"quotaList,omitempty"`

	// StorageType common.CacheStoreType `json:"storageType,omitempty"`
	// float64 is not supported, https://github.com/kubernetes-sigs/controller-tools/issues/245

	// Ratio of high watermark of the tier (e.g. 0.9)
	High string `json:"high,omitempty"`

	// Ratio of low watermark of the tier (e.g. 0.7)
	Low string `json:"low,omitempty"`
}

// Tieredstore is a description of the tiered store
type Tieredstore struct {
	// configurations for multiple tiers
	Levels []Level `json:"levels,omitempty"`
}

// SampleSetSpec defines the desired state of SampleSet
type SampleSetSpec struct {
	// The Partitions of the SimpleSet, need to be specified
	Partitions int32 `json:"Partitions,omitempty"`

	// Mount Points to be mounted on JuiceFS.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:UniqueItems=false
	// +required
	Mounts []Mount `json:"mounts,omitempty"`

	// Runtimes for supporting dataset (e.g. AlluxioRuntime)
	// +optional
	Runtime *Runtime `json:"runtime,omitempty"`

	// Tiered storage used by Alluxio
	// +optional
	Tieredstore Tieredstore `json:"tieredstore,omitempty"`

	// NodeAffinity defines constraints that limit what nodes this dataset can be cached to.
	// This field influences the scheduling of pods that use the cached dataset.
	// +optional
	NodeAffinity *CacheableNodeAffinity `json:"nodeAffinity,omitempty"`

	// If specified, the pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// AccessModes contains all ways the volume backing the PVC can be mounted
	// +optional
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`
}

// SampleSetStatus defines the observed state of SampleSet
type SampleSetStatus struct {
	// Total in GB of dataset in the cluster
	TotalSize string `json:"TotalSize,omitempty"`

	// Dataset Phase. One of the four phases: `Pending`, `Bound`, `NotBound` and `Failed`
	Phase SampleSetPhase `json:"phase,omitempty"`

	// CacheStatus represents the total resources of the dataset.
	CacheStates CacheStateList `json:"cacheStates,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SampleSet is the Schema for the samplesets API
type SampleSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SampleSetSpec   `json:"spec,omitempty"`
	Status SampleSetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SampleSetList contains a list of SampleSet
type SampleSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SampleSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SampleSet{}, &SampleSetList{})
}
