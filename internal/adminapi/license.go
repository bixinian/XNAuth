package adminapi

import (
	"encoding/json"
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
	"xnauth/pkg/utils"
)

type LicenseHandler struct {
	db                    *gorm.DB
	sessionTimeoutSeconds int
}

func NewLicenseHandler(db *gorm.DB, sessionTimeoutSeconds ...int) *LicenseHandler {
	timeout := 90
	if len(sessionTimeoutSeconds) > 0 && sessionTimeoutSeconds[0] > 0 {
		timeout = sessionTimeoutSeconds[0]
	}
	return &LicenseHandler{db: db, sessionTimeoutSeconds: timeout}
}

type licenseCreateReq struct {
	AppID      uint64  `json:"app_id" binding:"required"`
	LicenseKey string  `json:"license_key"`
	Status     *int    `json:"status"`
	Remark     *string `json:"remark"`
	MaxDevices *int    `json:"max_devices"`
	MaxOnline  *int    `json:"max_online"`
	ExpireAt   *string `json:"expire_at"`
}

type licenseUpdateReq struct {
	Remark     *string `json:"remark"`
	MaxDevices *int    `json:"max_devices"`
	MaxOnline  *int    `json:"max_online"`
	ExpireAt   *string `json:"expire_at"`

	// ExpireAtSet 用于区分“未提交过期时间字段”和“明确清空过期时间”。
	ExpireAtSet bool `json:"-"`
}

type licenseStatusReq struct {
	Status int `json:"status" binding:"required"`
}

type licenseListItem struct {
	model.LicenseCard
	CurrentOnline int64 `json:"current_online"`
}

func (r *licenseUpdateReq) UnmarshalJSON(data []byte) error {
	type licenseUpdateReqAlias licenseUpdateReq
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var parsed licenseUpdateReqAlias
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	*r = licenseUpdateReq(parsed)
	_, r.ExpireAtSet = raw["expire_at"]
	return nil
}

func (h *LicenseHandler) List(c *gin.Context) {
	page := pagination.FromQuery(c)
	query := h.db.Model(&model.LicenseCard{})

	if appID := strings.TrimSpace(c.Query("app_id")); appID != "" {
		query = query.Where("app_id = ?", appID)
	}
	if licenseKey := strings.TrimSpace(c.Query("license_key")); licenseKey != "" {
		query = query.Where("license_key LIKE ?", "%"+licenseKey+"%")
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	var list []model.LicenseCard
	if err := query.Order("id DESC").Limit(page.PageSize).Offset(page.Offset).Find(&list).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	response.PageOK(c, h.withCurrentOnline(list), page.Page, page.PageSize, total)
}

func (h *LicenseHandler) Create(c *gin.Context) {
	var req licenseCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}
	if !h.appExists(req.AppID) {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "app_not_found")
		return
	}

	licenseKey := strings.TrimSpace(req.LicenseKey)
	if licenseKey == "" {
		generated, err := utils.RandomLicenseKey()
		if err != nil {
			response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
			return
		}
		licenseKey = generated
	}

	expireAt, err := utils.ParseOptionalTime(req.ExpireAt)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid expire_at")
		return
	}

	now := time.Now()
	maxDevices := licenseMinOne(licenseValueOr(req.MaxDevices, 1))
	record := model.LicenseCard{
		AppID:      req.AppID,
		LicenseKey: licenseKey,
		Status:     licenseValueOr(req.Status, model.LicenseStatusInactive),
		Remark:     licenseCleanStringPtr(req.Remark),
		MaxDevices: maxDevices,
		MaxOnline:  maxDevices,
		ExpireAt:   expireAt,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := h.db.Create(&record).Error; err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "license_key_exists_or_invalid")
		return
	}

	operationlog.Write(h.db, c, "license", "create", &record.ID, record)
	response.OK(c, record)
}

func (h *LicenseHandler) Detail(c *gin.Context) {
	record, ok := h.findByID(c)
	if !ok {
		return
	}
	response.OK(c, record)
}

