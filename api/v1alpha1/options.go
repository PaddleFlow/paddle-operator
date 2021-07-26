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

// JuiceFSSyncOptions describes the JuiceFS sync options which user can set by SampleSet
type JuiceFSSyncOptions struct {
	// the first KEY to sync
	// +optional
	Start string `json:"start,omitempty"`
	// the last KEY to sync
	// +optional
	End string `json:"end,omitempty"`
	// number of concurrent threads (default: 10)
	// +optional
	Threads int `json:"threads,omitempty"`
	// HTTP PORT to listen to (default: 6070)
	// +optional
	HttpPort int `json:"http-port,omitempty"`
	// update existing file if the source is newer (default: false)
	// +optional
	Update bool `json:"update,omitempty"`
	// always update existing file (default: false)
	// +optional
	ForceUpdate bool `json:"force-update,omitempty"`
	// preserve permissions (default: false)
	// +optional
	Perms bool `json:"perms,omitempty"`
	// Sync directories or holders (default: false)
	// +optional
	Dirs bool `json:"dirs,omitempty"`
	// Don't copy file (default: false)
	// +optional
	Dry bool `json:"dry,omitempty"`
	// delete objects from source after synced (default: false)
	// +optional
	DeleteSrc bool `json:"delete-src,omitempty"`
	// delete extraneous objects from destination (default: false)
	// +optional
	DeleteDst bool `json:"delete-dst,omitempty"`
	// exclude keys containing PATTERN (POSIX regular expressions)
	// +optional
	Exclude string `json:"exclude,omitempty"`
	// only include keys containing PATTERN (POSIX regular expressions)
	// +optional
	Include string `json:"include,omitempty"`
	// manager address
	// +optional
	Manager string `json:"manager,omitempty"`
	// hosts (seperated by comma) to launch worker
	// +optional
	Worker string `json:"worker,omitempty"`
	// limit bandwidth in Mbps (0 means unlimited) (default: 0)
	// +optional
	BWLimit int `json:"bwlimit,omitempty"`
	// do not use HTTPS (default: false)
	NoHttps bool `json:"no-https,omitempty"`
}

// JuiceFSWarmupOptions describes the JuiceFS warmup options which user can set by SampleSet
type JuiceFSWarmupOptions struct {
	// file containing a list of paths
	// +optional
	File string `json:"file,omitempty"`
	// number of concurrent workers (default: 50)
	// +optional
	Threads int `json:"threads,omitempty"`
}

//
type SyncJobOptions struct {
	// data source that need sync to cache engine
	// +optional
	Source string `json:"source,omitempty"`
	//
	// +option
	Destination string `json:"destination,omitempty"`
	//
	// +optional
	JuiceFSSyncOptions `json:",inline"`
}

// WarmupJobOptions the options for warmup date to local host
type WarmupJobOptions struct {
	// A list of paths need to build cache
	// +optional
	Path []string `json:"path,omitempty"`
	//
	// +optional
	JuiceFSWarmupOptions `json:",inline"`
}

// RmrJobOptions the options for remove data from cache engine
type RmrJobOptions struct {
	Path string `json:"path,omitempty"`
}

// ClearJobOptions the options for clear cache from local host
type ClearJobOptions struct {
	Path string `json:"start,omitempty"`
}
