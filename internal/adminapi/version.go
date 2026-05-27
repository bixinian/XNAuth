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
	"xnauth/pkg/utils"
)

type VersionHandler struct {
	db *gorm.DB
}

func NewVersionHandler(db *gorm.DB) *VersionHandler {
	return &VersionHandler{db: db}
}

type versionCreateReq struct {
	AppID            uint64  `json:"app_id" binding:"required"`
	VersionName      string  `json:"version_name" binding:"required"`
	VersionCode      int     `json:"version_code" binding:"required"`
	MinSupportedCode *int    `json:"min_supported_code"`
	DownloadURL      *string `json:"download_url"`
	FileHash         *string `json:"file_hash"`
	FileSize         *int64  `json:"file_size"`
	Changelog        *string `json:"changelog"`
	ForceUpdate      *int    `json:"force_update"`
	Enabled          *int    `json:"enabled"`
	ReleasedAt       *string `json:"released_at"`
}

type versionUpdateReq struct {
	VersionName      *string `json:"version_name"`
	VersionCode      *int    `json:"version_code"`
	MinSupportedCode *int    `json:"min_supported_code"`
	DownloadURL      *string `json:"download_url"`
	FileHash         *string `json:"file_hash"`
	FileSize         *int64  `json:"file_size"`
	Changelog        *string `json:"changelog"`
	ForceUpdate      *int    `json:"force_update"`
	Enabled          *int    `json:"enabled"`
	ReleasedAt       *string `json:"released_at"`
}

func (h *VersionHandler) List(c *gin.Context) {
	page := pagination.FromQuery(c)
	query := h.db.Model(&model.AppVersion{})

	if appID := strings.TrimSpace(c.Query("app_id")); appID != "" {
		query = query.Where("app_id = ?", appID)
	}
	if enabled := strings.TrimSpace(c.Query("enabled")); enabled != "" {
		query = query.Where("enabled = ?", enabled)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	var list []model.AppVersion
	if err := query.Order("version_code DESC, id DESC").Limit(page.PageSize).Offset(page.Offset).Find(&list).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	response.PageOK(c, list, page.Page, page.PageSize, total)
}

func (h *VersionHandler) Create(c *gin.Context) {
	var req versionCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}
	if !h.appExists(req.AppID) {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "app_not_found")
		return
	}
	releasedAt, err := utils.ParseOptionalTime(req.ReleasedAt)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid released_at")
		return
	}

	now := time.Now()
	record := model.AppVersion{
		AppID:            req.AppID,
		VersionName:      strings.TrimSpace(req.VersionName),
		VersionCode:      req.VersionCode,
		MinSupportedCode: req.MinSupportedCode,
		DownloadURL:      versionCleanStringPtr(req.DownloadURL),
		FileHash:         versionCleanStringPtr(req.FileHash),
		FileSize:         req.FileSize,
		Changelog:        versionCleanStringPtr(req.Changelog),
		ForceUpdate:      versionIntOr(req.ForceUpdate, 0),
		Enabled:          versionIntOr(req.Enabled, 1),
		ReleasedAt:       releasedAt,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if record.VersionName == "" || record.VersionCode <= 0 {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}
	if err := h.db.Create(&record).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	operationlog.Write(h.db, c, "version", "create", &record.ID, record)
	response.OK(c, record)
}

func (h *VersionHandler) Detail(c *gin.Context) {
	record, ok := h.findByID(c)
	if !ok {
		return
	}
	response.OK(c, record)
}

func (h *VersionHandler) Update(c *gin.Context) {
	record, ok := h.findByID(c)
	if !ok {
		return
	}

	var req versionUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}

	updates := map[string]any{"updated_at": time.Now()}
	if req.VersionName != nil {
		name := strings.TrimSpace(*req.VersionName)
		if name == "" {
			response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
			return
		}
		updates["version_name"] = name
	}
	if req.VersionCode != nil {
		if *req.VersionCode <= 0 {
			response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
			return
		}
		updates["version_code"] = *req.VersionCode
	}
	if req.MinSupportedCode != nil {
		updates["min_supported_code"] = req.MinSupportedCode
	}
	if req.DownloadURL != nil {
		updates["download_url"] = versionCleanStringPtr(req.DownloadURL)
	}
	if req.FileHash != nil {
		updates["file_hash"] = versionCleanStringPtr(req.FileHash)
	}
	if req.FileSize != nil {
		updates["file_size"] = req.FileSize
	}
	if req.Changelog != nil {
		updates["changelog"] = versionCleanStringPtr(req.Changelog)
	}
	if req.ForceUpdate != nil {
		updates["force_update"] = *req.ForceUpdate
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.ReleasedAt != nil {
		releasedAt, err := utils.ParseOptionalTime(req.ReleasedAt)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid released_at")
			return
		}
		updates["released_at"] = releasedAt
	}

	if err := h.db.Model(&record).Updates(updates).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	operationlog.Write(h.db, c, "version", "update", &record.ID, updates)
	h.db.First(&record, record.ID)
	response.OK(c, record)
}

func (h *VersionHandler) Delete(c *gin.Context) {
	record, ok := h.findByID(c)
	if !ok {
		return
	}
	if err := h.db.Delete(&record).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	operationlog.Write(h.db, c, "version", "delete", &record.ID, nil)
	response.OK(c, gin.H{"id": record.ID})
}

func (h *VersionHandler) findByID(c *gin.Context) (model.AppVersion, bool) {
	var record model.AppVersion
	err := h.db.First(&record, "id = ?", c.Param("id")).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "version_not_found")
		return record, false
	}
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return record, false
	}
	return record, true
}

func (h *VersionHandler) appExists(appID uint64) bool {
	var count int64
	h.db.Model(&model.App{}).Where("id = ?", appID).Count(&count)
	return count > 0
}

func versionCleanStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func versionIntOr(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}
