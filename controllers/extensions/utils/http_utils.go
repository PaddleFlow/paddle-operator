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

package utils

import (
	"fmt"
	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"io"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"

	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
)

var DefaultClient = &HttpClient{}

type HttpClient struct {
	http.Client
}

func (c *HttpClient) Post(url, filename string, body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("filename", filename)
	req.Header.Set("Content-Type", "application/json")
	return c.Do(req)
}

func Post(url, filename string, body io.Reader) (resp *http.Response, err error) {
	return DefaultClient.Post(url, filename, body)
}

func Get(url string, filename string) (resp *http.Response, err error) {
	return DefaultClient.Get(url + "/" + filename)
}

func GetBaseUri(runtimeName, serviceName string, index int) string {
	return fmt.Sprintf("http://%s-%d.%s:%d", runtimeName, index, serviceName, common.RuntimeServicePort)
}

func GetUploadUri(baseUri, uploadPath string) string {
	return baseUri + common.PathUploadPrefix + uploadPath
}

func GetResultUri(baseUri, resultPath string) string {
	return baseUri + resultPath
}

func GetCacheStatus(baseUri string, status *v1alpha1.CacheStatus) error {
	resultUri := GetResultUri(baseUri, common.PathCacheStatus)
	resp, err := Get(resultUri, common.FilePathCacheInfo)
	if err != nil {
		return fmt.Errorf("get uri %s, error: %s", resultUri, err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("resp status code not ok: %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read resp body error: %s", err.Error())
	}
	defer resp.Body.Close()

	err = json.Unmarshal(body, status)
	if err != nil {
		return fmt.Errorf("unmarshal resp body error: %s", err.Error())
	}
	return nil
}

func GetJobResult(result *common.JobResult, baseUri, resultPath, fileName string) error {
	resultUri := GetResultUri(baseUri, resultPath)
	resp, err := Get(resultUri, fileName)
	if err != nil {
		return fmt.Errorf("get uri %s, filename: %s, error: %s", resultUri, fileName, err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("resp status code not ok: %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read resp body error: %s", err.Error())
	}
	defer resp.Body.Close()

	err = json.Unmarshal(body, result)
	if err != nil {
		return fmt.Errorf("unmarshal resp body error: %s", err.Error())
	}
	return nil
}