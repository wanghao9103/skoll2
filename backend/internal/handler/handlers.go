package handler

import (
	"net/http"

	"skoll2/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	authSvc   *service.AuthService
	pluginSvc *service.PluginService
	menuSvc   *service.MenuService
}

func New(authSvc *service.AuthService, pluginSvc *service.PluginService, menuSvc *service.MenuService) *Handler {
	return &Handler{
		authSvc:   authSvc,
		pluginSvc: pluginSvc,
		menuSvc:   menuSvc,
	}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": gin.H{"status": "up"}})
}

func (h *Handler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	res, err := h.authSvc.Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": res})
}

func (h *Handler) ListPlugins(c *gin.Context) {
	items := h.pluginSvc.List()
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": items})
}

func (h *Handler) InstallPlugin(c *gin.Context) {
	var req service.InstallPluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	item, err := h.pluginSvc.Install(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": item})
}

func (h *Handler) EnablePlugin(c *gin.Context) {
	var req service.TogglePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	item, err := h.pluginSvc.Enable(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": item})
}

func (h *Handler) DisablePlugin(c *gin.Context) {
	var req service.TogglePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	item, err := h.pluginSvc.Disable(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": item})
}

func (h *Handler) UpgradePlugin(c *gin.Context) {
	var req service.UpgradePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	item, err := h.pluginSvc.Upgrade(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": item})
}

func (h *Handler) UninstallPlugin(c *gin.Context) {
	var req service.TogglePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.pluginSvc.Uninstall(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": gin.H{"pluginKey": req.PluginKey}})
}

func (h *Handler) GetPluginConfig(c *gin.Context) {
	pluginKey := c.Query("pluginKey")
	configs, err := h.pluginSvc.GetConfig(pluginKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": configs})
}

func (h *Handler) SavePluginConfig(c *gin.Context) {
	var req service.SavePluginConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.pluginSvc.SaveConfig(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": gin.H{"pluginKey": req.PluginKey}})
}

func (h *Handler) Menus(c *gin.Context) {
	username, _ := c.Get("username")
	role, _ := c.Get("role")

	u, _ := username.(string)
	r, _ := role.(string)

	menus := h.menuSvc.BuildMenus(u, r)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": menus})
}
