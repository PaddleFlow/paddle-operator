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
	"fmt"
	"github.com/go-logr/logr"
	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
	zapOpt "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/json"
	"os"
	"strings"

	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/fsnotify/fsnotify"

	"github.com/paddleflow/paddle-operator/controllers/extensions/driver"
)

type Server struct {
	driver.Driver
	Log logr.Logger

	watcher *fsnotify.Watcher
	opt *common.RootCmdOptions
	doers map[string]func([]byte)error
}

func NewServer(opt *common.RootCmdOptions) (*Server, error) {
	driverName := v1alpha1.DriverName(opt.Driver)
	csiDriver, err :=  driver.GetDriver(driverName)
	if err != nil {
		return nil, err
	}

	// configure zap log and create a logger
	zapLog := zap.New(func(o *zap.Options) {
		o.Development = opt.Development
	}, func(o *zap.Options) {
		o.ZapOpts = append(o.ZapOpts, zapOpt.AddCaller())
	},
	func(o *zap.Options) {
		if !opt.Development {
			encCfg := zapOpt.NewProductionEncoderConfig()
			encCfg.EncodeLevel = zapcore.CapitalLevelEncoder
			encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
			o.Encoder = zapcore.NewConsoleEncoder(encCfg)
		}
	})

	// create file system notify watcher and add dir to watch
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	server := &Server{
		opt: opt,
		Log: zapLog,
		watcher: watcher,
		Driver: csiDriver,
	}
	return server, nil
}


func (s *Server) Run() error {
	defer s.watcher.Close()

	if err := addStaticHandlers(
		common.PathCacheInfo,
		common.PathSyncResult,
		common.PathClearResult,
		common.PathSyncOptions,
		common.PathClearOptions,
		); err != nil {
		return err
	}
	addUploadHandlers(s,
		common.PathSyncOptions,
		common.PathUploadPrefix,
		)

	if err := addWatchDirs(
		s.watcher,
		common.PathSyncOptions,
		common.PathClearOptions,
	); err != nil {
		return err
	}

	// add job doer for watcher's event
	s.doers[common.PathSyncOptions] = s.doSync
	s.doers[common.PathClearOptions] = s.doClear

	go s.watchAndDo()

	s.Log.V(1).Info("===== run server ======")
	addr := fmt.Sprintf(":%d", common.RuntimeServicePort)
	return http.ListenAndServe(addr, nil)
}

func (s *Server) uploadHandleFunc(pattern string) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		s.Log.WithValues("route", pattern)
		if req.Method != http.MethodPost {
			err := fmt.Errorf("http method %s not support", req.Method)
			s.Log.Error(err, "error occur when upload sync options")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			s.Log.Error(err, "read request body error")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		opt := &v1alpha1.SyncJobOptions{}
		if err := json.Unmarshal(body, opt); err != nil {
			s.Log.Error(err, "json unmarshal request body error")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		fileName := req.Header.Get("filename")
		if fileName == "" {
			e := fmt.Errorf("handle upload %s fail", pattern)
			s.Log.Error(e, "can not get filename from request header")
		}

		dirPath := common.PathServerRoot + pattern
		filePath := dirPath + "/" + fileName
		err = ioutil.WriteFile(filePath, body, os.ModePerm)
		if err != nil {
			s.Log.Error(err, "write file error", "file", filePath)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		s.Log.V(1).Info("upload options success", "file", filePath)
		w.WriteHeader(http.StatusOK)
		return
	}
}

func (s *Server) watchAndDo() {
	for {
		select {
		case event, ok := <-s.watcher.Events:
			if !ok { return }
			s.handleEvent(event)
		case err, ok := <-s.watcher.Errors:
			if !ok { return }
			s.Log.Error(err, "watcher get error")
		}
	}
}

func (s *Server) handleEvent(event fsnotify.Event) {
	s.Log.V(1).Info("get event", "event", event.String())
	if event.Op != fsnotify.Create {
		s.Log.V(1).Info("event operation is create")
		return
	}

	switch extractPattern(event.Name) {
	case common.PathSyncOptions:
		s.do(common.PathSyncOptions, event.Name)
	case common.PathClearOptions:
		s.do(common.PathClearOptions, event.Name)
	default:
		err := fmt.Errorf()
		s.Log.Error(err, "")
	}
}

func (s *Server) do(pattern string, optionFile string) {
	result := common.JobResult{}
	// 1. 创建结果文件
	body, err := ioutil.ReadFile(optionFile)
	if err != nil {

	}

	doer, exist := s.doers[pattern]
	if !exist {

	}

	if err := doer(body); err != nil {

	}

	// 2. 执行相关命令
	// 3. 更新结果文件
}

func (s *Server) doSync(body []byte) error {
	opt := &v1alpha1.SyncJobOptions{}
	err := json.Unmarshal(body, opt)
	if err != nil {
		return err
	}
	return s.DoSyncJob(opt)
}

func (s *Server) doClear(body []byte) error {
	opt := &v1alpha1.ClearJobOptions{}
	err := json.Unmarshal(body, opt)
	if err != nil {
		return err
	}
	return s.DoClearJob(opt)
}

func addWatchDirs(watcher *fsnotify.Watcher, patterns... string) error {
	for _, pattern := range patterns {
		err := watcher.Add(common.PathServerRoot +pattern)
		if err != nil {
			return fmt.Errorf("add watcher dir %s error", pattern)
		}
	}
	return nil
}

func addStaticHandlers(patterns... string) error {
	// Add static file server
	http.Handle("/", http.FileServer(http.Dir(common.PathServerRoot)))

	for _, pattern := range patterns {
		path := common.PathServerRoot + pattern
		if _, err := os.Stat(path); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			e := os.MkdirAll(path, os.ModePerm)
			if e != nil {
				return e
			}
		}

		handler := http.FileServer(http.Dir(path))
		http.Handle(pattern, http.StripPrefix(pattern, handler))

	}
	return nil
}

func addUploadHandlers(s *Server, patterns... string) {
	for _, pattern := range patterns {
		uploadUrl := common.PathUploadPrefix + pattern
		http.HandleFunc(uploadUrl, s.uploadHandleFunc(pattern))
	}
}

func extractPattern(path string) string {
	if strings.Contains(path, common.PathSyncOptions) {
		return common.PathSyncOptions
	}
	if strings.Contains(path, common.PathClearOptions) {
		return common.PathClearOptions
	}
	return ""
}
