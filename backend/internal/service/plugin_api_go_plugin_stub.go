//go:build !linux

package service

import "errors"

func (s *PluginAPIService) executeGoPlugin(pluginKey string, routeHandler string, req execRequest, profile goPluginProfile) (any, error) {
	_ = pluginKey
	_ = routeHandler
	_ = req
	_ = profile
	return nil, errors.New("go-plugin channel is supported on Linux only")
}
