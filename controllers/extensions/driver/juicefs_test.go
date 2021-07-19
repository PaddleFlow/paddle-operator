package driver

import (
	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)


//func TestBase64(t *testing.T) {
//	s := "aHR0cHM6Ly96aG91dGktbWNwLWVkZ2UuZ3ouYmNlYm9zLmNvbS9pbWFnZW5ldC1qZnM="
//	decodeString, err := base64.StdEncoding.DecodeString(s)
//	if err != nil {
//		t.Errorf("==err: %s", err.Error())
//	}
//	t.Log("decodeString== ", string(decodeString))
//
//}

//func TestGetVolumeHandle(t *testing.T) {
//	secret := &corev1.Secret{
//		ObjectMeta: metav1.ObjectMeta{
//			Name: "imagenet",
//			Namespace: "paddle-system",
//		},
//		Data: map[string][]byte{
//
//		},
//
//		Type: corev1.SecretTypeOpaque,
//	}
//	driver := NewJuiceFSDriver()
//	volumeHandle, err := driver.getVolumeHandle(secret)
//	if err != nil {
//		t.Errorf("test getVolumeHandle error: %s", err.Error())
//	}
//	t.Log(volumeHandle)
//}


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
		},
	}

	sampleSet := &v1alpha1.SampleSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "imagenet",
			Namespace: "paddle-system",
		},
		Spec: v1alpha1.SampleSetSpec{
			Partitions: 2,
			Sources: []v1alpha1.Source{
				{
					URI:  "https://imagenet.bj.bcebos.com/juicefs",
					Name: "imagenet",
					Path: "/imagenet",
				},
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
						Path: "/dev/shm/imagenet",
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
	t.Log("mountOptions: ", options)
}