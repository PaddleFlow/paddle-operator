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
	"context"
	"os"
	"strings"
	"testing"

	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestBaseDriver_DoClearJob(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Error(err)
	}
	var paths []string
	path1 := home + "/test_DoClearJob"
	err = os.MkdirAll(path1, os.ModePerm)
	if err != nil {
		t.Errorf("make dir %s error: %s", path1, err.Error())
	}
	paths = append(paths, path1)
	path2 := home + "/test_DoClearJob" + "/test"
	err = os.MkdirAll(path2, os.ModePerm)
	if err != nil {
		t.Errorf("make dir %s error: %s", path2, err.Error())
	}
	paths = append(paths, path2)

	opt := &v1alpha1.ClearJobOptions{Paths: paths}
	d, _ := GetDriver("juicefs")
	if err := d.DoClearJob(context.Background(), opt, nil); err != nil {
		t.Errorf("DoClearJob error: %s", err.Error())
	}
}

func TestBaseDriver_GetCacheStatus(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Error(err)
	}

	svrOpt := &common.ServerOptions{
		ServerPort: common.RuntimeServicePort,
		ServerDir: home + common.PathServerRoot,
		CacheDirs: []string{
			"",
			"",
		},
		DataDir: "",
		Interval: 30,
	}

	status := &v1alpha1.CacheStatus{}
	d, _ := GetDriver("juicefs")
	err = d.CreateCacheStatus(svrOpt, status)
	if status.TotalSize == "" {
		t.Error(err)
	}
	t.Log(status)

	txt := `Filesystem      Size  Used Avail Use% Mounted on
/dev/vda1        40G   26G   13G  68% /
tmpfs           3.9G   64M  3.9G   2% /dev/shm
total            44G   26G   16G  62% -
`
	lines := strings.Split(strings.TrimSpace(txt), "\n")
	if len(lines) == 0 {
		t.Error(lines)
	}
	total := strings.TrimSpace(lines[len(lines)-1])
	if !strings.Contains(total, "total") {
		t.Error("not contain total")
	}
	totalSlice := strings.FieldsFunc(total, func(r rune) bool { return r == ' ' || r == '\t' })
	if len(totalSlice) != 6 {
		t.Error("deal with output error")
	}
}

func TestBaseDriver_CreateClearJobOptions(t *testing.T) {
	volumes := []v1.Volume{
		{
			Name: "dev-shm-imagenet-0",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "/dev/shm/imagenet-0",
				},
			},
		},
		{
			Name: "dev-shm-imagenet-1",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "/dev/shm/imagenet-1",
				},
			},
		},
	}
	volumeMounts := []v1.VolumeMount{
		{
			Name: "dev-shm-imagenet-0",
			MountPath: "/cache/dev-shm-imagenet-0",
		},
		{
			Name: "dev-shm-imagenet-1",
			MountPath: "/cache/dev-shm-imagenet-1",
		},
	}
	statefulSet := &appv1.StatefulSet{
		Spec: appv1.StatefulSetSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Volumes: volumes,
					Containers: []v1.Container{
						{
							VolumeMounts: volumeMounts,
						},
					},
				},

			},
		},
	}

	sampleJob := &v1alpha1.SampleJob{
		Spec: v1alpha1.SampleJobSpec{
			JobOptions: v1alpha1.JobOptions{
				ClearOptions: &v1alpha1.ClearJobOptions{
					Paths: []string{
						"/dev/shm/imagenet-0/train",
						"/cache/dev-shm-imagenet-1",
						"/dev/shm/imagenet-0/",
					},
				},
			},
		},
	}
	ctx := &common.RequestContext{StatefulSet: statefulSet, SampleJob: sampleJob}

	d, _ := GetDriver("juicefs")
	opts := &v1alpha1.ClearJobOptions{}
	err := d.CreateClearJobOptions(opts, ctx)
	if err != nil {
		t.Error("create clear job options error: ", err.Error())
		return
	}
	if len(opts.Paths) != 3 {
		t.Error("length of paths should be 3")
		return
	}
	if opts.Paths[0] != "/cache/dev-shm-imagenet-0/train" {
		t.Error("create clear job options not as expected")
	}
	if opts.Paths[1] != "/cache/dev-shm-imagenet-1/*" {
		t.Error("create clear job options not as expected")
	}
	if opts.Paths[2] != "/cache/dev-shm-imagenet-0/*" {
		t.Error("create clear job options not as expected")
	}
}

func TestBaseDriver_CreateRmrJobOptions(t *testing.T) {
	sampleJob := &v1alpha1.SampleJob{
		Spec: v1alpha1.SampleJobSpec{
			JobOptions: v1alpha1.JobOptions{
				RmrOptions: &v1alpha1.RmrJobOptions{
					Paths: []string{
						"/train",
						"val/n123323",
					},
				},
			},
		},
	}
	request := &ctrl.Request{NamespacedName: types.NamespacedName{Name: "imagenet"}}
	ctx := &common.RequestContext{SampleJob: sampleJob, Req: request}
	d, _ := GetDriver("juicefs")
	opts := &v1alpha1.RmrJobOptions{}
	err := d.CreateRmrJobOptions(opts, ctx)
	if err != nil {
		t.Error("create rmr job option error: ", err.Error())
		return
	}
	if len(opts.Paths) != 2 {
		t.Error("length of paths should be 2")
		return
	}
	if opts.Paths[0] != "/mnt/imagenet/train" {
		t.Error("create rmr job options not as expected")
	}
	if opts.Paths[1] != "/mnt/imagenet/val/n123323" {
		t.Error("create rmr job options not as expected")
	}
}
