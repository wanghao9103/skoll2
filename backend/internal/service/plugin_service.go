package service

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"skoll2/backend/internal/plugin"
	"skoll2/backend/internal/store"
)

type PluginService struct {
	mu    sync.RWMutex
	items map[string]plugin.Item
	store *store.PluginStore
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

func NewPluginService(pluginStore *store.PluginStore) *PluginService {
	svc := &PluginService{
		items: map[string]plugin.Item{},
		store: pluginStore,
	}
	svc.loadFromStore()
	return svc
}

func (s *PluginService) List() []plugin.Item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]plugin.Item, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
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

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[key]; exists {
		return plugin.Item{}, errors.New("plugin already installed")
	}

	item := plugin.Item{
		Name:          strings.ToUpper(key),
		Key:           key,
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

	s.items[key] = item
	if err := s.store.UpsertPlugin(item); err != nil {
		delete(s.items, key)
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

	item.Status = status
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
			out = append(out, item)
		}
	}

	return out
}

func (s *PluginService) loadFromStore() {
	items, err := s.store.ListPlugins()
	if err != nil {
		return
	}

	for _, item := range items {
		s.items[item.Key] = item
	}
}

func inferPluginKey(packageURL string) string {
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
