package v1alpha1

// JuiceFSMountOptions describes the JuiceFS mount options which user can set
// All the mount options is list in https://github.com/juicedata/juicefs/blob/main/docs/en/command_reference.md
type JuiceFSMountOptions struct {
	// address to export metrics (default: "127.0.0.1:9567")
	// +optional
	Metrics string `json:"metrics,omitempty"`
	// attributes cache timeout in seconds (default: 1)
	// +optional
	AttrCache int `json:"attr-cache,omitempty"`
	// file entry cache timeout in seconds (default: 1)
	// +optional
	EntryCache int `json:"entry-cache,omitempty"`
	// dir entry cache timeout in seconds (default: 1)
	// +optional
	DirEntryCache int `json:"dir-entry-cache,omitempty"`
	// enable extended attributes (xattr) (default: false)
	// +optional
	EnableXattr bool `json:"enable-xattr,omitempty"`
	// the max number of seconds to download an object (default: 60)
	// +optional
	GetTimeout int `json:"get-timeout,omitempty"`
	// the max number of seconds to upload an object (default: 60)
	// +optional
	PutTimeout int `json:"put-timeout,omitempty"`
	// number of retries after network failure (default: 30)
	// +optional
	IoRetries int `json:"io-retries,omitempty"`
	// number of connections to upload (default: 20)
	// +optional
	MaxUploads int `json:"max-uploads,omitempty"`
	// total read/write buffering in MB (default: 300)
	// +optional
	BufferSize int `json:"buffer-size,omitempty"`
	// prefetch N blocks in parallel (default: 1)
	// +optional
	Prefetch int `json:"prefetch,omitempty"`
	// upload objects in background (default: false)
	// +optional
	WriteBack bool `json:"writeback,omitempty"`
	// directory paths of local cache, use colon to separate multiple paths
	// +optional
	CacheDir string `json:"cache-dir,omitempty"`
	// size of cached objects in MiB (default: 1024)
	// +optional
	CacheSize int `json:"cache-size,omitempty"`
	// min free space (ratio) (default: 0.1)
	// float64 is not supported https://github.com/kubernetes-sigs/controller-tools/issues/245
	// +optional
	FreeSpaceRatio string `json:"free-space-ratio,omitempty"`
	// cache only random/small read (default: false)
	// +optional
	CachePartialOnly bool `json:"cache-partial-only,omitempty"`
	// open files cache timeout in seconds (0 means disable this feature) (default: 0)
	// +optional
	OpenCache int `json:"open-cache,omitempty"`
	// mount a sub-directory as root
	// +optional
	SubDir string `json:"sub-dir,omitempty"`
}

type SyncJobOptions struct {
	//
	// +optional
	Source string `json:"source,omitempty"`
}

type WarmupJobOptions struct {
	Path string `json:"start,omitempty"`
}

type RmrJobOptions struct {
	Path string `json:"start,omitempty"`
}

type ClearJobOptions struct {
	Path string `json:"start,omitempty"`
}
