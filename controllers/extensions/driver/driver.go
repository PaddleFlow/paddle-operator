package driver

import (
	"fmt"
	"github.com/paddleflow/paddle-operator/api/v1alpha1"
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
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
	label := d.GetLabel(ctx.SampleSet.ObjectMeta.Name)
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

// GetLabel label is concatenated by PaddleLabel、driver name and SampleSet name
func (d *BaseDriver) GetLabel(sampleSetName string) string {
	return common.PaddleLabel + "/" +  string(d.Name) + "-" + sampleSetName
}
