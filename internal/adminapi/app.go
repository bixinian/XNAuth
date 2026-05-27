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

type AppHandler struct {
	db *gorm.DB
}

func NewAppHandler(db *gorm.DB) *AppHandler {
	return &AppHandler{db: db}
}

type appCreateReq struct {
	AppKey                  string  `json:"app_key" binding:"required"`
	AppName                 string  `json:"app_name" binding:"required"`
	Status                  *int    `json:"status"`
	MinLoginVersionCode     *int    `json:"min_login_version_code"`
	ForceUpdate             *int    `json:"force_update"`
	SecureKeyID             *string `json:"secure_key_id"`
	SecureX25519PrivateKey  *string `json:"secure_x25519_private_key"`
	SecureX25519PublicKey   *string `json:"secure_x25519_public_key"`
	SecureEd25519PrivateKey *string `json:"secure_ed25519_private_key"`
	SecureEd25519PublicKey  *string `json:"secure_ed25519_public_key"`
	Remark                  *string `json:"remark"`
}

type appUpdateReq struct {
	AppName                 *string `json:"app_name"`
	Status                  *int    `json:"status"`
	MinLoginVersionCode     *int    `json:"min_login_version_code"`
	ForceUpdate             *int    `json:"force_update"`
	SecureKeyID             *string `json:"secure_key_id"`
	SecureX25519PrivateKey  *string `json:"secure_x25519_private_key"`
	SecureX25519PublicKey   *string `json:"secure_x25519_public_key"`
	SecureEd25519PrivateKey *string `json:"secure_ed25519_private_key"`
	SecureEd25519PublicKey  *string `json:"secure_ed25519_public_key"`
	Remark                  *string `json:"remark"`
}

func (h *AppHandler) List(c *gin.Context) {
	page := pagination.FromQuery(c)
	query := h.db.Model(&model.App{})

	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("app_key LIKE ? OR app_name LIKE ?", like, like)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	var list []model.App
	if err := query.Order("id DESC").Limit(page.PageSize).Offset(page.Offset).Find(&list).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	response.PageOK(c, list, page.Page, page.PageSize, total)
}

func (h *AppHandler) Create(c *gin.Context) {
	var req appCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}

	status := model.AppStatusEnabled
	if req.Status != nil {
		status = *req.Status
	}
	keys, err := normalizeAppSecurityKeys(appSecurityKeyInput{
		KeyID:             appValueString(req.SecureKeyID),
		X25519PrivateKey:  appValueString(req.SecureX25519PrivateKey),
		X25519PublicKey:   appValueString(req.SecureX25519PublicKey),
		Ed25519PrivateKey: appValueString(req.SecureEd25519PrivateKey),
		Ed25519PublicKey:  appValueString(req.SecureEd25519PublicKey),
	}, true)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
		return
	}
	now := time.Now()
	record := model.App{
		AppKey:               strings.TrimSpace(req.AppKey),
		AppName:              strings.TrimSpace(req.AppName),
		Status:               status,
		MinLoginVersionCode:  appCleanVersionCodePtr(req.MinLoginVersionCode),
		ForceUpdate:          appCleanBoolInt(req.ForceUpdate),
		SecureKeyID:          &keys.KeyID,
		SecureX25519Private:  &keys.X25519PrivateKey,
		SecureX25519Public:   &keys.X25519PublicKey,
		SecureEd25519Private: &keys.Ed25519PrivateKey,
		SecureEd25519Public:  &keys.Ed25519PublicKey,
		Remark:               appCleanStringPtr(req.Remark),
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if record.AppKey == "" || record.AppName == "" {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}
	if err := h.db.Create(&record).Error; err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "app_key_exists_or_invalid")
		return
	}

	operationlog.Write(h.db, c, "app", "create", &record.ID, record)
	response.OK(c, record)
}

func (h *AppHandler) Detail(c *gin.Context) {
	record, ok := h.findByID(c)
	if !ok {
		return
	}
	response.OK(c, record)
}

func (h *AppHandler) Update(c *gin.Context) {
	record, ok := h.findByID(c)
	if !ok {
		return
	}

	var req appUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}

	updates := map[string]any{"updated_at": time.Now()}
	if req.AppName != nil {
		name := strings.TrimSpace(*req.AppName)
		if name == "" {
			response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
			return
		}
		updates["app_name"] = name
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.MinLoginVersionCode != nil {
		updates["min_login_version_code"] = appCleanVersionCodePtr(req.MinLoginVersionCode)
	}
	if req.ForceUpdate != nil {
		updates["force_update"] = appCleanBoolInt(req.ForceUpdate)
	}
	if hasAppSecurityKeyInput(req) {
		keys, err := normalizeAppSecurityKeys(appSecurityKeyInput{
			KeyID:             appValueString(req.SecureKeyID),
			X25519PrivateKey:  appValueString(req.SecureX25519PrivateKey),
			X25519PublicKey:   appValueString(req.SecureX25519PublicKey),
			Ed25519PrivateKey: appValueString(req.SecureEd25519PrivateKey),
			Ed25519PublicKey:  appValueString(req.SecureEd25519PublicKey),
		}, false)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
			return
		}
		for key, value := range appSecurityKeyUpdates(keys) {
			updates[key] = value
		}
	}
	if req.Remark != nil {
		updates["remark"] = appCleanStringPtr(req.Remark)
	}

	if err := h.db.Model(&record).Updates(updates).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	operationlog.Write(h.db, c, "app", "update", &record.ID, updates)
	h.db.First(&record, record.ID)
	response.OK(c, record)
}

func (h *AppHandler) GenerateSecurityKeys(c *gin.Context) {
	record, ok := h.findByID(c)
	if !ok {
		return
	}

	keys, err := generateAppSecurityKeys()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}
	updates := appSecurityKeyUpdates(keys)
	updates["updated_at"] = time.Now()
	if err := h.db.Model(&record).Updates(updates).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	operationlog.Write(h.db, c, "app", "generate_security_keys", &record.ID, gin.H{"secure_key_id": keys.KeyID})
	h.db.First(&record, record.ID)
	response.OK(c, record)
}

func (h *AppHandler) Delete(c *gin.Context) {
	record, ok := h.findByID(c)
	if !ok {
		return
	}

	if err := h.db.Delete(&record).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	operationlog.Write(h.db, c, "app", "delete", &record.ID, nil)
	response.OK(c, gin.H{"id": record.ID})
}

func (h *AppHandler) findByID(c *gin.Context) (model.App, bool) {
	var record model.App
	err := h.db.First(&record, "id = ?", c.Param("id")).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "app_not_found")
		return record, false
	}
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return record, false
	}
	return record, true
}

func appCleanStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func appCleanVersionCodePtr(value *int) *int {
	if value == nil || *value <= 0 {
		return nil
	}
	return value
}

func appCleanBoolInt(value *int) int {
	if value != nil && *value == 1 {
		return 1
	}
	return 0
}

func appValueString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func hasAppSecurityKeyInput(req appUpdateReq) bool {
	return req.SecureKeyID != nil ||
		req.SecureX25519PrivateKey != nil ||
		req.SecureX25519PublicKey != nil ||
		req.SecureEd25519PrivateKey != nil ||
		req.SecureEd25519PublicKey != nil
}
