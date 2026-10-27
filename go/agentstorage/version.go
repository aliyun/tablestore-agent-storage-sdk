package agentstorage

// SDKVersion is the Go SDK release version. 0.0.3 aligns public memory,
// file and Wiki APIs with the 2026-10-26 server contract.
const SDKVersion = "0.0.3"

// DefaultUserAgent is sent as the User-Agent header unless a suffix is added
// via Config.UserAgentSuffix.
const DefaultUserAgent = "tablestore-agent-storage-go/" + SDKVersion
