package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"skoll2/backend/internal/pluginruntime"
	"skoll2/backend/internal/store"

	"github.com/dop251/goja"
	"github.com/gin-gonic/gin"
)

type PluginAPIService struct {
	pluginSvc  *PluginService
	pluginsDir string
	runtime    *pluginruntime.Runtime
}

func NewPluginAPIService(pluginSvc *PluginService, pluginStore *store.PluginStore) *PluginAPIService {
	cache := pluginruntime.NewCache()
	return &PluginAPIService{
		pluginSvc:  pluginSvc,
		pluginsDir: pluginSvc.PluginsDir(),
		runtime:    pluginruntime.New(pluginStore.DB(), cache),
	}
}

func (s *PluginAPIService) Execute(c *gin.Context, pluginKey string, routeHandler string) (any, error) {
	if pluginKey == "" {
		return nil, errors.New("pluginKey is required")
	}
	if !s.pluginSvc.IsEnabled(pluginKey) {
		return nil, fmt.Errorf("%s plugin is not enabled", pluginKey)
	}

	scriptPath := filepath.Join(s.pluginsDir, pluginKey, "backend", "api", "handlers.js")
	raw, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, err
	}

	requestPayload, err := buildPluginRequestPayload(c)
	if err != nil {
		return nil, err
	}

	vm := goja.New()
	runtimeObj := s.buildRuntimeObject(pluginKey)

	if _, err := vm.RunString(string(raw)); err != nil {
		return nil, unwrapJSException(err)
	}

	fnName := normalizeHandlerName(routeHandler)
	fnValue := vm.Get(fnName)
	fn, ok := goja.AssertFunction(fnValue)
	if !ok {
		return nil, fmt.Errorf("handler function not found: %s", fnName)
	}

	result, err := fn(goja.Undefined(), vm.ToValue(requestPayload), vm.ToValue(runtimeObj))
	if err != nil {
		return nil, unwrapJSException(err)
	}

	return result.Export(), nil
}

func (s *PluginAPIService) buildRuntimeObject(pluginKey string) map[string]any {
	db := s.runtime.DB()
	cache := s.runtime.Cache()

	return map[string]any{
		"db": map[string]any{
			"list": func(table string, order string) ([]map[string]any, error) {
				return db.List(table, order)
			},
			"getById": func(table string, id int64) (map[string]any, error) {
				return db.FirstByID(table, id)
			},
			"create": func(table string, values map[string]any) (map[string]any, error) {
				return db.Create(table, values)
			},
			"updateById": func(table string, id int64, values map[string]any) (map[string]any, error) {
				return db.UpdateByID(table, id, values)
			},
			"deleteById": func(table string, id int64) (bool, error) {
				return true, db.DeleteByID(table, id)
			},
		},
		"cache": map[string]any{
			"set": func(key string, value any, ttlSeconds int64) bool {
				return cache.Set(pluginKey, key, value, ttlSeconds)
			},
			"get": func(key string) map[string]any {
				v, ok := cache.Get(pluginKey, key)
				return map[string]any{"value": v, "ok": ok}
			},
			"del": func(key string) bool {
				return cache.Delete(pluginKey, key)
			},
		},
	}
}

func normalizeHandlerName(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ".")
	return strings.TrimSpace(parts[len(parts)-1])
}

func unwrapJSException(err error) error {
	if ex, ok := err.(*goja.Exception); ok {
		return errors.New(ex.String())
	}
	return err
}

func buildPluginRequestPayload(c *gin.Context) (map[string]any, error) {
	query := map[string]any{}
	for k, v := range c.Request.URL.Query() {
		if len(v) == 0 {
			query[k] = ""
			continue
		}
		query[k] = v[0]
	}

	params := map[string]any{}
	for _, p := range c.Params {
		params[p.Key] = p.Value
	}

	body := map[string]any{}
	if c.Request.Body != nil {
		raw, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, err
		}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &body); err != nil {
				return nil, err
			}
		}
	}

	username, _ := c.Get("username")
	role, _ := c.Get("role")

	return map[string]any{
		"method": c.Request.Method,
		"path":   c.Request.URL.Path,
		"query":  query,
		"params": params,
		"body":   body,
		"auth": map[string]any{
			"username": username,
			"role":     role,
		},
	}, nil
}
