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
	"os/exec"
	"reflect"
	"strconv"
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
	JuiceFSDriver v1alpha1.DriverName = "juicefs"
	JuiceFSCacheDirOption = "cache-dir"
	JuiceFSCacheSizeOption = "cache-size"
	JuiceFSCSIDriverName = "csi.juicefs.com"
)

//
const (
	JuiceFSSecretName string = "name"
	JuiceFSSecretStorage string = "storage"
	JuiceFSSecretMetaURL string = "metaurl"
	JuiceFSSecretBucket string = "bucket"
	JuiceFSSecretSK string = "secret-key"
	JuiceFSSecretAK string = "access-key"
)

var (
	JuiceFSSecretDataKeys      []string
	JuiceFSSupportStorage      []string
	JuiceFSDefaultMountOptions *v1alpha1.JuiceFSMountOptions
)

func init() {
	// data keys of secret must contains when use JuiceFS as csi driver
	JuiceFSSecretDataKeys = []string{
		JuiceFSSecretName, JuiceFSSecretStorage,
		JuiceFSSecretMetaURL, JuiceFSSecretBucket,
	}

	// default JuiceFS mount volume options pass to pv
	JuiceFSDefaultMountOptions = &v1alpha1.JuiceFSMountOptions{
		OpenCache: 7200, CacheSize: 1024 * 1024,
		AttrCache: 7200, EntryCache: 7200,
		DirEntryCache: 7200, Prefetch: 1,
		BufferSize: 1024, CacheDir: "/dev/shm/",
	}

	JuiceFSSupportStorage = []string{
		common.StorageBOS, common.StorageS3, common.StorageHDFS,
		common.StorageGCS, common.StorageWASB, common.StorageOSS,
		common.StorageCOS, common.StorageKS3, common.StorageUFILE,
		common.StorageQingStor, common.StorageJSS, common.StorageQiNiu,
		common.StorageB2, common.StorageSpace, common.StorageOBS,
		common.StorageOOS, common.StorageSCW, common.StorageMinio,
		common.StorageSCS,
	}

}

type JuiceFS struct {
	BaseDriver
}

func NewJuiceFSDriver() *JuiceFS {
	return &JuiceFS{
		BaseDriver{
			Name: JuiceFSDriver,
		},
	}
}

// CreatePV create JuiceFS persistent volume with mount options.
// How to set parameters of pv can refer to https://github.com/juicedata/juicefs-csi-driver/tree/master/examples/static-provisioning-mount-options
func (j *JuiceFS) CreatePV(pv *v1.PersistentVolume, ctx *common.RequestContext) error {
	if !j.isSecretValid(ctx.Secret) {
		return fmt.Errorf("secret %s is not valid", ctx.Secret.Name)
	}

	label := j.GetLabel(ctx.Req.Name)
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
	pv.ObjectMeta = objectMeta

	volumeMode := v1.PersistentVolumeFilesystem
	mountOptions, err := j.getMountOptions(ctx.SampleSet)
	if err != nil {
		return err
	}

	secretReference := &v1.SecretReference{
		Name: ctx.Secret.Name,
		Namespace: ctx.Secret.Namespace,
	}
	namespacedName := ctx.Req.NamespacedName.String()
	volumeHandle := strings.ReplaceAll(namespacedName, "/", "-")
	spec := v1.PersistentVolumeSpec{
		AccessModes: []v1.PersistentVolumeAccessMode{
			v1.ReadWriteMany,
		},
		Capacity: v1.ResourceList{
			v1.ResourceStorage: resource.MustParse(common.ResourceStorage),
		},
		StorageClassName: StorageClassName,
		PersistentVolumeSource: v1.PersistentVolumeSource{
			CSI: &v1.CSIPersistentVolumeSource{
				Driver: JuiceFSCSIDriverName,
				FSType: string(JuiceFSDriver),
				VolumeHandle: volumeHandle,
				VolumeAttributes: map[string]string{
					"mountOptions": mountOptions,
				},
				NodePublishSecretRef: secretReference,
			},
		},
		PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimRetain,
		NodeAffinity: ctx.SampleSet.Spec.NodeAffinity.DeepCopy(),
		VolumeMode: &volumeMode,
	}
	pv.Spec = spec

	return nil
}

// isSecretValid check if the secret created by user is valid for JuiceFS csi driver
func (j *JuiceFS) isSecretValid(secret *v1.Secret) bool {
	for _, key := range JuiceFSSecretDataKeys {
		if _, exists := secret.Data[key]; !exists {
			return false
		}
	}
	return true
}

