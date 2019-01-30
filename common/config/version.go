package config

// These will be set by Travis
// -X github.com/WeTrustPlatform/account-indexer/common/config.*=*
var version = "unknown"
var buildID = "0"
var githash = "0"

// VersionInfo version information
type VersionInfo struct {
	Version string `json:"version"`
	BuildID string `json:"buildID"`
	GitHash string `json:"githash"`
}

// GetVersion for the Rest version api
func GetVersion() VersionInfo {
	return VersionInfo{
		Version: version,
		BuildID: buildID,
		GitHash: githash,
	}
}
