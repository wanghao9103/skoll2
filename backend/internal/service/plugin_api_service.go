package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"skoll2/backend/internal/config"
	"skoll2/backend/internal/pluginruntime"
	"skoll2/backend/internal/store"

	"github.com/dop251/goja"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type PluginAPIService struct {
	pluginSvc  *PluginService
	pluginsDir string
	runtime    *pluginruntime.Runtime

	defaults processDefaults
	client   *http.Client

	mu        sync.Mutex
	processes map[string]*processState
}

type processDefaults struct {
	DefaultChannel      string
	StartupStrategy     string
	StartupTimeoutMs    int
	RequestTimeoutMs    int
	IdleRecycleSeconds  int
	MaxIdleConnsPerHost int
}

type backendProfile struct {
	Channel  string          `yaml:"channel"`
	Process  processProfile  `yaml:"process"`
	GoPlugin goPluginProfile `yaml:"goPlugin"`
}

type goPluginProfile struct {
	SoPath string `yaml:"soPath"`
}

type processProfile struct {
	Command             string            `yaml:"command"`
	Args                []string          `yaml:"args"`
	Env                 map[string]string `yaml:"env"`
	Port                int               `yaml:"port"`
	HealthPath          string            `yaml:"healthPath"`
	RoutePrefix         string            `yaml:"routePrefix"`
	StartupStrategy     string            `yaml:"startupStrategy"`
	StartupTimeoutMs    int               `yaml:"startupTimeoutMs"`
	RequestTimeoutMs    int               `yaml:"requestTimeoutMs"`
	IdleRecycleSeconds  int               `yaml:"idleRecycleSeconds"`
	MaxIdleConnsPerHost int               `yaml:"maxIdleConnsPerHost"`
}

type processState struct {
	cmd      *exec.Cmd
	profile  processProfile
	lastUsed time.Time
}

type execRequest struct {
	method  string
	path    string
	query   map[string]any
	params  map[string]any
	bodyMap map[string]any
	bodyRaw []byte
}

func NewPluginAPIService(cfg config.Config, pluginSvc *PluginService, pluginStore *store.PluginStore) *PluginAPIService {
	cache := pluginruntime.NewCache()
	transport := &http.Transport{
		MaxIdleConnsPerHost: cfg.PluginProcessMaxIdleConnsHost,
	}
	svc := &PluginAPIService{
		pluginSvc:  pluginSvc,
		pluginsDir: pluginSvc.PluginsDir(),
		runtime:    pluginruntime.New(pluginStore.DB(), cache),
		defaults: processDefaults{
			DefaultChannel:      strings.TrimSpace(cfg.PluginDefaultChannel),
			StartupStrategy:     strings.TrimSpace(cfg.PluginStartupStrategy),
			StartupTimeoutMs:    cfg.PluginProcessStartupTimeoutMs,
			RequestTimeoutMs:    cfg.PluginProcessRequestTimeoutMs,
			IdleRecycleSeconds:  cfg.PluginProcessIdleRecycleSeconds,
			MaxIdleConnsPerHost: cfg.PluginProcessMaxIdleConnsHost,
		},
		client:    &http.Client{Transport: transport},
		processes: map[string]*processState{},
	}

	if svc.defaults.DefaultChannel == "" {
		svc.defaults.DefaultChannel = "js"
	}
	if svc.defaults.StartupStrategy == "" {
		svc.defaults.StartupStrategy = "lazy"
	}
	if svc.defaults.StartupTimeoutMs <= 0 {
		svc.defaults.StartupTimeoutMs = 2000
	}
	if svc.defaults.RequestTimeoutMs <= 0 {
		svc.defaults.RequestTimeoutMs = 3000
	}
	if svc.defaults.IdleRecycleSeconds <= 0 {
		svc.defaults.IdleRecycleSeconds = 180
	}
	if svc.defaults.MaxIdleConnsPerHost <= 0 {
		svc.defaults.MaxIdleConnsPerHost = 2
	}

	go svc.recycleIdleProcesses()
	return svc
}

