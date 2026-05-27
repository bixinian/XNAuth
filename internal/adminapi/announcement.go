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

type AnnouncementHandler struct {
	db *gorm.DB
}

func NewAnnouncementHandler(db *gorm.DB) *AnnouncementHandler {
	return &AnnouncementHandler{db: db}
}

type announcementCreateReq struct {
	AppID      uint64  `json:"app_id" binding:"required"`
	Title      string  `json:"title" binding:"required"`
	Content    string  `json:"content" binding:"required"`
	NoticeType *string `json:"notice_type"`
	Popup      *int    `json:"popup"`
	Enabled    *int    `json:"enabled"`
	StartAt    *string `json:"start_at"`
	EndAt      *string `json:"end_at"`
	SortOrder  *int    `json:"sort_order"`
}

type announcementUpdateReq struct {
	Title      *string `json:"title"`
	Content    *string `json:"content"`
	NoticeType *string `json:"notice_type"`
	Popup      *int    `json:"popup"`
	Enabled    *int    `json:"enabled"`
	StartAt    *string `json:"start_at"`
	EndAt      *string `json:"end_at"`
	SortOrder  *int    `json:"sort_order"`
}

func (h *AnnouncementHandler) List(c *gin.Context) {
	page := pagination.FromQuery(c)
	query := h.db.Model(&model.AppAnnouncement{})

	if appID := strings.TrimSpace(c.Query("app_id")); appID != "" {
		query = query.Where("app_id = ?", appID)
	}
	if enabled := strings.TrimSpace(c.Query("enabled")); enabled != "" {
		query = query.Where("enabled = ?", enabled)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("title LIKE ?", "%"+keyword+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	var list []model.AppAnnouncement
	if err := query.Order("sort_order DESC, id DESC").Limit(page.PageSize).Offset(page.Offset).Find(&list).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	response.PageOK(c, list, page.Page, page.PageSize, total)
}

func (h *AnnouncementHandler) Create(c *gin.Context) {
	var req announcementCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}
	if !h.appExists(req.AppID) {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "app_not_found")
		return
	}

	startAt, endAt, ok := announcementParseTimeRange(c, req.StartAt, req.EndAt)
	if !ok {
		return
	}

	now := time.Now()
	record := model.AppAnnouncement{
		AppID:      req.AppID,
		Title:      strings.TrimSpace(req.Title),
		Content:    strings.TrimSpace(req.Content),
		NoticeType: announcementStringPtrOr(req.NoticeType, "normal"),
		Popup:      announcementIntOr(req.Popup, 0),
		Enabled:    announcementIntOr(req.Enabled, 1),
		StartAt:    startAt,
		EndAt:      endAt,
		SortOrder:  announcementIntOr(req.SortOrder, 0),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if record.Title == "" || record.Content == "" {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}
	if err := h.db.Create(&record).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	operationlog.Write(h.db, c, "announcement", "create", &record.ID, record)
	response.OK(c, record)
}

func (h *AnnouncementHandler) Detail(c *gin.Context) {
	record, ok := h.findByID(c)
	if !ok {
		return
	}
	response.OK(c, record)
}

func (h *AnnouncementHandler) Update(c *gin.Context) {
	record, ok := h.findByID(c)
	if !ok {
		return
	}

	var req announcementUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}

	updates := map[string]any{"updated_at": time.Now()}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
			return
		}
		updates["title"] = title
	}
	if req.Content != nil {
		content := strings.TrimSpace(*req.Content)
		if content == "" {
			response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
			return
		}
		updates["content"] = content
	}
	if req.NoticeType != nil {
		updates["notice_type"] = announcementStringPtrOr(req.NoticeType, "normal")
	}
	if req.Popup != nil {
		updates["popup"] = *req.Popup
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if req.StartAt != nil {
		parsed, err := utils.ParseOptionalTime(req.StartAt)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid start_at")
			return
		}
		updates["start_at"] = parsed
	}
	if req.EndAt != nil {
		parsed, err := utils.ParseOptionalTime(req.EndAt)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid end_at")
			return
		}
		updates["end_at"] = parsed
	}

	if err := h.db.Model(&record).Updates(updates).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	operationlog.Write(h.db, c, "announcement", "update", &record.ID, updates)
	h.db.First(&record, record.ID)
	response.OK(c, record)
}

func (h *AnnouncementHandler) Delete(c *gin.Context) {
	record, ok := h.findByID(c)
	if !ok {
		return
	}
	if err := h.db.Delete(&record).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	operationlog.Write(h.db, c, "announcement", "delete", &record.ID, nil)
	response.OK(c, gin.H{"id": record.ID})
}

func (h *AnnouncementHandler) findByID(c *gin.Context) (model.AppAnnouncement, bool) {
	var record model.AppAnnouncement
	err := h.db.First(&record, "id = ?", c.Param("id")).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "announcement_not_found")
		return record, false
	}
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return record, false
	}
	return record, true
}

func (h *AnnouncementHandler) appExists(appID uint64) bool {
	var count int64
	h.db.Model(&model.App{}).Where("id = ?", appID).Count(&count)
	return count > 0
}

func announcementParseTimeRange(c *gin.Context, startValue *string, endValue *string) (*time.Time, *time.Time, bool) {
	startAt, err := utils.ParseOptionalTime(startValue)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid start_at")
		return nil, nil, false
	}
	endAt, err := utils.ParseOptionalTime(endValue)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid end_at")
		return nil, nil, false
	}
	return startAt, endAt, true
}

func announcementStringPtrOr(value *string, fallback string) *string {
	if value == nil {
		return &fallback
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return &fallback
	}
	return &trimmed
}

func announcementIntOr(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}
