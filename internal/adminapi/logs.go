package adminapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"xnauth/internal/model"
	"xnauth/pkg/pagination"
	"xnauth/pkg/response"
)

type AuditLogHandler struct {
	db *gorm.DB
}

func NewAuditLogHandler(db *gorm.DB) *AuditLogHandler {
	return &AuditLogHandler{db: db}
}

func (h *AuditLogHandler) VerifyLogs(c *gin.Context) {
	page := pagination.FromQuery(c)
	query := h.db.Model(&model.VerifyLog{})

	if appID := strings.TrimSpace(c.Query("app_id")); appID != "" {
		query = query.Where("app_id = ?", appID)
	}
	if licenseID := strings.TrimSpace(c.Query("license_id")); licenseID != "" {
		query = query.Where("license_id = ?", licenseID)
	}
	if result := strings.TrimSpace(c.Query("result")); result != "" {
		query = query.Where("result = ?", result)
	}
	if failReason := strings.TrimSpace(c.Query("fail_reason")); failReason != "" {
		query = query.Where("fail_reason = ?", failReason)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(
			"license_key LIKE ? OR machine_code_hash LIKE ? OR fail_reason LIKE ? OR client_ip LIKE ? OR client_version LIKE ?",
			like,
			like,
			like,
			like,
			like,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	var list []model.VerifyLog
	if err := query.Order("id DESC").Limit(page.PageSize).Offset(page.Offset).Find(&list).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}
	response.PageOK(c, list, page.Page, page.PageSize, total)
}

func (h *AuditLogHandler) OperationLogs(c *gin.Context) {
	page := pagination.FromQuery(c)
	query := h.db.Model(&model.OperationLog{})

	if adminID := strings.TrimSpace(c.Query("admin_id")); adminID != "" {
		query = query.Where("admin_id = ?", adminID)
	}
	if module := strings.TrimSpace(c.Query("module")); module != "" {
		query = query.Where("module = ?", module)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(
			"module LIKE ? OR action LIKE ? OR client_ip LIKE ? OR content LIKE ? OR CAST(target_id AS CHAR) LIKE ?",
			like,
			like,
			like,
			like,
			like,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	var list []model.OperationLog
	if err := query.Order("id DESC").Limit(page.PageSize).Offset(page.Offset).Find(&list).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}
	response.PageOK(c, list, page.Page, page.PageSize, total)
}