func (s *PluginAPIService) Execute(c *gin.Context, pluginKey string, meta PluginRouteMeta) (PluginExecuteResult, error) {
	if pluginKey == "" {
		return PluginExecuteResult{}, errors.New("pluginKey is required")
	}
	if !s.pluginSvc.IsEnabled(pluginKey) {
		return PluginExecuteResult{}, fmt.Errorf("%s plugin is not enabled", pluginKey)
	}

	req, err := buildExecRequest(c)
	if err != nil {
		return PluginExecuteResult{}, err
	}

	profile, err := s.loadBackendProfile(pluginKey)
	if err != nil {
		return PluginExecuteResult{}, err
	}

	channel := strings.TrimSpace(meta.Channel)
	if channel == "" {
		channel = strings.TrimSpace(profile.Channel)
	}
	if channel == "" {
		channel = s.defaults.DefaultChannel
	}

	switch strings.ToLower(channel) {
	case "js":
		data, err := s.executeJS(pluginKey, meta.Handler, req)
		if err != nil {
			return PluginExecuteResult{}, err
		}
		return PluginExecuteResult{StatusCode: http.StatusOK, Body: data}, nil
	case "process-http", "python":
		return s.executeProcessHTTP(pluginKey, profile.Process, req)
	case "go-plugin":
		data, err := s.executeGoPlugin(pluginKey, meta.Handler, req, profile.GoPlugin)
		if err != nil {
			return PluginExecuteResult{}, err
		}
		return PluginExecuteResult{StatusCode: http.StatusOK, Body: data}, nil
	default:
		return PluginExecuteResult{}, fmt.Errorf("unsupported plugin channel: %s", channel)
	}
}

func (s *PluginAPIService) executeJS(pluginKey string, routeHandler string, req execRequest) (any, error) {
	scriptPath := filepath.Join(s.pluginsDir, pluginKey, "backend", "api", "handlers.js")
	raw, err := os.ReadFile(scriptPath)
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

	requestPayload := map[string]any{
		"method": req.method,
		"path":   req.path,
		"query":  req.query,
		"params": req.params,
		"body":   req.bodyMap,
	}

	result, err := fn(goja.Undefined(), vm.ToValue(requestPayload), vm.ToValue(runtimeObj))
	if err != nil {
		return nil, unwrapJSException(err)
	}

	return result.Export(), nil
}

