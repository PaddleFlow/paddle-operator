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
	"os"
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
	if err := d.DoClearJob(opt); err != nil {
		t.Errorf("DoClearJob error: %s", err.Error())
	}
}

func TestBaseDriver_GetCacheStatus(t *testing.T) {

}
