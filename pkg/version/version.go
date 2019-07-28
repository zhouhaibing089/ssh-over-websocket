package version

var (
	// GitVersion is populated via -ldflags
	GitVersion = ""
)

// Get returns the version string
func Get() string {
	return GitVersion
}