func (s *PluginAPIService) executeProcessHTTP(pluginKey string, cfg processProfile, req execRequest) (PluginExecuteResult, error) {
	normalized := s.normalizeProcessProfile(cfg)
	if err := s.ensureProcess(pluginKey, normalized); err != nil {
		return PluginExecuteResult{}, err
	}

	forwardPath := req.path
	if normalized.RoutePrefix != "" && strings.HasPrefix(forwardPath, normalized.RoutePrefix) {
		forwardPath = strings.TrimPrefix(forwardPath, normalized.RoutePrefix)
		if forwardPath == "" {
			forwardPath = "/"
		}
	}
	if !strings.HasPrefix(forwardPath, "/") {
		forwardPath = "/" + forwardPath
	}

	target := fmt.Sprintf("http://127.0.0.1:%d%s", normalized.Port, forwardPath)
	if len(req.query) > 0 {
		q := url.Values{}
		for k, v := range req.query {
			q.Set(k, fmt.Sprintf("%v", v))
		}
		encoded := q.Encode()
		if encoded != "" {
			target += "?" + encoded
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(normalized.RequestTimeoutMs)*time.Millisecond)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, req.method, target, bytes.NewReader(req.bodyRaw))
	if err != nil {
		return PluginExecuteResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return PluginExecuteResult{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return PluginExecuteResult{}, err
	}

	if len(raw) == 0 {
		if resp.StatusCode >= http.StatusBadRequest {
			return PluginExecuteResult{}, fmt.Errorf("plugin process request failed: %d", resp.StatusCode)
		}
		return PluginExecuteResult{StatusCode: resp.StatusCode, Body: map[string]any{}}, nil
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err == nil {
		if obj, ok := decoded.(map[string]any); ok {
			if _, hasCode := obj["code"]; hasCode {
				return PluginExecuteResult{StatusCode: resp.StatusCode, Body: obj, Passthrough: true}, nil
			}
		}
		if resp.StatusCode >= http.StatusBadRequest {
			return PluginExecuteResult{}, fmt.Errorf("plugin process error: %v", decoded)
		}
		return PluginExecuteResult{StatusCode: resp.StatusCode, Body: decoded}, nil
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return PluginExecuteResult{}, fmt.Errorf("plugin process error: %s", string(raw))
	}
	return PluginExecuteResult{StatusCode: resp.StatusCode, Body: map[string]any{"raw": string(raw)}}, nil
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

func (s *PluginAPIService) loadBackendProfile(pluginKey string) (backendProfile, error) {
	profile := backendProfile{Channel: s.defaults.DefaultChannel}
	path := filepath.Join(s.pluginsDir, pluginKey, "backend", "api", "backend.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			profile.Process = s.normalizeProcessProfile(processProfile{})
			return profile, nil
		}
		return backendProfile{}, err
	}
	if err := yaml.Unmarshal(raw, &profile); err != nil {
		return backendProfile{}, err
	}
	if strings.TrimSpace(profile.Channel) == "" {
		profile.Channel = s.defaults.DefaultChannel
	}
	profile.Process = s.normalizeProcessProfile(profile.Process)
	if strings.TrimSpace(profile.GoPlugin.SoPath) == "" {
		profile.GoPlugin.SoPath = "./dist/plugin.so"
	}
	return profile, nil
}

func (s *PluginAPIService) normalizeProcessProfile(p processProfile) processProfile {
	if strings.TrimSpace(p.StartupStrategy) == "" {
		p.StartupStrategy = s.defaults.StartupStrategy
	}
	if p.StartupTimeoutMs <= 0 {
		p.StartupTimeoutMs = s.defaults.StartupTimeoutMs
	}
	if p.RequestTimeoutMs <= 0 {
		p.RequestTimeoutMs = s.defaults.RequestTimeoutMs
	}
	if p.IdleRecycleSeconds <= 0 {
		p.IdleRecycleSeconds = s.defaults.IdleRecycleSeconds
	}
	if p.MaxIdleConnsPerHost <= 0 {
		p.MaxIdleConnsPerHost = s.defaults.MaxIdleConnsPerHost
	}
	if strings.TrimSpace(p.HealthPath) == "" {
		p.HealthPath = "/health"
	}
	return p
}

func (s *PluginAPIService) ensureProcess(pluginKey string, cfg processProfile) error {
	s.mu.Lock()
	state, ok := s.processes[pluginKey]
	if ok && state.cmd != nil && state.cmd.Process != nil && (state.cmd.ProcessState == nil || !state.cmd.ProcessState.Exited()) {
		state.lastUsed = time.Now()
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	if strings.TrimSpace(cfg.Command) == "" || cfg.Port <= 0 {
		return errors.New("process channel requires backend/api/backend.yaml with process.command and process.port")
	}

	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Dir = filepath.Join(s.pluginsDir, pluginKey, "backend")
	cmd.Env = os.Environ()
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	healthURL := fmt.Sprintf("http://127.0.0.1:%d%s", cfg.Port, cfg.HealthPath)
	deadline := time.Now().Add(time.Duration(cfg.StartupTimeoutMs) * time.Millisecond)
	for {
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			return fmt.Errorf("plugin process startup timeout: %s", pluginKey)
		}
		resp, err := s.client.Get(healthURL)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode < http.StatusBadRequest {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	s.mu.Lock()
	s.processes[pluginKey] = &processState{cmd: cmd, profile: cfg, lastUsed: time.Now()}
	s.mu.Unlock()
	return nil
}

func (s *PluginAPIService) recycleIdleProcesses() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for key, state := range s.processes {
			if state == nil || state.cmd == nil || state.cmd.Process == nil {
				delete(s.processes, key)
				continue
			}
			if state.cmd.ProcessState != nil && state.cmd.ProcessState.Exited() {
				delete(s.processes, key)
				continue
			}
			idleLimit := time.Duration(state.profile.IdleRecycleSeconds) * time.Second
			if idleLimit <= 0 {
				continue
			}
			if now.Sub(state.lastUsed) > idleLimit {
				_ = state.cmd.Process.Kill()
				delete(s.processes, key)
			}
		}
		s.mu.Unlock()
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

func (s *PluginAPIService) buildPluginRuntimeMap(pluginKey string) map[string]any {
	return s.buildRuntimeObject(pluginKey)
}

func buildExecRequest(c *gin.Context) (execRequest, error) {
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

	bodyMap := map[string]any{}
	bodyRaw := []byte{}
	if c.Request.Body != nil {
		raw, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return execRequest{}, err
		}
		bodyRaw = raw
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &bodyMap); err != nil {
				return execRequest{}, err
			}
		}
	}

	return execRequest{
		method:  c.Request.Method,
		path:    c.Request.URL.Path,
		query:   query,
		params:  params,
		bodyMap: bodyMap,
		bodyRaw: bodyRaw,
	}, nil
}
