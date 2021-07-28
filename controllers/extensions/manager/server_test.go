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

package manager

import (
	"bytes"
	"fmt"
	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
	"github.com/paddleflow/paddle-operator/controllers/extensions/utils"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/uuid"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestServer_Run(t *testing.T) {
	rootOpt := &common.RootCmdOptions{
		Driver: "juicefs",
		Development: true,
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Error(err)
	}
	//t.Log("home== ", home)
	rootPath := home + common.PathServerRoot
	//t.Log("rootPath== ", rootPath)
	svrOpt := &common.ServerOptions{
		Port: common.RuntimeServicePort,
		Path: rootPath,
	}

	testDoSync := func(body []byte) error {
		opt := &v1alpha1.SyncJobOptions{}
		err := json.Unmarshal(body, opt)
		if err != nil {
		return err
	}
		//fmt.Println("do sync option: ", opt)
		return nil
	}

	testDoClear := func(body []byte) error {
		opt := &v1alpha1.ClearJobOptions{}
		err := json.Unmarshal(body, opt)
		if err != nil {
		return err
	}
		//fmt.Println("do clear option: ", opt)
		time.Sleep(600 * time.Second)
		return nil
	}

	var syncOptions = v1alpha1.SyncJobOptions{
		Source: "bos://imagenet.bj.bcebos.com/imagenet",
		Destination: "bos://imagenet.bj.bcebos.com/test",
		JuiceFSSyncOptions: v1alpha1.JuiceFSSyncOptions{
			Start: "startKey",
			Worker: "imagenet-1",
			BWLimit: 8,
			NoHttps: true,
		},
	}

	var clearOptions = v1alpha1.ClearJobOptions{
		Paths: []string{"/dev/shm/imagenet-0", "/dev/shm/imagenet-1"},
	}

	server, err := NewServer(rootOpt, svrOpt)
	if err != nil {
		t.Error(err)
	}

	go func() {
		if err := server.Run(); err != nil {
			t.Error("server error: ", err.Error())
			return
		}
	}()

	uri := fmt.Sprintf("http://localhost:%d", common.RuntimeServicePort)

	// wait util server is running
	for {
		time.Sleep(10 * time.Second)
		resp, err := http.Head(uri + "/")
		if err != nil {
			t.Log("wait server up, error: ", err.Error())
			continue
		}
		if resp.StatusCode == http.StatusOK {
			break
		} else {
			return
		}
	}

	server.doers[common.PathSyncOptions] = testDoSync
	server.doers[common.PathClearOptions] = testDoClear
	//t.Log("change server's doers")

	// upload sync option
	syncBody, err := json.Marshal(syncOptions)
	if err != nil {
		t.Error(err)
	}
	syncOptUri := uri + common.PathUploadPrefix + common.PathSyncOptions
	syncOptFileName := fmt.Sprintf("%s", uuid.NewUUID())
	resp, err := utils.Post(syncOptUri, syncOptFileName, bytes.NewReader(syncBody))
	if err != nil {
		t.Error("post sync option error", uri, err.Error())
		return
	}
	if resp.StatusCode != http.StatusOK {
		t.Error("post sync option status not ok")
		return
	}
	time.Sleep(5 * time.Second)
	// get sync result
	syncResUri := uri + common.PathSyncResult
	resp, err = utils.Get(syncResUri, syncOptFileName)
	if err != nil {
		t.Error("get sync result file error: ", err.Error())
		return
	}
	if resp.StatusCode != http.StatusOK {
		t.Error("get sync result file error resp status: ", resp.StatusCode)
	}
	syncResBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error("read sync result resp body error: ", err.Error())
	}

	var result common.JobResult
	err = json.Unmarshal(syncResBody, &result)
	if err != nil {
		t.Error("unmarshal sync resp body error: ", err.Error())
	}
	//t.Log("get sync result: ", result)

	if result.Status != common.JobStatusSuccess {
		t.Error("sync job should be fail, status: ", result.Status)
	}

	// upload clear options
	clearBody, err := json.Marshal(clearOptions)
	if err != nil {
		t.Error(err)
	}
	clearOptUri := uri + common.PathUploadPrefix + common.PathClearOptions
	clearOptFile := fmt.Sprintf("%s", uuid.NewUUID())
	resp, err = utils.Post(clearOptUri, clearOptFile, bytes.NewReader(clearBody))
	if err != nil {
		t.Error("post clear option error", uri, err.Error())
		return
	}
	if resp.StatusCode != http.StatusOK {
		t.Error("get clear result file error resp status: ", resp.Status)
		return
	}
	time.Sleep(2 * time.Second)
	// get clear result
	clearResUri := uri + common.PathClearResult
	resp, err = utils.Get(clearResUri, clearOptFile)
	if err != nil {
		t.Error("get clear result file error: ", err.Error())
		return
	}
	if resp.StatusCode != http.StatusOK {
		t.Error("get clear result file error resp status: ", resp.StatusCode)
		return
	}
	clearResBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error("read clear result resp body error: ", err.Error())
		return
	}

	err = json.Unmarshal(clearResBody, &result)
	if err != nil {
		t.Error("unmarshal clear resp body error: ", err.Error())
		return
	}
	t.Log("get clear result: ", result)
	if result.Status != common.JobStatusRunning {
		t.Error("clear job should be running, status: ", result.Status)
	}

	// get cache info


	_ = os.RemoveAll(rootPath)
}
