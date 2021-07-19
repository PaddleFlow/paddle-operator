package driver

import (
	"fmt"
	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
	"github.com/paddleflow/paddle-operator/controllers/extensions/utils"
	_ "github.com/paddleflow/paddle-operator/controllers/extensions/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"strconv"
	"strings"
)

const (
	// JuiceFSDriver a
	JuiceFSDriver v1alpha1.DriverName = "juicefs"

	// JuiceFSCacheDirOption a
	JuiceFSCacheDirOption = "cache-dir"

	JuiceFSCacheSizeOption = "cache-size"

	// JuiceFSCSIDriverName a
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

// JuiceFSSecretDataKeys is
var (
	// JuiceFSSecretDataKeys a
	JuiceFSSecretDataKeys []string
	// JuiceFSDefaultMountOptions a
	JuiceFSDefaultMountOptions *v1alpha1.JuiceFSMountOptions
)

func init() {
	//
	JuiceFSSecretDataKeys = []string{
		JuiceFSSecretSK, JuiceFSSecretAK,
		JuiceFSSecretName, JuiceFSSecretStorage,
		JuiceFSSecretMetaURL, JuiceFSSecretBucket,
	}

	//
	JuiceFSDefaultMountOptions = &v1alpha1.JuiceFSMountOptions{
		OpenCache: 7200, CacheSize: 1024 * 1024,
		AttrCache: 7200, EntryCache: 7200,
		DirEntryCache: 7200, Prefetch: 1,
		BufferSize: 1024, CacheDir: "/dev/shm/",
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
func (j *JuiceFS) CreatePV(pv *v1.PersistentVolume, ctx common.RequestContext) error {
	ctx.Log.V(1).WithName("CreatePV")
	if !j.isSecretValid(ctx.Secret) {
		return fmt.Errorf("secret %s is not valid", ctx.Secret.ObjectMeta.Name)
	}

	label := j.GetLabel(ctx.SampleSet.ObjectMeta.Name)
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
		ctx.Log.V(1).Error(err, "get getMountOptions error")
		return err
	}

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
				VolumeHandle: label,  // use label as VolumeHandle
				VolumeAttributes: map[string]string{
					"mountOptions": mountOptions,
				},
			},
		},
		PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimRetain,
		VolumeMode: &volumeMode,
	}
	pv.Spec = spec

	return nil
}

// isSecretValid check if the secret created by user is valid for juicefs csi driver
func (j *JuiceFS) isSecretValid(secret *v1.Secret) bool {
	for _, key := range JuiceFSSecretDataKeys {
		if _, exists := secret.Data[key]; !exists {
			return false
		}
	}
	return true
}

// GetMountOptions get the juicefs mount command options set by user
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
	if ratioStr, exists := optionMap["free-space-ratio"]; exists {
		ratio, err := strconv.ParseFloat(ratioStr.String(), 64)
		if err != nil {
			return "", fmt.Errorf("parse free-space-ratio:{%s} error", ratioStr)
		}
		if ratio >= 1.0 || ratio < 0.0 {
			return "", fmt.Errorf("free-space-ratio:{%s} is not valid", ratioStr)
		}
	}

	// Get the cache dir set by user, use default path if not set
	var cacheSize int
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
			if len(cacheDir) > 1 {
				cacheDirs := strings.Join(cacheDir, ":")
				cacheDirOption = JuiceFSCacheDirOption + "=" + cacheDirs
			} else {
				sampleSetName := strings.ToLower(sampleSet.ObjectMeta.Name)
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

