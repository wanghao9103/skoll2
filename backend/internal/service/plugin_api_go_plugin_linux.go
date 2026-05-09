//go:build linux

package service

import (
	"fmt"
	"path/filepath"
	"plugin"
)

type goPluginHandler func(map[string]any, map[string]any) (any, error)

func (s *PluginAPIService) executeGoPlugin(pluginKey string, routeHandler string, req execRequest, profile goPluginProfile) (any, error) {
	soPath := profile.SoPath
	if soPath == "" {
		soPath = "./dist/plugin.so"
	}
	fullPath := filepath.Join(s.pluginsDir, pluginKey, "backend", soPath)

	plg, err := plugin.Open(fullPath)
	if err != nil {
		return nil, err
	}

	symbolName := normalizeHandlerName(routeHandler)
	sym, err := plg.Lookup(symbolName)
	if err != nil {
		return nil, err
	}

	handlerFn, ok := sym.(func(map[string]any, map[string]any) (any, error))
	if !ok {
		if hptr, okPtr := sym.(*goPluginHandler); okPtr && hptr != nil {
			handlerFn = *hptr
		} else {
			return nil, fmt.Errorf("invalid go-plugin handler signature: %s", symbolName)
		}
	}

	requestPayload := map[string]any{
		"method": req.method,
		"path":   req.path,
		"query":  req.query,
		"params": req.params,
		"body":   req.bodyMap,
	}

	return handlerFn(requestPayload, s.buildPluginRuntimeMap(pluginKey))
}
