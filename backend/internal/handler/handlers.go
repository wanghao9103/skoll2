package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"skoll2/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	authSvc      *service.AuthService
	pluginSvc    *service.PluginService
	menuSvc      *service.MenuService
	pluginAPISvc *service.PluginAPIService
}

func New(authSvc *service.AuthService, pluginSvc *service.PluginService, menuSvc *service.MenuService, pluginAPISvc *service.PluginAPIService) *Handler {
	return &Handler{
		authSvc:      authSvc,
		pluginSvc:    pluginSvc,
		menuSvc:      menuSvc,
		pluginAPISvc: pluginAPISvc,
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

func (h *Handler) InstallPluginUpload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "file is required"})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".zip" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "only .zip plugin package is supported"})
		return
	}

	tmpFile, err := os.CreateTemp("", "skoll-plugin-upload-*.zip")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	_ = tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if err := c.SaveUploadedFile(file, tmpFile.Name()); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	item, err := h.pluginSvc.InstallFromZip(tmpFile.Name())
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

func (h *Handler) PluginProcessStatuses(c *gin.Context) {
	pluginKey := strings.TrimSpace(c.Query("pluginKey"))
	items := h.pluginAPISvc.ListProcessStatuses(pluginKey)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": items})
}

func (h *Handler) Menus(c *gin.Context) {
	username, _ := c.Get("username")
	role, _ := c.Get("role")

	u, _ := username.(string)
	r, _ := role.(string)

	menus := h.menuSvc.BuildMenus(u, r)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": menus})
}

func (h *Handler) PluginRouteHandler(pluginKey string, meta service.PluginRouteMeta) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := h.pluginAPISvc.Execute(c, pluginKey, meta)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
			return
		}

		if res.Passthrough {
			status := res.StatusCode
			if status == 0 {
				status = http.StatusOK
			}
			c.JSON(status, res.Body)
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": res.Body})
	}
}
