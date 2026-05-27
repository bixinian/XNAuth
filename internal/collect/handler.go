package collect

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

const (
	statTypeDistribution = "distribution"
	statTypeSum          = "sum"
)

type Handler struct {
	db                    *gorm.DB
	sessionTimeoutSeconds int
}

func NewHandler(db *gorm.DB, sessionTimeoutSeconds ...int) *Handler {
	timeout := 90
	if len(sessionTimeoutSeconds) > 0 && sessionTimeoutSeconds[0] > 0 {
		timeout = sessionTimeoutSeconds[0]
	}
	return &Handler{db: db, sessionTimeoutSeconds: timeout}
}

type fieldCreateReq struct {
	AppID         uint64 `json:"app_id" binding:"required"`
	FieldKey      string `json:"field_key" binding:"required"`
	FieldName     string `json:"field_name" binding:"required"`
	Enabled       *int   `json:"enabled"`
	ShowInList    *int   `json:"show_in_list"`
	StatEnabled   *int   `json:"stat_enabled"`
	StatType      string `json:"stat_type"`
	SearchEnabled *int   `json:"search_enabled"`
	SortOrder     *int   `json:"sort_order"`
}

type fieldUpdateReq struct {
	FieldName     *string `json:"field_name"`
	Enabled       *int    `json:"enabled"`
	ShowInList    *int    `json:"show_in_list"`
	StatEnabled   *int    `json:"stat_enabled"`
	StatType      *string `json:"stat_type"`
	SearchEnabled *int    `json:"search_enabled"`
	SortOrder     *int    `json:"sort_order"`
}

type statsResp struct {
	Source string      `json:"source"`
	Fields []statField `json:"fields"`
}

type statField struct {
	FieldID    uint64     `json:"field_id"`
	AppID      uint64     `json:"app_id"`
	FieldKey   string     `json:"field_key"`
	FieldName  string     `json:"field_name"`
	StatType   string     `json:"stat_type"`
	TotalCount int64      `json:"total_count"`
	NumericSum float64    `json:"numeric_sum,omitempty"`
	Items      []statItem `json:"items"`
	NumberNote string     `json:"number_note,omitempty"`
}

type statItem struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Count int64   `json:"count,omitempty"`
}

func (h *Handler) ListFields(c *gin.Context) {
	page := pagination.FromQuery(c)
	query := h.db.Model(&model.CollectField{})

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

	var list []model.CollectField
	if err := query.Order("sort_order DESC, id DESC").Limit(page.PageSize).Offset(page.Offset).Find(&list).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	response.PageOK(c, list, page.Page, page.PageSize, total)
}

func (h *Handler) CreateField(c *gin.Context) {
	var req fieldCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}
	if !h.appExists(req.AppID) {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "app_not_found")
		return
	}

	now := time.Now()
	record := model.CollectField{
		AppID:         req.AppID,
		FieldKey:      strings.TrimSpace(req.FieldKey),
		FieldName:     strings.TrimSpace(req.FieldName),
		Enabled:       intOr(req.Enabled, 1),
		ShowInList:    intOr(req.ShowInList, 0),
		StatEnabled:   intOr(req.StatEnabled, 0),
		StatType:      normalizeStatType(req.StatType),
		SearchEnabled: intOr(req.SearchEnabled, 0),
		SortOrder:     intOr(req.SortOrder, 0),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if record.FieldKey == "" || record.FieldName == "" {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}
	if err := h.db.Create(&record).Error; err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "field_key_exists_or_invalid")
		return
	}

	operationlog.Write(h.db, c, "collect_field", "create", &record.ID, record)
	response.OK(c, record)
}

func (h *Handler) UpdateField(c *gin.Context) {
	record, ok := h.findFieldByID(c)
	if !ok {
		return
	}

	var req fieldUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}

	updates := map[string]any{"updated_at": time.Now()}
	if req.FieldName != nil {
		name := strings.TrimSpace(*req.FieldName)
		if name == "" {
			response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
			return
		}
		updates["field_name"] = name
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.ShowInList != nil {
		updates["show_in_list"] = *req.ShowInList
	}
	if req.StatEnabled != nil {
		updates["stat_enabled"] = *req.StatEnabled
	}
	if req.StatType != nil {
		updates["stat_type"] = normalizeStatType(*req.StatType)
	}
	if req.SearchEnabled != nil {
		updates["search_enabled"] = *req.SearchEnabled
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}

	if err := h.db.Model(&record).Updates(updates).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	operationlog.Write(h.db, c, "collect_field", "update", &record.ID, updates)
	h.db.First(&record, record.ID)
	response.OK(c, record)
}