// GetMountOptions get the JuiceFS mount command options set by user
func (j *JuiceFS) getMountOptions(sampleSet *v1alpha1.SampleSet) (string, error) {
	optionMap := make(map[string]reflect.Value)
	// get default mount options and values
	utils.NoZeroOptionToMap(optionMap, JuiceFSDefaultMountOptions)

	// if mount options is not set by user, then cover default value with users
	var userOptions *v1alpha1.JuiceFSMountOptions
	if sampleSet.Spec.CSI != nil {
		csi := sampleSet.Spec.CSI
		if csi.JuiceFSMountOptions != nil {
			userOptions = csi.JuiceFSMountOptions.DeepCopy()
		}
	}
	if userOptions != nil {
		utils.NoZeroOptionToMap(optionMap, userOptions)
	}

	// check if free-space-ratio is valid
	if ratioStr, exist := optionMap["free-space-ratio"]; exist {
		ratio, err := strconv.ParseFloat(ratioStr.String(), 64)
		if err != nil {
			return "", fmt.Errorf("parse free-space-ratio:%s error", ratioStr)
		}
		if ratio >= 1.0 || ratio < 0.0 {
			return "", fmt.Errorf("free-space-ratio:%s is not valid", ratioStr)
		}
	}

	// Get the cache dir set by user, use default path if not set
	cacheSize := 0
	var cacheDir []string
	if len(sampleSet.Spec.Cache.Levels) > 0 {
		levels := sampleSet.Spec.Cache.Levels
		for _, cacheLevel := range levels {
			if cacheLevel.Path == "" {
				continue
			}
			cacheDir = append(cacheDir, cacheLevel.Path)
			cacheSize += cacheLevel.CacheSize
		}
	}

	var optionSlice []string
	for option, value := range optionMap {
		// Override cache-dir with the value from Cache Levels
		if option == JuiceFSCacheDirOption {
			var cacheDirOption string
			if len(cacheDir) > 0 {
				cacheDirs := strings.Join(cacheDir, ":")
				cacheDirOption = JuiceFSCacheDirOption + "=" + cacheDirs
			} else {
				sampleSetName := strings.ToLower(sampleSet.Name)
				cacheDirOption = option + "=" + value.String() + sampleSetName
			}
			optionSlice = append(optionSlice, cacheDirOption)
			continue
		}
		// Override cache-size with the value from Cache Levels
		if cacheSize > 0 && option == JuiceFSCacheSizeOption {
			cacheSizeOption := JuiceFSCacheSizeOption + "=" + strconv.Itoa(cacheSize)
			optionSlice = append(optionSlice, cacheSizeOption)
			continue
		}

		if value.Kind() != reflect.Bool {
			option = fmt.Sprintf("%s=%v", option, value)
		}
		optionSlice = append(optionSlice, option)
	}

	return strings.Join(optionSlice, ","), nil
}

func (j *JuiceFS) CreateRuntime(ds *appv1.StatefulSet, ctx *common.RequestContext) error {
	label := j.GetLabel(ctx.Req.Name)
	runtimeName := j.GetRuntimeName(ctx.Req.Name)
	image, err := utils.GetRuntimeImage()
	if err != nil {
		return err
	}

	objectMeta := metav1.ObjectMeta{
		Name: runtimeName,
		Namespace: ctx.Req.Namespace,
		Labels: map[string]string{
			label: "true",
		},
		Annotations: map[string]string{
			"CreatedBy": common.PaddleOperatorLabel,
		},
	}
	ds.ObjectMeta = objectMeta

	volumes, volumeMounts, serverOpt, err := j.getVolumeInfo(ctx.PV)
	if err != nil {
		return fmt.Errorf("getVolumeInfo error: %s", err.Error())
	}

	rootOpt := &common.RootCmdOptions{
		Driver: string(j.Name),
		Development: true,
	}
	command := []string{common.CmdRoot, common.CmdServer}
	rootArgs := utils.NoZeroOptionToArgs(rootOpt)
	serverArgs := utils.NoZeroOptionToArgs(serverOpt)
	command = append(command, rootArgs...)
	command = append(command, serverArgs...)

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			label: "true",
			"name": runtimeName,
		},
	}
	podAffinityTerm := v1.PodAffinityTerm{
		LabelSelector: &labelSelector,
		Namespaces: []string{ctx.Req.Namespace},
		TopologyKey: "kubernetes.io/hostname",
	}
	podAntiAffinity := v1.PodAntiAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
			podAffinityTerm,
		},
	}

	isPrivileged := true
	container := v1.Container{
		Name: common.RuntimeContainerName,
		Image: image,
		Ports: []v1.ContainerPort{
			{
				Name: ctx.Req.Name,
				ContainerPort: common.RuntimeServicePort,
			},
		},
		SecurityContext: &v1.SecurityContext{
			Privileged: &isPrivileged,
		},
		Command: command,
		VolumeMounts: volumeMounts,
	}

	var terminationGracePeriodSeconds int64 = 2
	template := v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				label: "true",
				"name": runtimeName,
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				container,
			},
			Volumes: volumes,
			Affinity: &v1.Affinity{
				PodAntiAffinity: &podAntiAffinity,
			},
			TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
		},
	}

	selector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			label: "true",
			"name": runtimeName,
		},
	}
	// construct StatefulSetSpec
	serviceName := j.GetServiceName(ctx.Req.Name)
	replicas := ctx.SampleSet.Spec.Partitions
	spec := appv1.StatefulSetSpec{
		Replicas: &replicas,
		Selector: &selector,
		Template: template,
		ServiceName: serviceName,
		PodManagementPolicy: appv1.ParallelPodManagement,
	}
	ds.Spec = spec

	return nil
}

