package service

type PluginRouteMeta struct {
	Channel string
	Handler string
}

type PluginExecuteResult struct {
	StatusCode  int
	Body        any
	Passthrough bool
}
