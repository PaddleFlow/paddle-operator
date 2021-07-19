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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SampleSetPhase indicates whether the loading is behaving
type SampleSetPhase string

// MediumType store medium type
type MediumType string

// DriverName a
type DriverName string

// Source describes a mounting. <br>
type Source struct {
	// MountPoint is the mount point of source.
	// +kubebuilder:validation:MinLength=10
	// +required
	URI string `json:"uri,omitempty"`
	// The name of mount
	// +kubebuilder:validation:MinLength=0
	// +required
	Name string `json:"name,omitempty"`
	// The path of mount, if not set will be /{Name}
	// +optional
	Path string `json:"path,omitempty"`
}

// MountOptions aa
type MountOptions struct {
	// JuiceFSMountOptions
	// +optional
	JuiceFSMountOptions *JuiceFSMountOptions `json:"juiceFSMountOptions,omitempty"`
}

// CSI describes a runtime to be used to support dataset
type CSI struct {
	// Name of cache runtime driver, now only support juicefs.
	// +kubebuilder:validation:Enum=juicefs
	// +kubebuilder:default=juicefs
	// +required
	Driver DriverName `json:"driver,omitempty"`
	// Namespace of the runtime object
	// +optional
	MountOptions `json:",inline,omitempty"`
}

// CacheLevel describes configurations a tier needs.
type CacheLevel struct {
	// Medium Type of the tier. One of the three types: `MEM`, `SSD`, `HDD`
	// +kubebuilder:validation:Enum=MEM;SSD;HDD
	// +required
	MediumType MediumType `json:"mediumType,omitempty"`
	// directory paths of local cache, use colon to separate multiple paths
	// For example: "/dev/shm/cache1:/dev/ssd/cache2:/mnt/cache3".
	// +kubebuilder:validation:MinLength=1
	// +required
	Path string `json:"path,omitempty"`
	// CacheSize size of cached objects in MiB
	// If multiple paths used for this, the cache size is total amount of cache objects in all paths
	// +optional
	CacheSize int `json:"cacheSize,omitempty"`
}

// Cache used to describe how cache data store
type Cache struct {
	// configurations for multiple storage tier
	Levels []CacheLevel `json:"levels,omitempty"`
}

// SampleSetSpec defines the desired state of SampleSet
type SampleSetSpec struct {
	// The Partitions of the SimpleSet, need to be specified
	Partitions int32 `json:"partitions,omitempty"`
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:UniqueItems=false
	// +required
	Sources []Source `json:"sources,omitempty"`
	// SecretRef is reference to the authentication secret for source storage and cache engine.
	// +required
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`
	// Runtimes for supporting dataset (e.g. AlluxioRuntime)
	// +optional
	CSI *CSI `json:"csi,omitempty"`
	// cache options used by cache runtime engine
	// +optional
	Cache Cache `json:"cache,omitempty"`
	// NodeAffinity defines constraints that limit what nodes this SampleSet can be cached to.
	// This field influences the scheduling of pods that use the cached dataset.
	// +optional
	NodeAffinity *corev1.VolumeNodeAffinity `json:"nodeAffinity,omitempty"`
	// If specified, the pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// SampleSetStatus defines the observed state of SampleSet
type SampleSetStatus struct {
	// TotalSize in GB of the SampleSet in the cluster
	TotalSize string `json:"totalSize,omitempty"`
	// TotalFiles represents the file numbers of the SampleSet
	TotalFiles string `json:"totalFiles,omitempty"`
	// CachedSize
	CachedSize string `json:"cachedSize,omitempty"`
	// CachedSize
	CacheCapacity string `json:"cacheCapacity,omitempty"`
	// CachedSize
	CachedPercentage string `json:"cachedPercentage,omitempty"`
	// Dataset Phase. One of the four phases: `Pending`, `Bound`, `NotBound` and `Failed`
	Phase SampleSetPhase `json:"phase,omitempty"`
	// SampleJobRef specifies the running SampleJob that manager this SampleSet
	SampleJobRef string `json:"sampleJobRef,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="TOTAL SIZE",type="string",JSONPath=`.status.totalSize`
//+kubebuilder:printcolumn:name="CACHED SIZE",type="string",JSONPath=`.status.cachedSize`
//+kubebuilder:printcolumn:name="CACHE CAPACITY",type="string",JSONPath=`.status.cacheCapacity`
//+kubebuilder:printcolumn:name="CACHED PERCENTAGE",type="string",JSONPath=`.status.cachedPercentage`
//+kubebuilder:printcolumn:name="PHASE",type="string",JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="AGE",type="date",JSONPath=`.metadata.creationTimestamp`

// SampleSet is the Schema for the SampleSets API
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