func (j *JuiceFS) getVolumeInfo(pv *v1.PersistentVolume) (
	[]v1.Volume, []v1.VolumeMount, *common.ServerOptions, error) {
	// Get cache dir configuration from PV
	if pv.Spec.CSI == nil || pv.Spec.CSI.VolumeAttributes == nil {
		return nil, nil, nil, fmt.Errorf("pv csi field %s is not valid", pv.Name)
	}
	if _, exits := pv.Spec.CSI.VolumeAttributes["mountOptions"]; !exits {
		return nil, nil, nil, fmt.Errorf("pv mountOptions %s is not exist", pv.Name)
	}
	mountOptions := pv.Spec.CSI.VolumeAttributes["mountOptions"]
	optionList := strings.Split(mountOptions, ",")
	serverOpt := &common.ServerOptions{}

	var cacheDirList []string
	for _, option := range optionList {
		if !strings.HasPrefix(option, JuiceFSCacheDirOption) {
			continue
		}
		cacheDir := strings.Split(option, "=")[1]
		cacheDirList = strings.Split(cacheDir, ":")
	}
	if len(cacheDirList) == 0 {
		return nil, nil, nil, fmt.Errorf("cache-dir is not valid")
	}

	hostPathType := v1.HostPathDirectoryOrCreate
	mountPropagation := v1.MountPropagationBidirectional

	var volumes []v1.Volume
	var volumeMounts []v1.VolumeMount
	// construct cache host path volume
	for _, path := range cacheDirList {
		pathTrimPrefix := strings.TrimPrefix(path, "/")
		name := strings.ReplaceAll(pathTrimPrefix, "/", "-")

		hostPath := v1.HostPathVolumeSource{
			Path: path,
			Type: &hostPathType,
		}
		hostPathVolume := v1.Volume{
			Name: name,
			VolumeSource: v1.VolumeSource{
				HostPath: &hostPath,
			},
		}
		volumes = append(volumes, hostPathVolume)
		mountPath := j.getRuntimeCacheMountPath(name)
		volumeMount := v1.VolumeMount{
			Name: name,
			MountPath: mountPath,
			MountPropagation: &mountPropagation,
		}
		volumeMounts = append(volumeMounts, volumeMount)
		serverOpt.CacheDirs = append(serverOpt.CacheDirs, mountPath)
	}

	// construct data persistent volume claim source pods
	pvcs := v1.PersistentVolumeClaimVolumeSource{
		ClaimName: pv.Name,
	}
	volume := v1.Volume{
		Name: pv.Name,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &pvcs,
		},
	}
	volumes = append(volumes, volume)

	mountPath := j.getRuntimeDataMountPath(pv.Name)
	volumeMount := v1.VolumeMount{
		Name: pv.Name,
		MountPath: mountPath,
		MountPropagation: &mountPropagation,
	}
	volumeMounts = append(volumeMounts, volumeMount)
	serverOpt.DataDir = mountPath

	return volumes, volumeMounts, serverOpt, nil
}

func (j *JuiceFS) DoSyncJob(ctx context.Context, opt *v1alpha1.SyncJobOptions) error {
	syncArgs := utils.NoZeroOptionToArgs(&opt.JuiceFSSyncOptions)

	args := []string{"sync"}
	args = append(args, syncArgs...)
	args = append(args, opt.Source)
	args = append(args, opt.Destination)

	cmd := exec.CommandContext(ctx,"juicefs", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("juice sync cmd: %s; error: %s", cmd.String(), err.Error())
	}
	return nil
}

