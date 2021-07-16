package driver

import (
	"fmt"
	"github.com/paddleflow/paddle-operator/controllers/extensions/common"
	v1 "k8s.io/api/core/v1"
)

var driverMap map[string]Driver

func init() {
	juiceFS := &JuiceFS{}
	driverMap = map[string]Driver{
		"juicefs": juiceFS,
	}
}

type Driver interface {
	// CreatePV create
	CreatePV(pv *v1.PersistentVolume, ctx common.RequestContext) error

	// CreatePVC create
	CreatePVC(pvc *v1.PersistentVolumeClaim, ctx common.RequestContext) error

	// CreateStorageClass create
	//CreateStorageClass(sc *storagev1.StorageClass) error
}

// GetDriver get csi driver by name
func GetDriver(name string) (Driver, error) {
	if name == "" {
		name = "juicefs"
	}
	if d, exists := driverMap[name]; exists {
		return d, nil
	}
	return nil, fmt.Errorf("driver %s not found", name)
}

type BaseDriver struct {
}

func (d *BaseDriver) CreatePVC(pvc *v1.PersistentVolumeClaim, ctx common.RequestContext) error {

	return nil
}
