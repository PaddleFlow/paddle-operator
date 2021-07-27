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
	"testing"
)


func TestWatchAndDo() {


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
	t.Log("opt===", opt)
}

func TestWatcher(t *testing.T) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(common.PathServerRoot + common.PathSyncOptions)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}