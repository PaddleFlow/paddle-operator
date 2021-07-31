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
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"os/exec"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
	"strings"
	"time"
)

// HasDeletionTimestamp method to check if an object need to delete.
func HasDeletionTimestamp(obj *metav1.ObjectMeta) bool {
	return !obj.GetDeletionTimestamp().IsZero()
}

// HasFinalizer check
func HasFinalizer(obj *metav1.ObjectMeta, finalizer string) bool {
	return ContainsString(obj.GetFinalizers(), finalizer)
}

func RemoveFinalizer(obj *metav1.ObjectMeta, finalizer string) {
	finalizers := RemoveString(obj.Finalizers, finalizer)
	obj.Finalizers = finalizers
}

func NoRequeue() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func RequeueWithError(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}

func RequeueImmediately() (ctrl.Result, error) {
	return ctrl.Result{Requeue: true}, nil
}

func RequeueAfter(requeueAfter time.Duration) (ctrl.Result, error) {
	return ctrl.Result{RequeueAfter: requeueAfter}, nil

}

// ContainsString Determine whether the string array contains a specific string,
// return true if contains the string and return false if not.
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func NoZeroOptionToMap(optionMap map[string]reflect.Value, i interface{}) {
	elem := reflect.ValueOf(i).Elem()
	for i := 0; i < elem.NumField(); i++ {
		value := elem.Field(i)
		if value.IsZero() {
			continue
		}
		field := elem.Type().Field(i)
		tag := field.Tag.Get("json")
		option := strings.Split(tag, ",")[0]
		optionMap[option] = value
	}
}

func NoZeroOptionToArgs(options interface{}) []string {
	var args []string

	elem := reflect.ValueOf(options).Elem()
	for i := 0; i < elem.NumField(); i++ {
		v := elem.Field(i)
		if v.IsZero() {
			continue
		}
		field := elem.Type().Field(i)
		tag := field.Tag.Get("json")
		opt := strings.Split(tag, ",")[0]
		switch v.Kind() {
		case reflect.Bool:
			args = append(args, fmt.Sprintf("--%s", opt))
		case reflect.String:
			args = append(args, fmt.Sprintf(`--%s="%s"`, opt, v))
		case reflect.Slice:  // []string
			for j := 0; j < v.Len(); j++ {
				args = append(args, fmt.Sprintf(`--%s="%+v"`, opt, v.Index(j)))
			}
		default:  // int int64 ...
			args = append(args, fmt.Sprintf("--%s=%+v", opt, v))
		}
	}
	return args
}

func DiskUsageOfPaths(timeout time.Duration, paths... string) (string, error) {
	var stdout, stderr bytes.Buffer
	args := []string{"-sch"}
	for _, path := range paths {
		args = append(args, path)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout * time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "du", args...)
	fmt.Printf("total size cmd: %s\n", cmd.String())
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("cmd:%s, error: %s", cmd.String(), stderr.String())
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("cmd:%s, output:%s", cmd.String(), stdout.String())
	}
	total := strings.TrimSpace(lines[len(lines)-1])
	if !strings.Contains(total, "total") {
		return "", fmt.Errorf("cmd:%s, output:%s", cmd.String(), stdout.String())
	}
	totalSlice := strings.FieldsFunc(total, func(r rune) bool { return r == ' ' || r == '\t' })
	if len(totalSlice) == 0 {
		return "", fmt.Errorf("cmd:%s, output:%s", cmd.String(), stdout.String())
	}

	return totalSlice[0], nil
}

func FileNumberOfPaths(timeout time.Duration, paths... string) (int, error) {
	filePaths := strings.Join(paths, " ")
	arg := "ls -lR " + filePaths + "| grep \"^-\" | wc -l"

	ctx, cancel := context.WithTimeout(context.Background(), timeout * time.Second)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "bash", "-c", arg)
	fmt.Printf("ls file number cmd: %s\n", cmd.String())
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("cmd:%s, error: %s", cmd.String(), stderr.String())
	}
	fileNum, err := strconv.Atoi(strings.TrimSpace(stdout.String()))
	if err != nil {
		return 0, fmt.Errorf("cmd:%s, atoi error: %s", cmd.String(), err)
	}
	return fileNum, nil
}

func DiskSpaceOfPaths(timeout time.Duration, paths... string) ([]string, error) {
	var stdout, stderr bytes.Buffer
	args := []string{"--total", "-h"}
	for _, path := range paths {
		args = append(args, path)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout * time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "df", args...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cmd:%s, error: %s", cmd.String(), stderr.String())
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("cmd:%s, output:%s", cmd.String(), stdout.String())
	}
	total := strings.TrimSpace(lines[len(lines)-1])
	if !strings.Contains(total, "total") {
		return nil, fmt.Errorf("cmd:%s, output:%s", cmd.String(), stdout.String())
	}
	totalSlice := strings.FieldsFunc(total, func(r rune) bool { return r == ' ' || r == '\t' })

	return totalSlice, nil
}

func Base64Decode(data []byte) (string, error) {
	s, err := base64.StdEncoding.DecodeString(string(data))
	return string(s), err
}

func GetRuntimeImage() (string, error) {
	image := os.Getenv("RUNTIME_IMAGE")
	if image == "" {
		return "", errors.New("RUNTIME_IMAGE is not in environment variable")
	}
	return image, nil
}
