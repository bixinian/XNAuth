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

type DeviceHandler struct {
	db *gorm.DB
}

func NewDeviceHandler(db *gorm.DB) *DeviceHandler {
	return &DeviceHandler{db: db}
}

type deviceStatusReq struct {
	Status int `json:"status" binding:"required"`
}

func (h *DeviceHandler) List(c *gin.Context) {
	page := pagination.FromQuery(c)
	query := h.db.Model(&model.LicenseDevice{})

	if appID := strings.TrimSpace(c.Query("app_id")); appID != "" {
		query = query.Where("app_id = ?", appID)
	}
	if licenseID := strings.TrimSpace(c.Query("license_id")); licenseID != "" {
		query = query.Where("license_id = ?", licenseID)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if machineCodeHash := strings.TrimSpace(c.Query("machine_code_hash")); machineCodeHash != "" {
		query = query.Where("machine_code_hash LIKE ?", "%"+machineCodeHash+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	var list []model.LicenseDevice
	if err := query.Order("id DESC").Limit(page.PageSize).Offset(page.Offset).Find(&list).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	response.PageOK(c, list, page.Page, page.PageSize, total)
}

func (h *DeviceHandler) Detail(c *gin.Context) {
	record, ok := h.findByID(c)
	if !ok {
		return
	}
	response.OK(c, record)
}

func (h *DeviceHandler) UpdateStatus(c *gin.Context) {
	record, ok := h.findByID(c)
	if !ok {
		return
	}

	var req deviceStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}
	if req.Status != model.DeviceStatusNormal && req.Status != model.DeviceStatusDisabled {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid status")
		return
	}

	updates := map[string]any{
		"status":     req.Status,
		"updated_at": time.Now(),
	}
	if err := h.db.Model(&record).Updates(updates).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	operationlog.Write(h.db, c, "device", "status", &record.ID, updates)
	h.db.First(&record, record.ID)
	response.OK(c, record)
}

func (h *DeviceHandler) Unbind(c *gin.Context) {
	record, ok := h.findByID(c)
	if !ok {
		return
	}

	now := time.Now()
	reason := "device unbound"
	updates := map[string]any{
		"status":              model.DeviceStatusUnbound,
		"device_public_key":   nil,
		"device_key_bound_at": nil,
		"updated_at":          now,
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		// 解绑只作用于当前设备记录。不要清理卡密级别的数据，否则同一卡密下其他设备会失效。
		if err := tx.Model(&record).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Model(&model.LicenseSession{}).
			Where("device_id = ? AND status = ?", record.ID, model.SessionStatusOnline).
			Updates(map[string]any{
				"status":        model.SessionStatusRevoked,
				"revoked_at":    now,
				"revoke_reason": reason,
				"updated_at":    now,
			}).Error
	})
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	operationlog.Write(h.db, c, "device", "unbind", &record.ID, updates)
	h.db.First(&record, record.ID)
	response.OK(c, record)
}

func (h *DeviceHandler) findByID(c *gin.Context) (model.LicenseDevice, bool) {
	var record model.LicenseDevice
	err := h.db.First(&record, "id = ?", c.Param("id")).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "device_not_found")
		return record, false
	}
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return record, false
	}
	return record, true
}
