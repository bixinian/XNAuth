package adminapi

import (
	"encoding/json"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"xnauth/internal/model"
	"xnauth/internal/operationlog"
	"xnauth/internal/systemconfig"
	"xnauth/pkg/response"
)

type SystemHandler struct {
	db        *gorm.DB
	startedAt time.Time
}

func NewSystemHandler(db *gorm.DB) *SystemHandler {
	return &SystemHandler{db: db, startedAt: time.Now()}
}

type systemSecuritySettingsReq struct {
	LoginCaptchaEnabled *bool `json:"login_captcha_enabled"`
}

type systemCleanupReq struct {
	Target string `json:"target" binding:"required"`
	Before string `json:"before" binding:"required"`
}

type systemFooterLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type systemSiteSettingsReq struct {
	SiteName    *string            `json:"site_name"`
	ICPNumber   *string            `json:"icp_number"`
	FooterLinks []systemFooterLink `json:"footer_links"`
}

func (h *SystemHandler) SecuritySettings(c *gin.Context) {
	response.OK(c, gin.H{
		"login_captcha_enabled": systemconfig.Bool(h.db, systemconfig.LoginCaptchaEnabledKey, false),
	})
}

func (h *SystemHandler) UpdateSecuritySettings(c *gin.Context) {
	var req systemSecuritySettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}
	if req.LoginCaptchaEnabled == nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}
	if err := systemconfig.SetBool(h.db, systemconfig.LoginCaptchaEnabledKey, *req.LoginCaptchaEnabled); err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}
	operationlog.Write(h.db, c, "system_settings", "update_security", nil, gin.H{
		"login_captcha_enabled": *req.LoginCaptchaEnabled,
	})
	h.SecuritySettings(c)
}

func (h *SystemHandler) PublicSiteSettings(c *gin.Context) {
	response.OK(c, h.siteSettingsPayload())
}

func (h *SystemHandler) SiteSettings(c *gin.Context) {
	response.OK(c, h.siteSettingsPayload())
}

func (h *SystemHandler) UpdateSiteSettings(c *gin.Context) {
	var req systemSiteSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}
	if req.SiteName == nil || req.ICPNumber == nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}

	siteName := strings.TrimSpace(*req.SiteName)
	icpNumber := strings.TrimSpace(*req.ICPNumber)
	links := sanitizeFooterLinks(req.FooterLinks)
	linksJSON, err := json.Marshal(links)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid footer links")
		return
	}
	if siteName == "" || len([]rune(siteName)) > 40 || len([]rune(icpNumber)) > 80 {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}

	tx := h.db.Begin()
	if tx.Error != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}
	if err := systemconfig.SetValue(tx, systemconfig.SiteNameKey, siteName); err != nil {
		tx.Rollback()
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}
	if err := systemconfig.SetValue(tx, systemconfig.ICPNumberKey, icpNumber); err != nil {
		tx.Rollback()
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}
	if err := systemconfig.SetValue(tx, systemconfig.FooterLinksKey, string(linksJSON)); err != nil {
		tx.Rollback()
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}
	if err := tx.Commit().Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	operationlog.Write(h.db, c, "system_settings", "update_site", nil, gin.H{
		"site_name":    siteName,
		"icp_number":   icpNumber,
		"footer_links": links,
	})
	response.OK(c, h.siteSettingsPayload())
}

func (h *SystemHandler) Status(c *gin.Context) {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)

	dbStatus := "ok"
	dbStats := gin.H{
		"open_connections": 0,
		"in_use":           0,
		"idle":             0,
		"wait_count":       0,
	}
	if sqlDB, err := h.db.DB(); err == nil {
		if err := sqlDB.PingContext(c.Request.Context()); err != nil {
			dbStatus = "error"
		}
		stats := sqlDB.Stats()
		dbStats = gin.H{
			"open_connections": stats.OpenConnections,
			"in_use":           stats.InUse,
			"idle":             stats.Idle,
			"wait_count":       stats.WaitCount,
		}
	} else {
		dbStatus = "error"
	}

	response.OK(c, gin.H{
		"status":         "ok",
		"uptime_seconds": int64(time.Since(h.startedAt).Seconds()),
		"goroutines":     runtime.NumGoroutine(),
		"memory": gin.H{
			"alloc_mb":       bytesToMB(memory.Alloc),
			"sys_mb":         bytesToMB(memory.Sys),
			"heap_alloc_mb":  bytesToMB(memory.HeapAlloc),
			"heap_inuse_mb":  bytesToMB(memory.HeapInuse),
			"last_gc_unix":   int64(memory.LastGC / uint64(time.Second)),
			"gc_count":       memory.NumGC,
			"next_gc_mb":     bytesToMB(memory.NextGC),
			"total_alloc_mb": bytesToMB(memory.TotalAlloc),
		},
		"database": gin.H{
			"status": dbStatus,
			"stats":  dbStats,
		},
	})
}

