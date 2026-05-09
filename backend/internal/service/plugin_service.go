package service

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"skoll2/backend/internal/plugin"
	"skoll2/backend/internal/store"

	"gopkg.in/yaml.v3"
)

type PluginService struct {
	mu         sync.RWMutex
	items      map[string]plugin.Item
	store      *store.PluginStore
	pluginsDir string
}

type pluginModuleManifest struct {
	Name          string   `yaml:"name"`
	Key           string   `yaml:"key"`
	Version       string   `yaml:"version"`
	Description   string   `yaml:"description"`
	Icon          string   `yaml:"icon"`
	APIPrefix     string   `yaml:"apiPrefix"`
	FrontendEntry string   `yaml:"frontendEntry"`
	RemoteModule  string   `yaml:"remoteModule"`
	Permissions   []string `yaml:"permissions"`
	Menus         []struct {
		Name         string `yaml:"name"`
		Path         string `yaml:"path"`
		Component    string `yaml:"component"`
		Icon         string `yaml:"icon"`
		RemoteModule string `yaml:"remoteModule"`
	} `yaml:"menus"`
}

type pluginBackendProfile struct {
	Channel string `yaml:"channel"`
}

type InstallPluginRequest struct {
	PackageURL string `json:"packageUrl"`
	Checksum   string `json:"checksum"`
}

type TogglePluginRequest struct {
	PluginKey string `json:"pluginKey"`
}

type UpgradePluginRequest struct {
	PluginKey     string `json:"pluginKey"`
	PackageURL    string `json:"packageUrl"`
	TargetVersion string `json:"targetVersion"`
}