func (j *JuiceFS) DoWarmupJob(ctx context.Context, opt *v1alpha1.WarmupJobOptions) error {
	if len(opt.Paths) == 0 {
		return errors.New("warmup job option paths not set")
	}
	warmupArgs := utils.NoZeroOptionToArgs(&opt.JuiceFSWarmupOptions)

	args := []string{"warmup"}
	args = append(args, warmupArgs...)
	for _, path := range opt.Paths {
		if !strings.HasPrefix(path, "/") {
			return fmt.Errorf("path %s is not valid", path)
		}
		args = append(args, path)
	}

	cmd := exec.CommandContext(ctx, "juicefs", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("juice warmup cmd: %s; error: %s; stderr: %s",
			cmd.String(), err.Error(), stderr.String())
	}
	return nil
}

// DoRmrJob delete the data of JuiceFS storage backend under the specified paths.
// (TODO) there some bugs in JuiceFS rmr command, after rmr paths the sync command
// can't work correctly in container, but posix rm can work well with JuiceFS sync command.
func (j *JuiceFS) DoRmrJob(ctx context.Context, opt *v1alpha1.RmrJobOptions) error {
	if len(opt.Paths) == 0 {
		return errors.New("rmr job option paths not set")
	}
	//args := []string{"rmr"}
	args := []string{"-rf"}
	for _, path := range opt.Paths {
		if !strings.HasPrefix(path, "/") {
			return fmt.Errorf("path %s is not valid", path)
		}
		args = append(args, path)
	}
	//cmd := exec.CommandContext(ctx,"juicefs", args...)
	cmd := exec.CommandContext(ctx,"rm", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("juice rmr cmd: %s; error: %s; stderr: %s",
			cmd.String(), err.Error(), stderr.String())
	}
	return nil
}

// CreateSyncJobOptions create sync job options by the information from request context, the options is used by
// controller to request runtime server do sync data task asynchronously.
// TODO: Support different uri format for all storage in JuiceFSSupportStorage,
// some data storage may need additional secret setting in v1alpha1.Source.SecretRef
// more info: https://github.com/juicedata/juicesync
func (j *JuiceFS) CreateSyncJobOptions(opt *v1alpha1.SyncJobOptions, ctx *common.RequestContext) error {
	// if SampleJob is not nil, use sync job options from SampleJob
	if ctx.SampleJob != nil && ctx.SampleJob.Spec.SyncOptions != nil {
		ctx.SampleJob.Spec.SyncOptions.DeepCopyInto(opt)
	}
	// if SampleJob is nil or source has not been set, use source from SampleSet
	if opt.Source == "" && ctx.SampleSet.Spec.Source != nil {
		opt.Source = ctx.SampleSet.Spec.Source.URI
	}
	// if source has not been set in SampleSet or SampleJob return error
	if opt.Source == "" {
		return fmt.Errorf("data source cannot be empty")
	}
	// verify the format of data source uri
	if !strings.Contains(opt.Source, "://") || len(strings.Split(opt.Source, "://")) != 2 {
		return fmt.Errorf("the format of data source uri is not support")
	}
	// verify the data source storage is supported
	storage := strings.TrimSpace(strings.Split(opt.Source, "://")[0])
	if !utils.ContainsString(JuiceFSSupportStorage, storage) {
		return fmt.Errorf("the object storage %s of is not support", storage)
	}

	var secretKeys string
	delimiter := "://"
	if strings.Contains(opt.Source, ":///") && len(strings.Split(opt.Source, ":///")) == 2 {
		delimiter = ":///"
	}
	path := strings.TrimSpace(strings.Split(opt.Source, delimiter)[1])
	// add access key to source uri if exists in secret
	if akByte, exist := ctx.Secret.Data[JuiceFSSecretAK]; exist && len(akByte) > 0 {
		secretKeys = string(akByte)
	}
	// add secret key to source uri if exists in secret
	if skByte, exist := ctx.Secret.Data[JuiceFSSecretSK]; exist && len(skByte) > 0 {
		secretKeys = secretKeys + ":" + string(skByte)
	}
	if secretKeys != "" {
		opt.Source = storage + delimiter + secretKeys + "@" + path
	}

	// add relative path to sync destination uri
	mountPath := "file://"  + j.getRuntimeDataMountPath(ctx.SampleSet.Name)
	if opt.Destination != "" {
		opt.Destination = mountPath + "/" + strings.TrimPrefix(opt.Destination, "/")
	}

	// source and destination should both end with / or not
	if strings.HasSuffix(opt.Source, "/") != strings.HasSuffix(opt.Destination, "/") {
		opt.Source = strings.TrimSuffix(opt.Source, "/") + "/"
		opt.Destination = strings.TrimSuffix(opt.Destination, "/") + "/"
	}

	return nil
}

func (j *JuiceFS) CreateWarmupJobOptions(opt *v1alpha1.WarmupJobOptions, ctx *common.RequestContext) error {
	return nil
}