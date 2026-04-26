package updater

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/felixdotgo/querybox/pkg/version"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	registryURL  = "https://raw.githubusercontent.com/felixdotgo/querybox/main/plugins/registry.json"
	githubAPIURL = "https://api.github.com/repos/felixdotgo/querybox/releases/latest"
	cacheTTL     = 24 * time.Hour
	cacheFile    = "update-cache.json"
)

// PluginEntry holds registry info for a single plugin.
type PluginEntry struct {
	Version string `json:"version"`
}

// registry mirrors the shape of plugins/registry.json.
type registry struct {
	Plugins map[string]PluginEntry `json:"plugins"`
}

// githubRelease is the subset of the GitHub releases API response we need.
type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// UpdateStatus is returned to the frontend and cached to disk.
type UpdateStatus struct {
	AppUpdateAvailable bool                   `json:"appUpdateAvailable"`
	AppLatestVersion   string                 `json:"appLatestVersion"`
	AppCurrentVersion  string                 `json:"appCurrentVersion"`
	AppReleaseURL      string                 `json:"appReleaseUrl"`
	PluginRegistry     map[string]PluginEntry `json:"pluginRegistry"`
	CheckedAt          time.Time              `json:"checkedAt"`
}

// Updater is a Wails-bound service that checks for app and plugin updates.
type Updater struct {
	app    *application.App
	status *UpdateStatus
}

func New() *Updater {
	return &Updater{}
}

func (u *Updater) SetApp(app *application.App) {
	u.app = app
	go u.maybeCheck()
}

// GetUpdateStatus returns cached update status without blocking.
func (u *Updater) GetUpdateStatus() UpdateStatus {
	if u.status == nil {
		return UpdateStatus{AppCurrentVersion: version.Version}
	}
	return *u.status
}

// CheckNow forces an immediate check, bypassing the 24h cache.
func (u *Updater) CheckNow() UpdateStatus {
	u.doCheck()
	return *u.status
}

func (u *Updater) maybeCheck() {
	if cached, err := u.loadCache(); err == nil && time.Since(cached.CheckedAt) < cacheTTL {
		u.status = cached
		return
	}
	u.doCheck()
}

func (u *Updater) doCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := &UpdateStatus{
		AppCurrentVersion: version.Version,
		CheckedAt:         time.Now(),
	}

	anySuccess := false

	if reg, err := fetchRegistry(ctx); err == nil {
		s.PluginRegistry = reg.Plugins
		anySuccess = true
	}

	if rel, err := fetchAppRelease(ctx); err == nil {
		latest := strings.TrimPrefix(rel.TagName, "v")
		current := strings.TrimPrefix(version.Version, "v")
		s.AppLatestVersion = latest
		s.AppReleaseURL = rel.HTMLURL
		// dev builds cannot be meaningfully compared — skip banner but still fetch
		if version.Version != "dev" {
			s.AppUpdateAvailable = semverGreater(latest, current)
		}
		anySuccess = true
	}

	u.status = s
	// only cache when at least one fetch succeeded — failed results must be retried next startup
	if anySuccess {
		_ = u.saveCache(s)
	}
}

func fetchRegistry(ctx context.Context) (*registry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, registryURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var reg registry
	return &reg, json.NewDecoder(resp.Body).Decode(&reg)
}

func fetchAppRelease(ctx context.Context) (*githubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPIURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var rel githubRelease
	return &rel, json.NewDecoder(resp.Body).Decode(&rel)
}

// semverGreater returns true if a > b (both X.Y.Z, no leading 'v').
func semverGreater(a, b string) bool {
	ap, bp := parseSemver(a), parseSemver(b)
	for i := range ap {
		if ap[i] != bp[i] {
			return ap[i] > bp[i]
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	parts := strings.SplitN(v, ".", 3)
	var out [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		out[i], _ = strconv.Atoi(parts[i])
	}
	return out
}

func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "querybox")
	return dir, os.MkdirAll(dir, 0o700)
}

func (u *Updater) loadCache() (*UpdateStatus, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, cacheFile))
	if err != nil {
		return nil, err
	}
	var s UpdateStatus
	return &s, json.Unmarshal(data, &s)
}

func (u *Updater) saveCache(s *UpdateStatus) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, cacheFile), data, 0o600)
}
