package version

// Version is injected at build time via -ldflags "-X github.com/felixdotgo/querybox/pkg/version.Version=..."
// Falls back to "dev" for local builds.
var Version = "dev"
