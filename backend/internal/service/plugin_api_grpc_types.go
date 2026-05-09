package service

type GRPCRouteRequest struct {
	Handler string         `json:"handler"`
	Method  string         `json:"method"`
	Path    string         `json:"path"`
	Query   map[string]any `json:"query"`
	Params  map[string]any `json:"params"`
	Body    map[string]any `json:"body"`
}

type GRPCRouteResponse struct {
	StatusCode  int    `json:"statusCode"`
	Body        any    `json:"body"`
	Passthrough bool   `json:"passthrough"`
	Error       string `json:"error"`
}
