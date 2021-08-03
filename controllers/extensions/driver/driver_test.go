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
	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
	"os"
	"strings"
	"testing"
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
	if err := d.DoClearJob(context.Background(), opt); err != nil {
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
