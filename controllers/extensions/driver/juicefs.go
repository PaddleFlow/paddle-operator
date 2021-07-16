package driver

import (
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JuiceFSSecretDataKey string

const (
	JuiceFSType = "juicefs"
	JuiceFSDriverName = "csi.juicefs.com"
)

const (
	JuiceFSSecretName JuiceFSSecretDataKey = "name"
	JuiceFSSecretStorage JuiceFSSecretDataKey = "storage"
	JuiceFSSecretMetaURL JuiceFSSecretDataKey = "metaurl"
	JuiceFSSecretBucket JuiceFSSecretDataKey = "bucket"
	JuiceFSSecretSK JuiceFSSecretDataKey = "secret-key"
	JuiceFSSecretAK JuiceFSSecretDataKey = "access-key"
)

func init() {
	JuiceFSSecretDataKeyList := []JuiceFSSecretDataKey{
		JuiceFSSecretSK, JuiceFSSecretAK,
		JuiceFSSecretName, JuiceFSSecretStorage,
		JuiceFSSecretMetaURL, JuiceFSSecretBucket,
	}
}

type JuiceFS struct {
	BaseDriver
}

func (j *JuiceFS) CreatePV(pv *v1.PersistentVolume, ctx common.RequestContext) error {
	ctx.Log.V(1).Info("Create PV")

	objectMeta := metav1.ObjectMeta{
		Name: ctx.Req.Name,
		Namespace: ctx.Req.Namespace,
		Labels: map[string]string{
			common.PaddleLabel: "true",
		},
		Annotations: map[string]string{
			"CreatedBy": common.PaddleOperatorLabel,
		},
	}
	pv.ObjectMeta = objectMeta

	volumeMode := v1.PersistentVolumeFilesystem

	var volumeHandle string
	if _, exist := ctx.Secret.Data[]


	spec := v1.PersistentVolumeSpec{
		AccessModes: []v1.PersistentVolumeAccessMode{
			v1.ReadWriteMany,
		},
		Capacity: v1.ResourceList{
			v1.ResourceStorage: resource.MustParse("10Pi"),
		},
		StorageClassName: common.StorageClassName,
		PersistentVolumeSource: v1.PersistentVolumeSource{
			CSI: &v1.CSIPersistentVolumeSource{
				Driver: JuiceFSDriverName,
				FSType: JuiceFSType,
				VolumeHandle: "",
			},
		}
		PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimRetain,
		VolumeMode: &volumeMode,
	}
	pv.Spec = spec


	return nil
}

func (j *JuiceFS) IsSecretValid() {

}

