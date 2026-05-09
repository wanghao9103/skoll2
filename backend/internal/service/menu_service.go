package service

import "skoll2/backend/internal/plugin"

type MenuService struct {
	pluginSvc *PluginService
}

type MenuNode struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	Component     string `json:"component"`
	Icon          string `json:"icon"`
	PluginKey     string `json:"pluginKey,omitempty"`
	FrontendEntry string `json:"frontendEntry,omitempty"`
	RemoteModule  string `json:"remoteModule,omitempty"`
}

func NewMenuService(pluginSvc *PluginService) *MenuService {
	return &MenuService{pluginSvc: pluginSvc}
}

func (s *MenuService) BuildMenus(username string, role string) []MenuNode {
	base := []MenuNode{
		{
			Name:      "首页",
			Path:      "/dashboard",
			Component: "DashboardPage",
			Icon:      "HomeFilled",
		},
		{
			Name:      "插件管理",
			Path:      "/plugins",
			Component: "PluginManagerPage",
			Icon:      "Operation",
		},
	}

	enabledPlugins := s.pluginSvc.EnabledPlugins()
	for _, p := range enabledPlugins {
		for _, m := range p.Menus {
			remoteModule := m.RemoteModule
			if remoteModule == "" {
				remoteModule = "./App"
			}

			base = append(base, MenuNode{
				Name:          m.Name,
				Path:          m.Path,
				Component:     "RemotePluginPage",
				Icon:          m.Icon,
				PluginKey:     p.Key,
				FrontendEntry: p.FrontendEntry,
				RemoteModule:  remoteModule,
			})
		}
	}

	return base
}

func ConvertPluginMenus(items []plugin.Menu) []MenuNode {
	out := make([]MenuNode, 0, len(items))
	for _, m := range items {
		out = append(out, MenuNode{
			Name:      m.Name,
			Path:      m.Path,
			Component: m.Component,
			Icon:      m.Icon,
		})
	}
	return out
}