func (h *SystemHandler) CleanupData(c *gin.Context) {
	var req systemCleanupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}
	before, err := parseCleanupTime(req.Before)
	if err != nil || !before.Before(time.Now()) {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid cleanup time")
		return
	}

	target := strings.TrimSpace(req.Target)
	switch target {
	case "collect_records":
		h.cleanupCollectRecords(c, before)
	case "operation_logs":
		h.cleanupOperationLogs(c, before)
	default:
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid cleanup target")
	}
}

func (h *SystemHandler) cleanupCollectRecords(c *gin.Context, before time.Time) {
	var recordCount int64
	if err := h.db.Model(&model.CollectRecord{}).Where("created_at < ?", before).Count(&recordCount).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	tx := h.db.Begin()
	if tx.Error != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}
	values := tx.Exec("DELETE FROM collect_record_values WHERE record_id IN (SELECT id FROM collect_records WHERE created_at < ?)", before)
	if values.Error != nil {
		tx.Rollback()
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}
	records := tx.Where("created_at < ?", before).Delete(&model.CollectRecord{})
	if records.Error != nil {
		tx.Rollback()
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}
	if err := tx.Commit().Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	result := gin.H{
		"target":          "collect_records",
		"before":          before.Format(time.RFC3339),
		"deleted_records": records.RowsAffected,
		"deleted_values":  values.RowsAffected,
		"matched_records": recordCount,
	}
	operationlog.Write(h.db, c, "data_cleanup", "collect_records", nil, result)
	response.OK(c, result)
}

func (h *SystemHandler) cleanupOperationLogs(c *gin.Context, before time.Time) {
	var matched int64
	if err := h.db.Model(&model.OperationLog{}).Where("created_at < ?", before).Count(&matched).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}
	deleted := h.db.Where("created_at < ?", before).Delete(&model.OperationLog{})
	if deleted.Error != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	result := gin.H{
		"target":          "operation_logs",
		"before":          before.Format(time.RFC3339),
		"deleted_records": deleted.RowsAffected,
		"matched_records": matched,
	}
	operationlog.Write(h.db, c, "data_cleanup", "operation_logs", nil, result)
	response.OK(c, result)
}

func parseCleanupTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	if parsed, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
		return parsed, nil
	}
	return time.ParseInLocation("2006-01-02", value, time.Local)
}

func bytesToMB(value uint64) float64 {
	return float64(value) / 1024 / 1024
}

func (h *SystemHandler) siteSettingsPayload() gin.H {
	return gin.H{
		"site_name":    systemconfig.String(h.db, systemconfig.SiteNameKey, "XNAuth 汐念验证"),
		"icp_number":   systemconfig.String(h.db, systemconfig.ICPNumberKey, ""),
		"footer_links": h.systemFooterLinks(),
	}
}

func (h *SystemHandler) systemFooterLinks() []systemFooterLink {
	raw := systemconfig.String(h.db, systemconfig.FooterLinksKey, "")
	if raw == "" {
		return defaultFooterLinks()
	}
	var links []systemFooterLink
	if err := json.Unmarshal([]byte(raw), &links); err != nil {
		return defaultFooterLinks()
	}
	return sanitizeFooterLinks(links)
}

func sanitizeFooterLinks(links []systemFooterLink) []systemFooterLink {
	// 已保存的空列表是有效配置：管理员可能希望不展示任何公开快捷入口。
	// 只有配置缺失或无法解析时才回退到默认入口。
	result := make([]systemFooterLink, 0, len(links))
	for _, item := range links {
		label := strings.TrimSpace(item.Label)
		url := strings.TrimSpace(item.URL)
		if label == "" || url == "" {
			continue
		}
		if len([]rune(label)) > 24 || len([]rune(url)) > 200 {
			continue
		}
		if !strings.HasPrefix(url, "/") && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			continue
		}
		result = append(result, systemFooterLink{Label: label, URL: url})
		if len(result) >= 6 {
			break
		}
	}
	return result
}

func defaultFooterLinks() []systemFooterLink {
	return []systemFooterLink{
		{Label: "进入后台", URL: "/login"},
		{Label: "接口健康", URL: "/api/health"},
	}
}
