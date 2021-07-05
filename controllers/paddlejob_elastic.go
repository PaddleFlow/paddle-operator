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

package controllers

import (
	"context"
	"fmt"
	"os"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	pdv1 "github.com/paddleflow/paddle-operator/api/v1"
)

var etcdctl *clientv3.Client

func initClient() error {
	host := os.Getenv("PADDLE_ELASTIC_ETCD_SERVICE_HOST")
	port := os.Getenv("PADDLE_ELASTIC_ETCD_SERVICE_PORT")
	etcd, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{fmt.Sprintf("http://%s:%s", host, port)},
		DialTimeout: 2 * time.Second,
	})
	if err == nil {
		etcdctl = etcd
		return nil
	} else {
		return err
	}
}

func syncEtcd(ctx context.Context, path string, np string) (error, bool) {
	if etcdctl == nil {
		err := initClient()
		if err != nil {
			return err, false
		}
	}

	if resp, err := etcdctl.Get(ctx, path); err != nil {
		return err, false
	} else if len(resp.Kvs) != 1 || string(resp.Kvs[0].Value) == np {
		return nil, false
	}

	if _, err := etcdctl.Put(ctx, path, np); err != nil {
		return err, false
	} else {
		return nil, true
	}
}

func syncNP(pdj *pdv1.PaddleJob) (*string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if pdj.Status.Mode == pdv1.PaddleJobModeCollective {
		path := fmt.Sprintf("/paddle/%s-%s/np", pdj.Namespace, pdj.Name)
		np := fmt.Sprintf("%d", pdj.Spec.Worker.Replicas)
		if err, updated := syncEtcd(ctx, path, np); updated {
			return &np, err
		} else {
			return nil, err
		}
	}
	return nil, nil
}
