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
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
	"github.com/paddleflow/paddle-operator/controllers/extensions/utils"
)

const (
	DefaultDriver = JuiceFSDriver
)

var (
	StorageClassName = "paddle-operator"
	driverMap map[v1alpha1.DriverName]Driver
)

func init() {
	juiceFS := NewJuiceFSDriver()
	driverMap = map[v1alpha1.DriverName]Driver{
		JuiceFSDriver: juiceFS,
	}
}

type Driver interface {
	// CreatePV create persistent volume by specified driver
	CreatePV(pv *v1.PersistentVolume, ctx common.RequestContext) error

	// CreatePVC create persistent volume claim for PaddleJob
	CreatePVC(pvc *v1.PersistentVolumeClaim, ctx common.RequestContext) error

	// GetLabel get the label to mark pv、pvc and nodes which have cached data
	GetLabel(sampleSetName string) string

	// CreateService create a service for runtime StatefulSet
	CreateService(service *v1.Service, ctx common.RequestContext) error

	// GetServiceName get the name of runtime StatefulSet service
	GetServiceName(sampleSetName string) string

	// CreateRuntime create runtime StatefulSet to manager cache data
	CreateRuntime(ds *appv1.StatefulSet, ctx common.RequestContext) error

	// GetRuntimeName get the runtime StatefulSet name
	GetRuntimeName(sampleSetName string) string

	//CreateSyncJob(job *v1alpha1.SampleJob, ctx common.RequestContext) error

	GetCacheStatus(opt *common.ServerOptions, status *v1alpha1.CacheStatus) error

	DoSyncJob(ctx context.Context, opt *v1alpha1.SyncJobOptions) error

	DoClearJob(ctx context.Context, opt *v1alpha1.ClearJobOptions) error

	DoWarmupJob(ctx context.Context, opt *v1alpha1.WarmupJobOptions) error

	DoRmrJob(ctx context.Context, opt *v1alpha1.RmrJobOptions) error
}

// GetDriver get csi driver by name, return error if not found
func GetDriver(name v1alpha1.DriverName) (Driver, error) {
	if string(name) == "" {
		name = DefaultDriver
	}
	if d, exists := driverMap[name]; exists {
		return d, nil
	}
	return nil, fmt.Errorf("driver %s not found", name)
}

type BaseDriver struct {
	Name v1alpha1.DriverName
}

// CreatePVC a
func (d *BaseDriver) CreatePVC(pvc *v1.PersistentVolumeClaim, ctx common.RequestContext) error {
	label := d.GetLabel(ctx.Req.Name)
	objectMeta := metav1.ObjectMeta{
		Name: ctx.Req.Name,
		Namespace: ctx.Req.Namespace,
		Labels: map[string]string{
			label: "true",
		},
		Annotations: map[string]string{
			"CreatedBy": common.PaddleOperatorLabel,
		},
	}
	pvc.ObjectMeta = objectMeta

	spec := v1.PersistentVolumeClaimSpec{
		AccessModes: []v1.PersistentVolumeAccessMode{
			v1.ReadWriteMany,
		},
		Resources: v1.ResourceRequirements{
			Requests: map[v1.ResourceName]resource.Quantity{
				v1.ResourceStorage: resource.MustParse(common.ResourceStorage),
			},
		},
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				label: "true",
			},
		},
		StorageClassName: &StorageClassName,
	}
	pvc.Spec = spec

	return nil
}

// CreateService create service for runtime StatefulSet server
func (d *BaseDriver) CreateService(service *v1.Service, ctx common.RequestContext) error {
	label := d.GetLabel(ctx.Req.Name)
	serviceName := d.GetServiceName(ctx.Req.Name)
	objectMeta := metav1.ObjectMeta{
		Name: serviceName,
		Namespace: ctx.Req.Namespace,
		Labels: map[string]string{
			label: "true",
		},
		Annotations: map[string]string{
			"CreatedBy": common.PaddleOperatorLabel,
		},
	}
	service.ObjectMeta = objectMeta

	runtimeName := d.GetRuntimeName(ctx.Req.Name)
	selector := map[string]string{
		label: "true",
		"name": runtimeName,
	}

	port := v1.ServicePort{
		Name: common.RuntimeServiceName,
		Port: common.RuntimeServicePort,
	}

	spec := v1.ServiceSpec{
		Selector: selector,
		Ports: []v1.ServicePort{port},
		ClusterIP: "None",
	}
	service.Spec = spec

	return nil
}

