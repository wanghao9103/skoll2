package main

import (
	"log"

	"skoll2/backend/internal/config"
	"skoll2/backend/internal/router"
	"skoll2/backend/internal/service"
	"skoll2/backend/internal/store"
)

func main() {
	cfg := config.Load()
	pluginStore, err := store.NewPluginStore(cfg.DBDriver, cfg.DBDSN)
	if err != nil {
		log.Fatalf("init plugin store failed: %v", err)
	}

	authSvc := service.NewAuthService(cfg.JWTSecret)
	pluginSvc := service.NewPluginService(pluginStore)
	menuSvc := service.NewMenuService(pluginSvc)

	r := router.NewEngine(cfg, authSvc, pluginSvc, menuSvc)

	log.Printf("server listening on %s", cfg.ServerAddr)
	if err := r.Run(cfg.ServerAddr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
