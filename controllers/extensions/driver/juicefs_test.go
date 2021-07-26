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

package driver

import (
	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"testing"
)

// TestGetMountOptions
func TestGetMountOptions(t *testing.T) {
	mountOptions := v1alpha1.MountOptions{
		JuiceFSMountOptions: &v1alpha1.JuiceFSMountOptions{
			OpenCache: 7200,
			CacheSize: 307200,
			AttrCache: 7200,
			EntryCache: 7200,
			DirEntryCache: 7200,
			Prefetch: 1,
			BufferSize: 1024,
			EnableXattr: true,
			WriteBack: true,
			FreeSpaceRatio: "0.2",
			CacheDir: "/dev/shm/imagenet",
		},
	}

	sampleSet := &v1alpha1.SampleSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "imagenet",
			Namespace: "paddle-system",
		},
		Spec: v1alpha1.SampleSetSpec{
			Partitions: 2,
			Source: &v1alpha1.Source{
				URI: "https://imagenet.bj.bcebos.com/juicefs",
			},
			SecretRef: &corev1.LocalObjectReference{
				Name: "imagenet",
			},
			CSI: &v1alpha1.CSI{
				Driver: JuiceFSDriver,
				MountOptions: mountOptions,
			},
			Cache: v1alpha1.Cache{
				Levels: []v1alpha1.CacheLevel{
					{
						MediumType: common.Memory,
						Path: "/dev/shm/imagenet-0:/dev/shm/imagenet-1",
						CacheSize: 150,
					},
					{
						MediumType: common.SSD,
						Path: "/data/imagenet",
						CacheSize: 150,
					},
				},
			},
			NodeAffinity: &corev1.VolumeNodeAffinity{
				Required: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{{
						MatchExpressions: []corev1.NodeSelectorRequirement{{
								Key: "sampleset",
								Operator: corev1.NodeSelectorOpIn,
								Values: []string{"true"},
							},
						},
					},
					},
				},
			},
		},
	}

	driver := NewJuiceFSDriver()
	options, err := driver.getMountOptions(sampleSet)
	if err != nil {
		t.Errorf("test getMountOptions error: %s", err.Error())
	}
	for _, option := range strings.Split(options, ",") {
		if strings.HasPrefix(option, "cache-dir") {
			cacheDir := strings.Split(option, "=")[1]
			if len(strings.Split(cacheDir, ":")) != 3 {
				t.Errorf("length of cache-dir should be 2: %s", cacheDir)
			}
		}
	}

	t.Log("mountOptions: ", options)
}

// TestGetVolumeInfo
func TestGetVolumeInfo(t *testing.T) {
	mountOptions := "dir-entry-cache=7200,buffer-size=1024,prefetch=1,cache-dir=/dev/shm/imagenet:/data/imagenet:/dev/ssd/imagenet,cache-size=1048576"
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "imagenet",
		},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				CSI: &corev1.CSIPersistentVolumeSource{
					VolumeAttributes: map[string]string{
						"mountOptions": mountOptions,
					},
				},
			},
		},
	}

	driver := NewJuiceFSDriver()
	volumes, volumeMounts, err := driver.getVolumeInfo(pv)
	if err != nil {
		t.Error(err)
	}
	if len(volumes) != 4 || len(volumeMounts) != 4 {
		t.Errorf("len of volumes or volumeMounts not right \n")
	}

}