// DoClearJob clear the cache data in folders specified by options
func (d *BaseDriver) DoClearJob(ctx context.Context, opt *v1alpha1.ClearJobOptions) error {
	if len(opt.Paths) == 0 {
		return errors.New("clear job option paths not set")
	}
	args := []string{"-rf"}
	for _, path := range opt.Paths {
		if !strings.HasPrefix(path, "/") {
			return fmt.Errorf("path %s is not valid", path)
		}
		args = append(args, path)
	}

	cmd := exec.CommandContext(ctx,"rm", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cmd: %s; error: %s; stderr: %s",
			cmd.String(), err.Error(), stderr.String())
	}
	fmt.Println(stdout)
	return nil
}

func (d *BaseDriver) GetCacheStatus(opt *common.ServerOptions, status *v1alpha1.CacheStatus) error {
	if opt.CacheDirs == nil || len(opt.CacheDirs) == 0 {
		return fmt.Errorf("option cacheDirs not set")
	}
	if !strings.HasPrefix(opt.CacheDirs[0], "/") {
		return fmt.Errorf("option cacheDirs: %s is not valid", opt.CacheDirs[0])
	}
	if !strings.HasPrefix(opt.DataDir, "/") {
		return fmt.Errorf("option dataDir: %s is not valid", opt.DataDir)
	}

	var errs []string
	// get total data size from data mount path
	totalSize, err := utils.DiskUsageOfPaths(opt.DataDir)
	if err == nil {
		status.TotalSize = totalSize
	} else {
		errs = append(errs, fmt.Sprintf("get totalSize error: %s", err.Error()))
	}

	// get cache data size from cache data mount path
	cacheSize, err := utils.DiskUsageOfPaths(opt.CacheDirs...)
	if err == nil {
		status.CachedSize = cacheSize
	} else {
		errs = append(errs, fmt.Sprintf("get cacheSize error: %s", err.Error()))
	}

	// get disk space status of cache paths
	diskStatus, err := utils.DiskSpaceOfPaths(opt.CacheDirs...)
	if err == nil {
		status.DiskSize = strings.TrimSpace(diskStatus[1])
		status.DiskUsed = strings.TrimSpace(diskStatus[2])
		status.DiskAvail = strings.TrimSpace(diskStatus[3])
		status.DiskUsageRate = strings.TrimSpace(diskStatus[4])
	} else {
		errs = append(errs, fmt.Sprintf("get disk status error: %s", err.Error()))
	}

	if len(errs) != 0 {
		return errors.New(strings.Join(errs, ";"))
	}
	return nil
}

// GetLabel label is concatenated by PaddleLabel、driver name and SampleSet name
func (d *BaseDriver) GetLabel(sampleSetName string) string {
	return common.PaddleLabel + "/" +  string(d.Name) + "-" + sampleSetName
}

func (d *BaseDriver) GetRuntimeName(sampleSetName string) string {
	return sampleSetName + "-" + common.RuntimeContainerName
}

func (d *BaseDriver) GetServiceName(sampleSetName string) string {
	return d.GetRuntimeName(sampleSetName) + "-" + common.RuntimeServiceName
}

func (d *BaseDriver) getRuntimeCacheMountPath(name string) string {
	mountPath := os.Getenv("CACHE_MOUNT_PATH")
	if mountPath == "" {
		return common.RuntimeCacheMountPath + "/" + name
	}
	return mountPath + "/" + name
}

func (d *BaseDriver) getRuntimeDataMountPath(name string) string {
	mountPath := os.Getenv("DATA_MOUNT_PATH")
	if mountPath == "" {
		return common.RuntimeDateMountPath + "/" + name
	}
	return mountPath + "/" + name
}