package runtimes

type RuntimeInterface interface {

	SyncPartitions() error

	SyncMetadata() error

	CreatePersistentVolume() error

	CreatePersistentVolumeClaim() error

	DeletePersistentVolume() error

	DeletePersistentVolumeClaim() error

	DeleteCache() error
}
