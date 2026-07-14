package api

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ety001/multitune/internal/model"
	"github.com/gin-gonic/gin"
)

// DeviceInfoRequest 设备信息请求
type DeviceInfoRequest struct {
	UserAgent      string `json:"userAgent"`
	ChromeVersion  int    `json:"chromeVersion"`
	WebviewVersion int    `json:"webViewVersion"`
	IsWebview      bool   `json:"isWebView"`
	ScreenWidth    int    `json:"screenWidth"`
	ScreenHeight   int    `json:"screenHeight"`
	WindowWidth    int    `json:"windowWidth"`
	WindowHeight   int    `json:"windowHeight"`
	Language       string `json:"language"`
	Platform       string `json:"platform"`
	CookieEnabled  bool   `json:"cookieEnabled"`
	Online         bool   `json:"onLine"`
	Timestamp      string `json:"timestamp"`
}

// CreateDeviceLog POST /api/device-info
func (h *Handler) CreateDeviceLog(c *gin.Context) {
	var req DeviceInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	log := &model.DeviceLog{
		UserAgent:      req.UserAgent,
		ChromeVersion:  req.ChromeVersion,
		WebviewVersion: req.WebviewVersion,
		IsWebview:      req.IsWebview,
		ScreenWidth:    req.ScreenWidth,
		ScreenHeight:   req.ScreenHeight,
		WindowWidth:    req.WindowWidth,
		WindowHeight:   req.WindowHeight,
		Language:       req.Language,
		Platform:       req.Platform,
		CookieEnabled:  req.CookieEnabled,
		Online:         req.Online,
		Timestamp:      req.Timestamp,
	}

	created, err := h.deviceLogRepo.Create(log)
	if err != nil {
		slog.Error("创建设备日志失败", "error", err)
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data:    created,
	})
}

// ListDeviceLogs GET /api/device-logs
func (h *Handler) ListDeviceLogs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	logs, total, err := h.deviceLogRepo.List(limit, offset)
	if err != nil {
		slog.Error("查询设备日志失败", "error", err)
		c.JSON(http.StatusInternalServerError, model.APIResponse{
			Code:    9001,
			Message: "内部错误",
		})
		return
	}

	c.JSON(http.StatusOK, model.APIResponse{
		Code:    0,
		Message: "ok",
		Data: model.ListResponse{
			Items: logs,
			Total: total,
		},
	})
}
