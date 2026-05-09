package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
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
	stats     map[string]*processStats
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
	GRPC     grpcProfile     `yaml:"grpc"`
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

type grpcProfile struct {
	Command            string            `yaml:"command"`
	Args               []string          `yaml:"args"`
	Env                map[string]string `yaml:"env"`
	Address            string            `yaml:"address"`
	StartupStrategy    string            `yaml:"startupStrategy"`
	StartupTimeoutMs   int               `yaml:"startupTimeoutMs"`
	RequestTimeoutMs   int               `yaml:"requestTimeoutMs"`
	IdleRecycleSeconds int               `yaml:"idleRecycleSeconds"`
}

type processState struct {
	cmd               *exec.Cmd
	pluginKey         string
	key               string
	channel           string
	idleRecycleSecond int
	startedAt         time.Time
	lastUsed          time.Time
}

type processStats struct {
	key            string
	pluginKey      string
	channel        string
	restartCount   int
	lastStartAt    time.Time
	lastUsedAt     time.Time
	lastStopAt     time.Time
	lastExitReason string
	lastError      string
}

type PluginProcessStatus struct {
	Key            string `json:"key"`
	PluginKey      string `json:"pluginKey"`
	Channel        string `json:"channel"`
	Running        bool   `json:"running"`
	PID            int    `json:"pid"`
	RestartCount   int    `json:"restartCount"`
	LastStartAt    string `json:"lastStartAt,omitempty"`
	LastUsedAt     string `json:"lastUsedAt,omitempty"`
	LastStopAt     string `json:"lastStopAt,omitempty"`
	LastExitReason string `json:"lastExitReason,omitempty"`
	LastError      string `json:"lastError,omitempty"`
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
		stats:     map[string]*processStats{},
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
	case "process-grpc":
		return s.executeProcessGRPC(pluginKey, meta.Handler, profile.GRPC, req)
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
	if err := s.ensureHTTPProcess(pluginKey, normalized); err != nil {
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

func (s *PluginAPIService) executeProcessGRPC(pluginKey string, routeHandler string, cfg grpcProfile, req execRequest) (PluginExecuteResult, error) {
	normalized := s.normalizeGRPCProfile(cfg)
	if err := s.ensureGRPCProcess(pluginKey, normalized); err != nil {
		return PluginExecuteResult{}, err
	}

	conn, err := net.DialTimeout("tcp", normalized.Address, time.Duration(normalized.RequestTimeoutMs)*time.Millisecond)
	if err != nil {
		return PluginExecuteResult{}, err
	}
	defer conn.Close()

	client := rpc.NewClientWithCodec(jsonrpc.NewClientCodec(conn))
	defer client.Close()

	grpcReq := &GRPCRouteRequest{
		Handler: normalizeHandlerName(routeHandler),
		Method:  req.method,
		Path:    req.path,
		Query:   req.query,
		Params:  req.params,
		Body:    req.bodyMap,
	}

	grpcRes := &GRPCRouteResponse{}
	if err := client.Call("PluginGateway.Handle", grpcReq, grpcRes); err != nil {
		return PluginExecuteResult{}, err
	}
	if grpcRes.Error != "" {
		return PluginExecuteResult{}, errors.New(grpcRes.Error)
	}

	status := grpcRes.StatusCode
	if status <= 0 {
		status = http.StatusOK
	}

	return PluginExecuteResult{StatusCode: status, Body: grpcRes.Body, Passthrough: grpcRes.Passthrough}, nil
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
	profile.GRPC = s.normalizeGRPCProfile(profile.GRPC)
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

func (s *PluginAPIService) normalizeGRPCProfile(p grpcProfile) grpcProfile {
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
	return p
}

func buildProcessKey(pluginKey string, channel string) string {
	return pluginKey + ":" + channel
}

func (s *PluginAPIService) upsertStatLocked(key string, pluginKey string, channel string) *processStats {
	item, ok := s.stats[key]
	if !ok || item == nil {
		item = &processStats{key: key, pluginKey: pluginKey, channel: channel}
		s.stats[key] = item
	}
	return item
}

func (s *PluginAPIService) markStartLocked(key string, pluginKey string, channel string) {
	now := time.Now()
	item := s.upsertStatLocked(key, pluginKey, channel)
	item.restartCount++
	item.lastStartAt = now
	item.lastUsedAt = now
	item.lastError = ""
	item.lastExitReason = ""
}

func (s *PluginAPIService) markUsedLocked(key string, pluginKey string, channel string) {
	now := time.Now()
	item := s.upsertStatLocked(key, pluginKey, channel)
	item.lastUsedAt = now
}

func (s *PluginAPIService) markErrorLocked(key string, pluginKey string, channel string, err error) {
	if err == nil {
		return
	}
	item := s.upsertStatLocked(key, pluginKey, channel)
	item.lastError = err.Error()
}

func (s *PluginAPIService) markStopLocked(key string, pluginKey string, channel string, reason string) {
	item := s.upsertStatLocked(key, pluginKey, channel)
	item.lastStopAt = time.Now()
	item.lastExitReason = strings.TrimSpace(reason)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func isProcessRunning(state *processState) bool {
	if state == nil || state.cmd == nil || state.cmd.Process == nil {
		return false
	}
	if state.cmd.ProcessState == nil {
		return true
	}
	return !state.cmd.ProcessState.Exited()
}

func (s *PluginAPIService) ensureHTTPProcess(pluginKey string, cfg processProfile) error {
	key := buildProcessKey(pluginKey, "process-http")

	s.mu.Lock()
	state, ok := s.processes[key]
	if ok && state.cmd != nil && state.cmd.Process != nil && (state.cmd.ProcessState == nil || !state.cmd.ProcessState.Exited()) {
		state.lastUsed = time.Now()
		s.markUsedLocked(key, pluginKey, "process-http")
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	if strings.TrimSpace(cfg.Command) == "" || cfg.Port <= 0 {
		err := errors.New("process channel requires backend/api/backend.yaml with process.command and process.port")
		s.mu.Lock()
		s.markErrorLocked(key, pluginKey, "process-http", err)
		s.mu.Unlock()
		return err
	}

	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Dir = filepath.Join(s.pluginsDir, pluginKey, "backend")
	cmd.Env = os.Environ()
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	if err := cmd.Start(); err != nil {
		s.mu.Lock()
		s.markErrorLocked(key, pluginKey, "process-http", err)
		s.mu.Unlock()
		return err
	}

	healthURL := fmt.Sprintf("http://127.0.0.1:%d%s", cfg.Port, cfg.HealthPath)
	deadline := time.Now().Add(time.Duration(cfg.StartupTimeoutMs) * time.Millisecond)
	for {
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			err := fmt.Errorf("plugin process startup timeout: %s", pluginKey)
			s.mu.Lock()
			s.markErrorLocked(key, pluginKey, "process-http", err)
			s.markStopLocked(key, pluginKey, "process-http", "startup timeout")
			s.mu.Unlock()
			return err
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
	now := time.Now()
	s.processes[key] = &processState{cmd: cmd, key: key, pluginKey: pluginKey, channel: "process-http", idleRecycleSecond: cfg.IdleRecycleSeconds, startedAt: now, lastUsed: now}
	s.markStartLocked(key, pluginKey, "process-http")
	s.mu.Unlock()
	return nil
}

func (s *PluginAPIService) ensureGRPCProcess(pluginKey string, cfg grpcProfile) error {
	key := buildProcessKey(pluginKey, "process-grpc")

	s.mu.Lock()
	state, ok := s.processes[key]
	if ok && state.cmd != nil && state.cmd.Process != nil && (state.cmd.ProcessState == nil || !state.cmd.ProcessState.Exited()) {
		state.lastUsed = time.Now()
		s.markUsedLocked(key, pluginKey, "process-grpc")
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	if strings.TrimSpace(cfg.Command) == "" || strings.TrimSpace(cfg.Address) == "" {
		err := errors.New("process-grpc channel requires backend/api/backend.yaml with grpc.command and grpc.address")
		s.mu.Lock()
		s.markErrorLocked(key, pluginKey, "process-grpc", err)
		s.mu.Unlock()
		return err
	}

	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Dir = filepath.Join(s.pluginsDir, pluginKey, "backend")
	cmd.Env = os.Environ()
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	if err := cmd.Start(); err != nil {
		s.mu.Lock()
		s.markErrorLocked(key, pluginKey, "process-grpc", err)
		s.mu.Unlock()
		return err
	}

	deadline := time.Now().Add(time.Duration(cfg.StartupTimeoutMs) * time.Millisecond)
	for {
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			err := fmt.Errorf("plugin grpc process startup timeout: %s", pluginKey)
			s.mu.Lock()
			s.markErrorLocked(key, pluginKey, "process-grpc", err)
			s.markStopLocked(key, pluginKey, "process-grpc", "startup timeout")
			s.mu.Unlock()
			return err
		}
		conn, err := net.DialTimeout("tcp", cfg.Address, 300*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	s.mu.Lock()
	now := time.Now()
	s.processes[key] = &processState{cmd: cmd, key: key, pluginKey: pluginKey, channel: "process-grpc", idleRecycleSecond: cfg.IdleRecycleSeconds, startedAt: now, lastUsed: now}
	s.markStartLocked(key, pluginKey, "process-grpc")
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
				if state != nil {
					s.markStopLocked(key, state.pluginKey, state.channel, "invalid process state")
				}
				delete(s.processes, key)
				continue
			}
			if state.cmd.ProcessState != nil && state.cmd.ProcessState.Exited() {
				exitReason := "exited"
				if code := state.cmd.ProcessState.ExitCode(); code >= 0 {
					exitReason = fmt.Sprintf("exited(%d)", code)
				}
				s.markStopLocked(key, state.pluginKey, state.channel, exitReason)
				delete(s.processes, key)
				continue
			}
			idleLimit := time.Duration(state.idleRecycleSecond) * time.Second
			if idleLimit <= 0 {
				continue
			}
			if now.Sub(state.lastUsed) > idleLimit {
				_ = state.cmd.Process.Kill()
				s.markStopLocked(key, state.pluginKey, state.channel, "idle recycle")
				delete(s.processes, key)
			}
		}
		s.mu.Unlock()
	}
}

func (s *PluginAPIService) ListProcessStatuses(pluginKey string) []PluginProcessStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]PluginProcessStatus, 0, len(s.stats)+len(s.processes))
	seen := map[string]bool{}

	for key, item := range s.stats {
		if item == nil {
			continue
		}
		if pluginKey != "" && item.pluginKey != pluginKey {
			continue
		}
		status := PluginProcessStatus{
			Key:            key,
			PluginKey:      item.pluginKey,
			Channel:        item.channel,
			RestartCount:   item.restartCount,
			LastStartAt:    formatTime(item.lastStartAt),
			LastUsedAt:     formatTime(item.lastUsedAt),
			LastStopAt:     formatTime(item.lastStopAt),
			LastExitReason: item.lastExitReason,
			LastError:      item.lastError,
		}

		if state, ok := s.processes[key]; ok && isProcessRunning(state) {
			status.Running = true
			if state.cmd != nil && state.cmd.Process != nil {
				status.PID = state.cmd.Process.Pid
			}
		}

		seen[key] = true
		result = append(result, status)
	}

	for key, state := range s.processes {
		if seen[key] || state == nil {
			continue
		}
		if pluginKey != "" && state.pluginKey != pluginKey {
			continue
		}
		status := PluginProcessStatus{
			Key:         key,
			PluginKey:   state.pluginKey,
			Channel:     state.channel,
			Running:     isProcessRunning(state),
			LastStartAt: formatTime(state.startedAt),
			LastUsedAt:  formatTime(state.lastUsed),
		}
		if state.cmd != nil && state.cmd.Process != nil {
			status.PID = state.cmd.Process.Pid
		}
		result = append(result, status)
	}

	return result
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
