package pluginmgr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/felixdotgo/querybox/pkg/driverid"
	"github.com/felixdotgo/querybox/pkg/plugin"
)

type PluginRegistry struct {
	mu      sync.RWMutex
	dirs    []string
	plugins map[string]PluginInfo
}

func NewPluginRegistry(dirs []string) *PluginRegistry {
	return &PluginRegistry{
		dirs:    append([]string(nil), dirs...),
		plugins: make(map[string]PluginInfo),
	}
}

func (r *PluginRegistry) SetDirs(dirs []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dirs = append([]string(nil), dirs...)
}

func (r *PluginRegistry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plugins = make(map[string]PluginInfo)
}

func (r *PluginRegistry) Scan() map[string]PluginInfo {
	r.mu.RLock()
	dirs := append([]string(nil), r.dirs...)
	existing := clonePluginInfoMap(r.plugins)
	r.mu.RUnlock()

	found := map[string]struct{}{}
	type candidate struct {
		name   string
		full   string
		dirIdx int
	}
	var toProbe []candidate

	for idx, dir := range dirs {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			origName := file.Name()
			if filepath.Ext(origName) == ".json" {
				continue
			}
			name := driverid.Normalize(origName)
			if _, seen := found[name]; seen {
				continue
			}
			full := filepath.Join(dir, origName)
			if !isExecutable(full) {
				continue
			}
			found[name] = struct{}{}
			current, ok := existing[name]
			if !ok || current.LastError != "" {
				toProbe = append(toProbe, candidate{name: name, full: full, dirIdx: idx})
			} else {
				existing[name] = current
			}
		}
	}

	type result struct {
		name string
		info PluginInfo
	}
	resCh := make(chan result, len(toProbe))
	var wg sync.WaitGroup
	for _, cand := range toProbe {
		wg.Add(1)
		go func(c candidate) {
			defer wg.Done()
			info, err := discoverPlugin(c.full, c.name)
			if err != nil && c.dirIdx == 0 && len(dirs) > 1 {
				alt := filepath.Join(dirs[len(dirs)-1], filepath.Base(c.full))
				if alt != c.full && isExecutable(alt) {
					if altInfo, altErr := discoverPlugin(alt, c.name); altErr == nil {
						info = altInfo
						err = nil
					}
				}
			}
			if err != nil {
				info.LastError = err.Error()
			}
			resCh <- result{name: c.name, info: info}
		}(cand)
	}
	wg.Wait()
	close(resCh)

	next := make(map[string]PluginInfo, len(found))
	for name, info := range existing {
		if _, ok := found[name]; ok {
			next[name] = info
		}
	}
	for res := range resCh {
		next[res.name] = res.info
	}

	r.mu.Lock()
	r.plugins = clonePluginInfoMap(next)
	r.mu.Unlock()
	return next
}

func discoverPlugin(fullpath, fallbackName string) (PluginInfo, error) {
	info := PluginInfo{
		ID:      fallbackName,
		Name:    fallbackName,
		Path:    fullpath,
		Running: false,
	}

	manifestPath := fullpath + plugin.ManifestFileSuffix
	manifest, err := loadManifestFile(manifestPath, fallbackName)
	if err != nil {
		return info, fmt.Errorf("load manifest: %w", err)
	}
	applyManifest(&info, manifestPath, manifest)
	return info, nil
}

func loadManifestFile(path, expectedID string) (*plugin.Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest plugin.Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest json: %w", err)
	}
	if err := plugin.ValidateManifest(manifest, expectedID); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func applyManifest(info *PluginInfo, manifestPath string, manifest *plugin.Manifest) {
	if manifest == nil {
		return
	}
	info.ID = manifest.ID
	info.ManifestPath = manifestPath
	if manifest.Name != "" {
		info.Name = manifest.Name
	}
	if manifest.Description != "" {
		info.Description = manifest.Description
	}
	info.Version = manifest.Version
	info.URL = manifest.URL
	info.Author = manifest.Author
	info.Type = manifest.Type
	info.Capabilities = append([]string(nil), manifest.Capabilities...)
	info.Permissions = append([]plugin.PermissionDecl(nil), manifest.Permissions...)
	limitsCopy := manifest.Limits
	info.Limits = &limitsCopy
	runtimeCopy := manifest.Runtime
	info.Runtime = &runtimeCopy
	info.Tags = append([]string(nil), manifest.Tags...)
	info.License = manifest.License
	info.IconURL = manifest.IconURL
	info.Contact = manifest.Contact
	if len(manifest.Metadata) > 0 {
		info.Metadata = cloneStringMap(manifest.Metadata)
	}
	if len(manifest.Settings) > 0 {
		info.Settings = cloneStringMap(manifest.Settings)
	}
}

func clonePluginInfoMap(src map[string]PluginInfo) map[string]PluginInfo {
	dst := make(map[string]PluginInfo, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}