func (h *Handler) DeleteField(c *gin.Context) {
	record, ok := h.findFieldByID(c)
	if !ok {
		return
	}
	if err := h.db.Delete(&record).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	operationlog.Write(h.db, c, "collect_field", "delete", &record.ID, nil)
	response.OK(c, gin.H{"id": record.ID})
}

func (h *Handler) Stats(c *gin.Context) {
	source := normalizeStatsSource(c.Query("source"))
	h.markTimedOutSessions()

	query := h.db.Model(&model.CollectField{}).Where("enabled = 1 AND stat_enabled = 1")
	if appID := strings.TrimSpace(c.Query("app_id")); appID != "" {
		query = query.Where("app_id = ?", appID)
	}

	var fields []model.CollectField
	if err := query.Order("sort_order DESC, id DESC").Find(&fields).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	result := statsResp{
		Source: source,
		Fields: make([]statField, 0, len(fields)),
	}
	for _, field := range fields {
		item, err := h.fieldStats(field, source, c.Query("app_id"))
		if err != nil {
			response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
			return
		}
		result.Fields = append(result.Fields, item)
	}

	response.OK(c, result)
}

func (h *Handler) ListRecords(c *gin.Context) {
	page := pagination.FromQuery(c)
	query := h.db.Model(&model.CollectRecord{})

	if appID := strings.TrimSpace(c.Query("app_id")); appID != "" {
		query = query.Where("app_id = ?", appID)
	}
	if licenseID := strings.TrimSpace(c.Query("license_id")); licenseID != "" {
		query = query.Where("license_id = ?", licenseID)
	}
	if event := strings.TrimSpace(c.Query("event")); event != "" {
		query = query.Where("event = ?", event)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(
			"license_key LIKE ? OR machine_code_hash LIKE ? OR event LIKE ? OR client_ip LIKE ? OR user_agent LIKE ?",
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

	var list []model.CollectRecord
	if err := query.Order("id DESC").Limit(page.PageSize).Offset(page.Offset).Find(&list).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	response.PageOK(c, list, page.Page, page.PageSize, total)
}

func (h *Handler) RecordDetail(c *gin.Context) {
	record, ok := h.findRecordByID(c)
	if !ok {
		return
	}
	values := h.recordValues(record.ID)
	response.OK(c, gin.H{"record": record, "values": values})
}

func (h *Handler) LatestByLicense(c *gin.Context) {
	licenseID := c.Param("id")
	var record model.CollectRecord
	err := h.db.Where("license_id = ?", licenseID).Order("id DESC").First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.OK(c, gin.H{"record": nil, "values": []model.CollectRecordValue{}})
		return
	}
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}
	values := h.recordValues(record.ID)
	response.OK(c, gin.H{"record": record, "values": values})
}

func (h *Handler) RecordsByLicense(c *gin.Context) {
	page := pagination.FromQuery(c)
	query := h.db.Model(&model.CollectRecord{}).Where("license_id = ?", c.Param("id"))

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	var list []model.CollectRecord
	if err := query.Order("id DESC").Limit(page.PageSize).Offset(page.Offset).Find(&list).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	response.PageOK(c, list, page.Page, page.PageSize, total)
}

