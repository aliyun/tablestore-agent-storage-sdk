package agentstorage

// SDKVersion is the release version of the Go SDK. 0.0.2 adds the memory
// session-affinity and memory-time fields (SearchMemoriesRequest.ContextScope,
// SearchHit.RankingScore / RankingSignals, MemoryUnit title / keywords /
// effective_at / updated_at / time_source / time_precision and the matching
// UpdateMemory overrides).
const SDKVersion = "0.0.2"

// DefaultUserAgent is sent as the User-Agent header unless a suffix is added
// via Config.UserAgentSuffix.
const DefaultUserAgent = "tablestore-agent-storage-go/" + SDKVersion
