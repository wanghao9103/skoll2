package router

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"skoll2/backend/internal/handler"
	"skoll2/backend/internal/service"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type pluginRouteManifest struct {
	APIPrefix string            `yaml:"apiPrefix"`
	Routes    []pluginRouteItem `yaml:"routes"`
}

type pluginRouteItem struct {
	Method  string `yaml:"method"`
	Path    string `yaml:"path"`
	Handler string `yaml:"handler"`
}

func registerPluginRoutes(protected *gin.RouterGroup, h *handler.Handler, pluginSvc *service.PluginService) {
	for _, item := range pluginSvc.List() {
		manifest, ok, err := loadPluginRouteManifest(pluginSvc.PluginsDir(), item.Key)
		if err != nil || !ok {
			continue
		}

		base := strings.TrimSpace(manifest.APIPrefix)
		if base == "" {
			base = item.APIPrefix
		}

		for _, rt := range manifest.Routes {
			fullPath := joinRoutePath(base, rt.Path)
			handlerFn := h.PluginRouteHandler(item.Key, strings.TrimSpace(rt.Handler))
			registerRoute(protected, strings.ToUpper(strings.TrimSpace(rt.Method)), fullPath, handlerFn)
		}
	}
}

func loadPluginRouteManifest(pluginsDir, pluginKey string) (pluginRouteManifest, bool, error) {
	manifestPaths := []string{
		filepath.Join(pluginsDir, pluginKey, "backend", "api", "routes.yaml"),
		filepath.Join(pluginsDir, pluginKey, "backend", "module.yaml"),
	}

	for _, manifestPath := range manifestPaths {
		raw, err := os.ReadFile(manifestPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return pluginRouteManifest{}, false, err
		}

		manifest := pluginRouteManifest{}
		if err := yaml.Unmarshal(raw, &manifest); err != nil {
			return pluginRouteManifest{}, false, err
		}

		if len(manifest.Routes) == 0 {
			continue
		}

		return manifest, true, nil
	}

	return pluginRouteManifest{}, false, nil
}

func joinRoutePath(base, sub string) string {
	base = strings.TrimSpace(base)
	sub = strings.TrimSpace(sub)

	if base == "" {
		base = "/"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	base = strings.TrimRight(base, "/")

	if sub == "" || sub == "/" {
		return base
	}
	if !strings.HasPrefix(sub, "/") {
		sub = "/" + sub
	}

	return base + sub
}

func registerRoute(group *gin.RouterGroup, method, fullPath string, handlerFn gin.HandlerFunc) {
	switch method {
	case "POST":
		group.POST(fullPath, handlerFn)
	case "PUT":
		group.PUT(fullPath, handlerFn)
	case "PATCH":
		group.PATCH(fullPath, handlerFn)
	case "DELETE":
		group.DELETE(fullPath, handlerFn)
	default:
		group.GET(fullPath, handlerFn)
	}
}
