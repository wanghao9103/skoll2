package router

import (
	"time"

	"skoll2/backend/internal/config"
	"skoll2/backend/internal/handler"
	"skoll2/backend/internal/middleware"
	"skoll2/backend/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewEngine(cfg config.Config, authSvc *service.AuthService, pluginSvc *service.PluginService, menuSvc *service.MenuService) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	sampleHelloSvc := service.NewSampleHelloService(pluginSvc, pluginSvc.Store())
	h := handler.New(authSvc, pluginSvc, menuSvc, sampleHelloSvc)

	r.GET("/health", h.Health)

	api := r.Group("/api")
	{
		api.POST("/auth/login", h.Login)

		protected := api.Group("")
		protected.Use(middleware.JWT(authSvc))
		{
			protected.GET("/menus", h.Menus)
			protected.GET("/plugin/sample-hello/records", h.ListSampleHelloRecords)
			protected.POST("/plugin/sample-hello/records", h.CreateSampleHelloRecord)
			protected.PUT("/plugin/sample-hello/records/:id", h.UpdateSampleHelloRecord)
			protected.DELETE("/plugin/sample-hello/records/:id", h.DeleteSampleHelloRecord)

			plugin := protected.Group("/plugin")
			plugin.Use(middleware.AdminOnly())
			{
				plugin.GET("/list", h.ListPlugins)
				plugin.GET("/config", h.GetPluginConfig)
				plugin.POST("/install", h.InstallPlugin)
				plugin.POST("/config/save", h.SavePluginConfig)
				plugin.POST("/upgrade", h.UpgradePlugin)
				plugin.POST("/enable", h.EnablePlugin)
				plugin.POST("/disable", h.DisablePlugin)
				plugin.POST("/uninstall", h.UninstallPlugin)
			}
		}
	}

	_ = cfg
	return r
}
