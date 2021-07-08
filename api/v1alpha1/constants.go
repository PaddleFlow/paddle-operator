package v1alpha1

const (
	// Pending Bound to runtime, can't be deleted
	Pending SampleSetPhase = "Pending"
	// Bound to dataset, can't be released
	Bound SampleSetPhase = "Bound"
	// Ready can't be deleted
	Ready SampleSetPhase = "Ready"
	// PartialReady Not bound to runtime, can be deleted
	PartialReady SampleSetPhase = "PartialReady"
	// Failed Not bound to runtime, can be deleted
	Failed SampleSetPhase = "Failed"

	Unknown SampleSetPhase = "Unknown"
)

const (
	Memory MediumType = "MEM"

	SSD MediumType = "SSD"

	HDD MediumType = "HDD"
)

// CacheStateName names must be not more than 63 characters, consisting of upper- or lower-case alphanumeric characters,
// with the -, _, and . characters allowed anywhere, except the first or last character.
// The default convention, matching that for annotations, is to use lower-case names, with dashes, rather than
// camel case, separating compound words.
// Fully-qualified resource typenames are constructed from a DNS-style subdomain, followed by a slash `/` and a name.
const (
	// Cached in bytes. (500Gi = 500GiB = 500 * 1024 * 1024 * 1024)
	Cached CacheStateName = "cached"

	// Memory, in bytes. (500Gi = 500GiB = 500 * 1024 * 1024 * 1024)
	// Cacheable CacheStateName = "cacheable"
	LowWaterMark CacheStateName = "lowWaterMark"

	// Memory, in bytes. (500Gi = 500GiB = 500 * 1024 * 1024 * 1024)
	HighWaterMark CacheStateName = "highWaterMark"

	// NonCacheable size, in bytes (e,g. 5Gi = 5GiB = 5 * 1024 * 1024 * 1024)
	NonCacheable CacheStateName = "nonCacheable"

	// Percentage represents the cache percentage over the total data in the underlayer filesystem.
	// 1.5 = 1500m
	CachedPercentage CacheStateName = "cachedPercentage"

	CacheCapacity CacheStateName = "cacheCapacity"

	// CacheHitRatio defines total cache hit ratio(both local hit and remote hit), it is a metric to learn
	// how much profit a distributed cache brings.
	CacheHitRatio CacheStateName = "cacheHitRatio"

	// LocalHitRatio defines local hit ratio. It represents how many data is requested from local cache worker
	LocalHitRatio CacheStateName = "localHitRatio"

	// RemoteHitRatio defines remote hit ratio. It represents how many data is requested from remote cache worker(s).
	RemoteHitRatio CacheStateName = "remoteHitRatio"

	// CacheThroughputRatio defines total cache hit throughput ratio, both local hit and remote hit are included.
	CacheThroughputRatio CacheStateName = "cacheThroughputRatio"

	// LocalThroughputRatio defines local cache hit throughput ratio.
	LocalThroughputRatio CacheStateName = "localThroughputRatio"

	// RemoteThroughputRatio defines remote cache hit throughput ratio.
	RemoteThroughputRatio CacheStateName = "remoteThroughputRatio"
)