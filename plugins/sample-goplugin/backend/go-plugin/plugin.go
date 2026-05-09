package main

func Ping(request map[string]any, runtime map[string]any) (any, error) {
	return map[string]any{
		"channel": "go-plugin",
		"message": "pong from linux go plugin",
		"request": request,
	}, nil
}
