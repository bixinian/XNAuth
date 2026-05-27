package adminapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"xnauth/internal/model"
	"xnauth/internal/operationlog"
	"xnauth/pkg/pagination"
	"xnauth/pkg/response"
)

type SessionHandler struct {
	db                    *gorm.DB
	sessionTimeoutSeconds int
}

func NewSessionHandler(db *gorm.DB, sessionTimeoutSeconds ...int) *SessionHandler {
	timeout := 90
	if len(sessionTimeoutSeconds) > 0 && sessionTimeoutSeconds[0] > 0 {
		timeout = sessionTimeoutSeconds[0]
	}
	return &SessionHandler{db: db, sessionTimeoutSeconds: timeout}
}

type sessionRevokeReq struct {
	Reason string `json:"reason"`
}

func (h *SessionHandler) List(c *gin.Context) {
	page := pagination.FromQuery(c)
	h.markTimedOutSessions()

	query := h.db.Model(&model.LicenseSession{})

	if appID := strings.TrimSpace(c.Query("app_id")); appID != "" {
		query = query.Where("app_id = ?", appID)
	}
	if licenseID := strings.TrimSpace(c.Query("license_id")); licenseID != "" {
		query = query.Where("license_id = ?", licenseID)
	}
	if deviceID := strings.TrimSpace(c.Query("device_id")); deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	var list []model.LicenseSession
	if err := query.Order("id DESC").Limit(page.PageSize).Offset(page.Offset).Find(&list).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	response.PageOK(c, list, page.Page, page.PageSize, total)
}

func (h *SessionHandler) markTimedOutSessions() {
	cutoff := time.Now().Add(-time.Duration(h.sessionTimeoutSeconds) * time.Second)
	_ = h.db.Model(&model.LicenseSession{}).
		Where("status = ? AND last_heartbeat_at < ?", model.SessionStatusOnline, cutoff).
		Updates(map[string]any{"status": model.SessionStatusTimeout, "updated_at": time.Now()}).Error
}

func (h *SessionHandler) Revoke(c *gin.Context) {
	var record model.LicenseSession
	err := h.db.First(&record, "id = ?", c.Param("id")).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "session_not_found")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	var req sessionRevokeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}

	now := time.Now()
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "admin revoked"
	}
	updates := map[string]any{
		"status":        model.SessionStatusRevoked,
		"revoked_at":    now,
		"revoke_reason": reason,
		"updated_at":    now,
	}
	if err := h.db.Model(&record).Updates(updates).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	operationlog.Write(h.db, c, "session", "revoke", &record.ID, updates)
	h.db.First(&record, record.ID)
	response.OK(c, record)
}