func (h *LicenseHandler) Update(c *gin.Context) {
	record, ok := h.findByID(c)
	if !ok {
		return
	}

	var req licenseUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}

	updates := map[string]any{"updated_at": time.Now()}
	if req.Remark != nil {
		updates["remark"] = licenseCleanStringPtr(req.Remark)
	}
	if req.MaxDevices != nil {
		maxDevices := licenseMinOne(*req.MaxDevices)
		updates["max_devices"] = maxDevices
		updates["max_online"] = maxDevices
	}
	if req.MaxOnline != nil && req.MaxDevices == nil {
		updates["max_online"] = record.MaxDevices
	}
	if req.ExpireAtSet {
		if req.ExpireAt == nil {
			updates["expire_at"] = nil
		} else {
			expireAt, err := utils.ParseOptionalTime(req.ExpireAt)
			if err != nil {
				response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid expire_at")
				return
			}
			updates["expire_at"] = expireAt
		}
	}

	if err := h.db.Model(&record).Updates(updates).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	operationlog.Write(h.db, c, "license", "update", &record.ID, updates)
	h.db.First(&record, record.ID)
	response.OK(c, record)
}

func (h *LicenseHandler) UpdateStatus(c *gin.Context) {
	record, ok := h.findByID(c)
	if !ok {
		return
	}

	var req licenseStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}
	if !licenseValidStatus(req.Status) {
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

	operationlog.Write(h.db, c, "license", "status", &record.ID, updates)
	h.db.First(&record, record.ID)
	response.OK(c, record)
}

func (h *LicenseHandler) Delete(c *gin.Context) {
	record, ok := h.findByID(c)
	if !ok {
		return
	}

	if err := h.db.Delete(&record).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	operationlog.Write(h.db, c, "license", "delete", &record.ID, record)
	response.OK(c, gin.H{"id": record.ID})
}

func (h *LicenseHandler) DeleteExpired(c *gin.Context) {
	now := time.Now()
	query := h.db.Where("status = ? OR (expire_at IS NOT NULL AND expire_at < ?)", model.LicenseStatusExpired, now)
	if appID := strings.TrimSpace(c.Query("app_id")); appID != "" {
		query = query.Where("app_id = ?", appID)
	}

	result := query.Delete(&model.LicenseCard{})
	if result.Error != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	operationlog.Write(h.db, c, "license", "cleanup_expired", nil, gin.H{
		"deleted": result.RowsAffected,
		"app_id":  strings.TrimSpace(c.Query("app_id")),
	})
	response.OK(c, gin.H{"deleted": result.RowsAffected})
}

func (h *LicenseHandler) findByID(c *gin.Context) (model.LicenseCard, bool) {
	var record model.LicenseCard
	err := h.db.First(&record, "id = ?", c.Param("id")).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "license_not_found")
		return record, false
	}
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return record, false
	}
	return record, true
}

func (h *LicenseHandler) withCurrentOnline(cards []model.LicenseCard) []licenseListItem {
	items := make([]licenseListItem, 0, len(cards))
	if len(cards) == 0 {
		return items
	}

	ids := make([]uint64, 0, len(cards))
	for _, card := range cards {
		ids = append(ids, card.ID)
	}

	type onlineCount struct {
		LicenseID uint64
		Count     int64
	}

	cutoff := time.Now().Add(-time.Duration(h.sessionTimeoutSeconds) * time.Second)
	// 当前在线数根据心跳时间窗口实时计算，不维护单独计数器。
	// 即使客户端没有正常离线，过期会话超出窗口后也不会继续计入在线。
	var counts []onlineCount
	err := h.db.Model(&model.LicenseSession{}).
		Select("license_id, COUNT(*) AS count").
		Where("license_id IN ? AND status = ? AND last_heartbeat_at >= ?", ids, model.SessionStatusOnline, cutoff).
		Group("license_id").
		Scan(&counts).Error
	countByLicense := map[uint64]int64{}
	if err == nil {
		for _, item := range counts {
			countByLicense[item.LicenseID] = item.Count
		}
	}

	for _, card := range cards {
		items = append(items, licenseListItem{
			LicenseCard:   card,
			CurrentOnline: countByLicense[card.ID],
		})
	}
	return items
}

func (h *LicenseHandler) appExists(appID uint64) bool {
	var count int64
	h.db.Model(&model.App{}).Where("id = ?", appID).Count(&count)
	return count > 0
}

func licenseCleanStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func licenseValueOr(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func licenseMinOne(value int) int {
	if value < 1 {
		return 1
	}
	return value
}

func licenseValidStatus(status int) bool {
	switch status {
	case model.LicenseStatusInactive,
		model.LicenseStatusActive,
		model.LicenseStatusExpired,
		model.LicenseStatusFrozen,
		model.LicenseStatusBanned:
		return true
	default:
		return false
	}
}
