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
	"github.com/fsnotify/fsnotify"
	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
	"github.com/paddleflow/paddle-operator/controllers/extensions/utils"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/uuid"
	"log"
	"os"
	"strconv"
	"testing"
	"time"
)

type TestServer struct {
	Server
}

func (s *TestServer) doSync(body []byte) error {
	opt := &v1alpha1.SyncJobOptions{}
	err := json.Unmarshal(body, opt)
	if err != nil {
		return err
	}
	fmt.Printf("do sync option: %+v\n", opt)
	return nil
}

func (s *Server) doClear(body []byte) error {
	opt := &v1alpha1.ClearJobOptions{}
	err := json.Unmarshal(body, opt)
	if err != nil {
		return err
	}
	fmt.Printf("do clear option: %+v\n", opt)
	return nil
}

func TestServer_Run(t *testing.T) {
	rootOpt := &common.RootCmdOptions{
		Driver: "juicefs",
		Development: true,
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Error(err)
	}
	svrOpt := &common.ServerOptions{
		Port: common.RuntimeServicePort,
		Path: home,
	}

	var testSyncOptions = v1alpha1.SyncJobOptions{
		Source: "bos://imagenet.bj.bcebos.com/imagenet",
		Destination: "bos://imagenet.bj.bcebos.com/test",
		JuiceFSSyncOptions: v1alpha1.JuiceFSSyncOptions{
			Start: "startKey",
			Worker: "imagenet-1",
			BWLimit: 8,
			NoHttps: true,
		},
	}

	var testClearOptions = v1alpha1.ClearJobOptions{

	}

	server, err := NewServer(rootOpt, svrOpt)
	if err != nil {
		t.Error(err)
	}

	ioutil.

}


func TestUUID (t *testing.T) {

	uid := uuid.NewUUID()

	t.Log(fmt.Sprintf("uuid %s", uid))
}

func TestMkdir(t *testing.T) {
	path := "/Users/chenenquan/Desktop/test/test"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		e := os.Mkdir(path, os.ModePerm)
		if e != nil {
			t.Fatal(e)
		}
	}
}

func TestPostGet(t *testing.T) {
	options := v1alpha1.SyncJobOptions{
		Source: "test",
		Destination: "test",
		JuiceFSSyncOptions: v1alpha1.JuiceFSSyncOptions{
			Start: "test",
		},
	}

	body, err := json.Marshal(options)
	if err != nil {
		t.Error(err)
	}

	resp, err := utils.Post("http://localhost:7716/upload/syncOptions", "test.json", bytes.NewReader(body))
	if err != nil {
		t.Error(err)
	}
	t.Log(resp.Status)

	resp, err = utils.Get("http://localhost:7716/syncOptions", "test.json")
	if err != nil {
		t.Error(err)
	}
	content, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Error(err)
	}
	opt := &v1alpha1.SyncJobOptions{}
	if err := json.Unmarshal(content, opt); err != nil {
		t.Error(err)
	}
	t.Log("rootOpt===", opt)
}

func TestWatcher(t *testing.T) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	handleEvent := func(event fsnotify.Event) {
		time.Sleep(30 * time.Second)
		fmt.Println("handle event", event)
	}

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				handleEvent(event)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("hande error: ", err)
			}
		}
	}()

	err = watcher.Add(common.PathServerRoot + common.PathSyncOptions)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		filename := common.PathServerRoot + common.PathSyncOptions + "/" + strconv.Itoa(i) + ".txt"
		err := ioutil.WriteFile(filename, []byte(strconv.Itoa(i)), os.ModePerm)
		if err != nil {
			t.Error(err)
		}
	}

	<-done
}