type PluginConfigItem struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"isSecret"`
}

type SavePluginConfigRequest struct {
	PluginKey string             `json:"pluginKey"`
	Configs   []PluginConfigItem `json:"configs"`
}

func NewPluginService(pluginStore *store.PluginStore, pluginsDir string) *PluginService {
	svc := &PluginService{
		items:      map[string]plugin.Item{},
		store:      pluginStore,
		pluginsDir: pluginsDir,
	}
	svc.loadFromStore()
	return svc
}

func (s *PluginService) List() []plugin.Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]plugin.Item, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, s.applyRuntimeSupport(item))
	}
	return out
}

func (s *PluginService) Install(req InstallPluginRequest) (plugin.Item, error) {
	if req.PackageURL == "" {
		return plugin.Item{}, errors.New("packageUrl is required")
	}

	key := inferPluginKey(req.PackageURL)
	if key == "" {
		return plugin.Item{}, errors.New("cannot infer plugin key from packageUrl")
	}
	isPluginProtocol := strings.HasPrefix(strings.ToLower(strings.TrimSpace(req.PackageURL)), "plugin://")

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[key]; exists {
		return plugin.Item{}, errors.New("plugin already installed")
	}

	item := plugin.Item{
		Name:          strings.ToUpper(key),
		Key:           key,
		Type:          "unknown",
		Version:       "1.0.0",
		Description:   "Installed from " + req.PackageURL,
		Icon:          "Puzzle",
		Status:        plugin.StatusInstalled,
		APIPrefix:     "/plugin/" + key,
		FrontendEntry: "/plugins/" + key + "/remoteEntry.js",
		Menus: []plugin.Menu{
			{
				Name:      key + " Dashboard",
				Path:      "/plugins/" + key,
				Component: "PluginPage",
				Icon:      "Grid",
			},
		},
		Permissions: []string{key + ":view"},
	}

	manifest, loaded, err := s.loadManifest(key)
	if err != nil {
		return plugin.Item{}, err
	}
	if loaded {
		item = mergeManifest(item, manifest)
	}
	item.Type = s.detectPluginType(item.Key)
	item = s.applyRuntimeSupport(item)
	if isPluginProtocol && !loaded {
		return plugin.Item{}, fmt.Errorf("plugin manifest not found: %s", s.moduleManifestPath(key))
	}

	s.items[key] = item
	if err := s.store.UpsertPlugin(item); err != nil {
		delete(s.items, key)
		return plugin.Item{}, err
	}
	return item, nil
}

func (s *PluginService) InstallFromZip(zipPath string) (plugin.Item, error) {
	if strings.TrimSpace(zipPath) == "" {
		return plugin.Item{}, errors.New("zip path is required")
	}

	tmpDir, err := os.MkdirTemp("", "skoll-plugin-upload-*")
	if err != nil {
		return plugin.Item{}, err
	}
	defer os.RemoveAll(tmpDir)

	if err := unzipArchive(zipPath, tmpDir); err != nil {
		return plugin.Item{}, err
	}

	manifestPath, manifest, err := findPluginManifest(tmpDir)
	if err != nil {
		return plugin.Item{}, err
	}

	key := strings.ToLower(strings.TrimSpace(manifest.Key))
	if key == "" {
		return plugin.Item{}, errors.New("module.yaml missing key")
	}

	pluginSrcDir := filepath.Dir(filepath.Dir(manifestPath))
	pluginDstDir := filepath.Join(s.pluginsDir, key)
	if _, err := os.Stat(pluginDstDir); err == nil {
		return plugin.Item{}, errors.New("plugin already installed")
	}

	if err := copyDir(pluginSrcDir, pluginDstDir); err != nil {
		return plugin.Item{}, err
	}

	item, err := s.Install(InstallPluginRequest{PackageURL: "plugin://" + key})
	if err != nil {
		_ = os.RemoveAll(pluginDstDir)
		return plugin.Item{}, err
	}

	return item, nil
}

func (s *PluginService) Enable(req TogglePluginRequest) (plugin.Item, error) {
	return s.setState(req.PluginKey, plugin.StatusEnabled)
}

func (s *PluginService) Disable(req TogglePluginRequest) (plugin.Item, error) {
	return s.setState(req.PluginKey, plugin.StatusDisabled)
}

func (s *PluginService) Uninstall(req TogglePluginRequest) error {
	if req.PluginKey == "" {
		return errors.New("pluginKey is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[req.PluginKey]
	if !ok {
		return errors.New("plugin not found")
	}
	if item.Status == plugin.StatusEnabled {
		return errors.New("disable plugin before uninstall")
	}

	delete(s.items, req.PluginKey)
	if err := s.store.DeletePlugin(req.PluginKey); err != nil {
		s.items[req.PluginKey] = item
		return err
	}
	return nil
}

func (s *PluginService) Upgrade(req UpgradePluginRequest) (plugin.Item, error) {
	if req.PluginKey == "" {
		return plugin.Item{}, errors.New("pluginKey is required")
	}
	if req.TargetVersion == "" {
		return plugin.Item{}, errors.New("targetVersion is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[req.PluginKey]
	if !ok {
		return plugin.Item{}, errors.New("plugin not found")
	}

	if compareSemver(req.TargetVersion, item.Version) <= 0 {
		return plugin.Item{}, errors.New("targetVersion must be greater than current version")
	}

	item.Version = req.TargetVersion
	if req.PackageURL != "" {
		item.Description = "Upgraded from " + req.PackageURL
	}

	s.items[req.PluginKey] = item
	if err := s.store.UpsertPlugin(item); err != nil {
		return plugin.Item{}, err
	}
	return item, nil
}

func (s *PluginService) SaveConfig(req SavePluginConfigRequest) error {
	if req.PluginKey == "" {
		return errors.New("pluginKey is required")
	}

	s.mu.RLock()
	_, ok := s.items[req.PluginKey]
	s.mu.RUnlock()
	if !ok {
		return errors.New("plugin not found")
	}

	configs := make([]store.PluginConfigItem, 0, len(req.Configs))
	for _, cfg := range req.Configs {
		configs = append(configs, store.PluginConfigItem{
			Key:      strings.TrimSpace(cfg.Key),
			Value:    cfg.Value,
			IsSecret: cfg.IsSecret,
		})
	}

	return s.store.ReplacePluginConfigs(req.PluginKey, configs)
}

func (s *PluginService) GetConfig(pluginKey string) ([]PluginConfigItem, error) {
	if pluginKey == "" {
		return nil, errors.New("pluginKey is required")
	}

	s.mu.RLock()
	_, ok := s.items[pluginKey]
	s.mu.RUnlock()
	if !ok {
		return nil, errors.New("plugin not found")
	}

	configs, err := s.store.ListPluginConfigs(pluginKey)
	if err != nil {
		return nil, err
	}

	out := make([]PluginConfigItem, 0, len(configs))
	for _, cfg := range configs {
		out = append(out, PluginConfigItem{
			Key:      cfg.Key,
			Value:    cfg.Value,
			IsSecret: cfg.IsSecret,
		})
	}

	return out, nil
}

func (s *PluginService) EnabledMenus() []plugin.Menu {
	s.mu.RLock()
	defer s.mu.RUnlock()

	menus := make([]plugin.Menu, 0)
	for _, item := range s.items {
		if item.Status == plugin.StatusEnabled {
			menus = append(menus, item.Menus...)
		}
	}
	return menus
}

func (s *PluginService) setState(pluginKey string, status plugin.Status) (plugin.Item, error) {
	if pluginKey == "" {
		return plugin.Item{}, errors.New("pluginKey is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.items[pluginKey]
	if !ok {
		return plugin.Item{}, errors.New("plugin not found")
	}

	if status == plugin.StatusEnabled {
		manifest, loaded, err := s.loadManifest(pluginKey)
		if err != nil {
			return plugin.Item{}, err
		}
		if loaded {
			item = mergeManifest(item, manifest)
		}
	}
	item.Type = s.detectPluginType(pluginKey)

	item.Status = status
	item = s.applyRuntimeSupport(item)
	if status == plugin.StatusEnabled && !item.RuntimeSupported {
		if strings.TrimSpace(item.RuntimeReason) != "" {
			return plugin.Item{}, errors.New(item.RuntimeReason)
		}
		return plugin.Item{}, errors.New("plugin runtime is not supported in current environment")
	}

	s.items[pluginKey] = item
	if err := s.store.UpsertPlugin(item); err != nil {
		return plugin.Item{}, err
	}
	return item, nil
}

func (s *PluginService) EnabledPlugins() []plugin.Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]plugin.Item, 0)
	for _, item := range s.items {
		if item.Status == plugin.StatusEnabled {
			out = append(out, s.applyRuntimeSupport(item))
		}
	}

	return out
}

func (s *PluginService) Exists(pluginKey string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.items[pluginKey]
	return ok
}

func (s *PluginService) IsEnabled(pluginKey string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[pluginKey]
	if !ok {
		return false
	}
	return item.Status == plugin.StatusEnabled
}

func (s *PluginService) Store() *store.PluginStore {
	return s.store
}

func (s *PluginService) PluginsDir() string {
	return s.pluginsDir
}

func (s *PluginService) loadFromStore() {
	items, err := s.store.ListPlugins()
	if err != nil {
		return
	}

	for _, item := range items {
		manifest, loaded, loadErr := s.loadManifest(item.Key)
		if loadErr == nil && loaded {
			status := item.Status
			item = mergeManifest(item, manifest)
			item.Status = status
		}
		item.Type = s.detectPluginType(item.Key)
		item = s.applyRuntimeSupport(item)
		s.items[item.Key] = item
	}
}

func (s *PluginService) applyRuntimeSupport(item plugin.Item) plugin.Item {
	channel := strings.ToLower(strings.TrimSpace(item.Type))
	if channel == "" {
		channel = "unknown"
	}

	item.RuntimeSupported = true
	item.RuntimeReason = ""

	if channel == "go-plugin" && runtime.GOOS != "linux" {
		item.RuntimeSupported = false
		item.RuntimeReason = fmt.Sprintf("插件通道 go-plugin 仅支持 Linux，当前环境为 %s", runtime.GOOS)
		return item
	}

	if channel == "unknown" {
		item.RuntimeReason = "未识别插件通道，请检查 backend/api/backend.yaml 中的 channel 配置"
	}

	return item
}

func (s *PluginService) detectPluginType(pluginKey string) string {
	path := filepath.Join(s.pluginsDir, pluginKey, "backend", "api", "backend.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return "unknown"
	}
	profile := pluginBackendProfile{}
	if err := yaml.Unmarshal(raw, &profile); err != nil {
		return "unknown"
	}
	channel := strings.ToLower(strings.TrimSpace(profile.Channel))
	if channel == "" {
		return "unknown"
	}
	return channel
}

func findPluginManifest(root string) (string, pluginModuleManifest, error) {
	var foundPath string
	var found pluginModuleManifest

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.ToLower(d.Name()) != "module.yaml" {
			return nil
		}
		if strings.ToLower(filepath.Base(filepath.Dir(path))) != "backend" {
			return nil
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		manifest := pluginModuleManifest{}
		if err := yaml.Unmarshal(raw, &manifest); err != nil {
			return err
		}
		if strings.TrimSpace(manifest.Key) == "" {
			return nil
		}

		foundPath = path
		found = manifest
		return fs.SkipAll
	})
	if err != nil {
		return "", pluginModuleManifest{}, err
	}
	if foundPath == "" {
		return "", pluginModuleManifest{}, errors.New("module.yaml not found in uploaded zip")
	}

	return foundPath, found, nil
}

func unzipArchive(zipPath string, targetDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	cleanRoot := filepath.Clean(targetDir)
	for _, f := range r.File {
		name := filepath.Clean(f.Name)
		if name == "." || strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			return fmt.Errorf("invalid zip entry path: %s", f.Name)
		}

		dstPath := filepath.Join(cleanRoot, name)
		if !strings.HasPrefix(filepath.Clean(dstPath), cleanRoot+string(filepath.Separator)) && filepath.Clean(dstPath) != cleanRoot {
			return fmt.Errorf("zip entry escapes target dir: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dstPath, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}

		src, err := f.Open()
		if err != nil {
			return err
		}

		dst, err := os.Create(dstPath)
		if err != nil {
			_ = src.Close()
			return err
		}

		_, copyErr := io.Copy(dst, src)
		closeErr1 := src.Close()
		closeErr2 := dst.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr1 != nil {
			return closeErr1
		}
		if closeErr2 != nil {
			return closeErr2
		}
	}

	return nil
}

func copyDir(src string, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}

		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()

		if _, err := io.Copy(out, in); err != nil {
			return err
		}
		return nil
	})
}

func inferPluginKey(packageURL string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(packageURL)), "plugin://") {
		key := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(packageURL), "plugin://"))
		key = strings.ToLower(strings.Trim(key, "/"))
		return key
	}

	base := filepath.Base(packageURL)
	if base == "." || base == "/" || base == "" {
		return ""
	}

	ext := filepath.Ext(base)
	key := strings.TrimSuffix(base, ext)
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "_", "-")
	key = strings.ReplaceAll(key, " ", "-")
	return key
}

func compareSemver(a, b string) int {
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")

	maxLen := len(ap)
	if len(bp) > maxLen {
		maxLen = len(bp)
	}

	for i := 0; i < maxLen; i++ {
		av := semverPart(ap, i)
		bv := semverPart(bp, i)
		if av > bv {
			return 1
		}
		if av < bv {
			return -1
		}
	}

	return 0
}

func semverPart(parts []string, idx int) int {
	if idx >= len(parts) {
		return 0
	}

	clean := strings.TrimSpace(parts[idx])
	if clean == "" {
		return 0
	}

	value := 0
	for _, r := range clean {
		if r < '0' || r > '9' {
			break
		}
		value = value*10 + int(r-'0')
	}

	return value
}

func (s *PluginService) moduleManifestPath(pluginKey string) string {
	return filepath.Join(s.pluginsDir, pluginKey, "backend", "module.yaml")
}

func (s *PluginService) loadManifest(pluginKey string) (pluginModuleManifest, bool, error) {
	manifestPath := s.moduleManifestPath(pluginKey)
	if _, err := os.Stat(manifestPath); err != nil {
		if os.IsNotExist(err) {
			return pluginModuleManifest{}, false, nil
		}
		return pluginModuleManifest{}, false, err
	}

	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return pluginModuleManifest{}, false, err
	}

	manifest := pluginModuleManifest{}
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		return pluginModuleManifest{}, false, err
	}

	return manifest, true, nil
}

func mergeManifest(item plugin.Item, manifest pluginModuleManifest) plugin.Item {
	if manifest.Name != "" {
		item.Name = manifest.Name
	}
	if manifest.Key != "" {
		item.Key = manifest.Key
	}
	if manifest.Version != "" {
		item.Version = manifest.Version
	}
	if manifest.Description != "" {
		item.Description = manifest.Description
	}
	if manifest.Icon != "" {
		item.Icon = manifest.Icon
	}
	if manifest.APIPrefix != "" {
		item.APIPrefix = manifest.APIPrefix
	}
	if manifest.FrontendEntry != "" {
		item.FrontendEntry = manifest.FrontendEntry
	}

	if len(manifest.Permissions) > 0 {
		item.Permissions = manifest.Permissions
	}

	if len(manifest.Menus) > 0 {
		menus := make([]plugin.Menu, 0, len(manifest.Menus))
		for _, m := range manifest.Menus {
			remoteModule := m.RemoteModule
			if remoteModule == "" {
				remoteModule = manifest.RemoteModule
			}
			menus = append(menus, plugin.Menu{
				Name:         m.Name,
				Path:         m.Path,
				Component:    m.Component,
				Icon:         m.Icon,
				RemoteModule: remoteModule,
			})
		}
		item.Menus = menus
	}

	return item
}
