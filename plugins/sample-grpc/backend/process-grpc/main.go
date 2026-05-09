package main

import (
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

type routeRequest struct {
	Handler string         `json:"handler"`
	Method  string         `json:"method"`
	Path    string         `json:"path"`
	Query   map[string]any `json:"query"`
	Params  map[string]any `json:"params"`
	Body    map[string]any `json:"body"`
}

type routeResponse struct {
	StatusCode  int    `json:"statusCode"`
	Body        any    `json:"body"`
	Passthrough bool   `json:"passthrough"`
	Error       string `json:"error"`
}

type gateway struct{}

func (gateway) Handle(req *routeRequest, resp *routeResponse) error {
	switch req.Handler {
	case "Ping":
		*resp = routeResponse{
			StatusCode: 200,
			Body: map[string]any{
				"message": "hello from process-grpc",
				"path":    req.Path,
				"method":  req.Method,
			},
		}
	default:
		*resp = routeResponse{StatusCode: 404, Error: "unknown handler: " + req.Handler}
	}
	return nil
}

func main() {
	lis, err := net.Listen("tcp", "127.0.0.1:19104")
	if err != nil {
		log.Fatal(err)
	}
	if err := rpc.RegisterName("PluginGateway", gateway{}); err != nil {
		log.Fatal(err)
	}

	log.Println("sample-grpc listening on 127.0.0.1:19104")
	for {
		conn, err := lis.Accept()
		if err != nil {
			continue
		}
		go rpc.ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}