func (h *Handler) fieldStats(field model.CollectField, source string, appIDFilter string) (statField, error) {
	statType := normalizeStatType(field.StatType)
	result := statField{
		FieldID:   field.ID,
		AppID:     field.AppID,
		FieldKey:  field.FieldKey,
		FieldName: field.FieldName,
		StatType:  statType,
	}

	query := h.statBaseQuery(field, source, appIDFilter)
	if statType == statTypeSum {
		var row struct {
			Total float64
			Count int64
		}
		err := query.
			Where("v.field_value REGEXP ?", "^-?[0-9]+(\\.[0-9]+)?$").
			Select("COALESCE(SUM(CAST(v.field_value AS DECIMAL(20,4))), 0) AS total, COUNT(*) AS count").
			Scan(&row).Error
		if err != nil {
			return result, err
		}
		result.NumericSum = row.Total
		result.TotalCount = row.Count
		result.Items = []statItem{{Label: "累计", Value: row.Total, Count: row.Count}}
		result.NumberNote = "数额统计按数字字段累加，非数字值已忽略。"
		return result, nil
	}

	var rows []struct {
		Label string
		Count int64
	}
	if err := h.statBaseQuery(field, source, appIDFilter).Count(&result.TotalCount).Error; err != nil {
		return result, err
	}
	err := query.
		Select("COALESCE(NULLIF(v.field_value, ''), '(空)') AS label, COUNT(*) AS count").
		Group("COALESCE(NULLIF(v.field_value, ''), '(空)')").
		Order("count DESC").
		Limit(12).
		Scan(&rows).Error
	if err != nil {
		return result, err
	}
	result.Items = make([]statItem, 0, len(rows))
	for _, row := range rows {
		result.Items = append(result.Items, statItem{Label: row.Label, Value: float64(row.Count), Count: row.Count})
	}
	return result, nil
}

func (h *Handler) statBaseQuery(field model.CollectField, source string, appIDFilter string) *gorm.DB {
	query := h.db.Table("collect_record_values AS v").
		Joins("JOIN collect_records AS r ON r.id = v.record_id").
		Where("v.app_id = ? AND v.field_key = ?", field.AppID, field.FieldKey)

	if source == "all" {
		return query
	}

	cutoff := time.Now().Add(-time.Duration(h.sessionTimeoutSeconds) * time.Second)
	onlineDevices := h.db.Model(&model.LicenseSession{}).
		Select("DISTINCT device_id").
		Where("status = ? AND last_heartbeat_at >= ?", model.SessionStatusOnline, cutoff)
	if appID := strings.TrimSpace(appIDFilter); appID != "" {
		onlineDevices = onlineDevices.Where("app_id = ?", appID)
	}

	if source == "online" {
		return query.Where("r.device_id IN (?)", onlineDevices)
	}
	return query.Where("r.device_id IS NULL OR r.device_id NOT IN (?)", onlineDevices)
}

func (h *Handler) markTimedOutSessions() {
	cutoff := time.Now().Add(-time.Duration(h.sessionTimeoutSeconds) * time.Second)
	_ = h.db.Model(&model.LicenseSession{}).
		Where("status = ? AND last_heartbeat_at < ?", model.SessionStatusOnline, cutoff).
		Updates(map[string]any{"status": model.SessionStatusTimeout, "updated_at": time.Now()}).Error
}

func (h *Handler) findFieldByID(c *gin.Context) (model.CollectField, bool) {
	var record model.CollectField
	err := h.db.First(&record, "id = ?", c.Param("id")).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "collect_field_not_found")
		return record, false
	}
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return record, false
	}
	return record, true
}

func (h *Handler) findRecordByID(c *gin.Context) (model.CollectRecord, bool) {
	var record model.CollectRecord
	err := h.db.First(&record, "id = ?", c.Param("id")).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "collect_record_not_found")
		return record, false
	}
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return record, false
	}
	return record, true
}

func (h *Handler) recordValues(recordID uint64) []model.CollectRecordValue {
	var values []model.CollectRecordValue
	h.db.Where("record_id = ?", recordID).Order("id ASC").Find(&values)
	return values
}

func (h *Handler) appExists(appID uint64) bool {
	var count int64
	h.db.Model(&model.App{}).Where("id = ?", appID).Count(&count)
	return count > 0
}

func intOr(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func normalizeStatType(value string) string {
	switch strings.TrimSpace(value) {
	case statTypeSum:
		return statTypeSum
	default:
		return statTypeDistribution
	}
}

func normalizeStatsSource(value string) string {
	switch strings.TrimSpace(value) {
	case "online":
		return "online"
	case "offline":
		return "offline"
	default:
		return "all"
	}
}
