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
	"fmt"
	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	DoSyncJob(opt *v1alpha1.SyncJobOptions) error

	DoClearJob(opt *v1alpha1.ClearJobOptions) error
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
func (d *BaseDriver) DoClearJob(opt *v1alpha1.ClearJobOptions) error {
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
	return common.RuntimeCacheMountPath + "/" + name
}

func (d *BaseDriver) getRuntimeDataMountPath(name string) string {
	return common.RuntimeDateMountPath + "/" + name
}