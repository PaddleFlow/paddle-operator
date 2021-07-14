package driver

type Driver interface {
	SetupRuntime()

	SyncMetadata()

	WarmupCache()

	DeleteCache()

	CreateVolume()

	DeleteVolume()
}

type BaseDriver struct {

}


