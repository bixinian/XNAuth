package adminapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"xnauth/internal/model"
	"xnauth/pkg/response"
)

type DashboardHandler struct {
	db                    *gorm.DB
	sessionTimeoutSeconds int
}

type trendRow struct {
	Day     string `gorm:"column:day"`
	Total   int64  `gorm:"column:total"`
	Success int64  `gorm:"column:success"`
	Failed  int64  `gorm:"column:failed"`
}

type countRow struct {
	Day   string `gorm:"column:day"`
	Total int64  `gorm:"column:total"`
}

func NewDashboardHandler(db *gorm.DB, sessionTimeoutSeconds int) *DashboardHandler {
	if sessionTimeoutSeconds <= 0 {
		sessionTimeoutSeconds = 90
	}
	return &DashboardHandler{db: db, sessionTimeoutSeconds: sessionTimeoutSeconds}
}

func (h *DashboardHandler) Summary(c *gin.Context) {
	appID, ok := parseAppID(c)
	if !ok {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid app_id")
		return
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	cutoff := now.Add(-time.Duration(h.sessionTimeoutSeconds) * time.Second)

	apps := h.count(scopeModel(h.db.Model(&model.App{}), appID, "id"))
	activeApps := h.count(scopeModel(h.db.Model(&model.App{}).Where("status = ?", model.AppStatusEnabled), appID, "id"))
	licenses := h.count(scopeModel(h.db.Model(&model.LicenseCard{}), appID, "app_id"))
	activeLicenses := h.count(scopeModel(h.db.Model(&model.LicenseCard{}).Where("status = ?", model.LicenseStatusActive), appID, "app_id"))
	devices := h.count(scopeModel(h.db.Model(&model.LicenseDevice{}).Where("status = ?", model.DeviceStatusNormal), appID, "app_id"))
	onlineSessions := h.count(scopeModel(
		h.db.Model(&model.LicenseSession{}).
			Where("status = ? AND last_heartbeat_at >= ?", model.SessionStatusOnline, cutoff),
		appID,
		"app_id",
	))
	collectRecords := h.count(scopeModel(h.db.Model(&model.CollectRecord{}), appID, "app_id"))
	todayVerify := h.count(scopeModel(h.db.Model(&model.VerifyLog{}).Where("created_at >= ?", today), appID, "app_id"))
	todaySuccess := h.count(scopeModel(
		h.db.Model(&model.VerifyLog{}).Where("created_at >= ? AND result = ?", today, 1),
		appID,
		"app_id",
	))
	todayFailed := h.count(scopeModel(
		h.db.Model(&model.VerifyLog{}).Where("created_at >= ? AND result <> ?", today, 1),
		appID,
		"app_id",
	))

	response.OK(c, gin.H{
		"scope": gin.H{
			"app_id":       nullableAppID(appID),
			"generated_at": now.Format(time.RFC3339),
		},
		"metrics": gin.H{
			"apps":                apps,
			"active_apps":         activeApps,
			"licenses":            licenses,
			"active_licenses":     activeLicenses,
			"devices":             devices,
			"online_sessions":     onlineSessions,
			"collect_records":     collectRecords,
			"verify_today":        todayVerify,
			"verify_success":      todaySuccess,
			"verify_failed":       todayFailed,
			"verify_success_rate": successRate(todaySuccess, todayVerify),
		},
		"trend":         h.verifyTrend(appID, today.AddDate(0, 0, -6), today),
		"session_trend": h.sessionTrend(appID, today.AddDate(0, 0, -6), today),
	})
}

func parseAppID(c *gin.Context) (uint64, bool) {
	value := c.Query("app_id")
	if value == "" {
		return 0, true
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func nullableAppID(appID uint64) *uint64 {
	if appID == 0 {
		return nil
	}
	return &appID
}

func scopeModel(query *gorm.DB, appID uint64, column string) *gorm.DB {
	if appID == 0 {
		return query
	}
	return query.Where(column+" = ?", appID)
}

func (h *DashboardHandler) count(query *gorm.DB) int64 {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0
	}
	return total
}

func successRate(success int64, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(success) * 100 / float64(total)
}

func (h *DashboardHandler) verifyTrend(appID uint64, start time.Time, today time.Time) []gin.H {
	query := h.db.Model(&model.VerifyLog{}).
		Select("DATE_FORMAT(created_at, '%Y-%m-%d') AS day, COUNT(*) AS total, SUM(CASE WHEN result = 1 THEN 1 ELSE 0 END) AS success, SUM(CASE WHEN result <> 1 THEN 1 ELSE 0 END) AS failed").
		Where("created_at >= ?", start).
		Group("DATE_FORMAT(created_at, '%Y-%m-%d')").
		Order("day ASC")
	query = scopeModel(query, appID, "app_id")

	var rows []trendRow
	if err := query.Scan(&rows).Error; err != nil {
		rows = nil
	}
	rowByDay := make(map[string]trendRow, len(rows))
	for _, row := range rows {
		rowByDay[row.Day] = row
	}

	result := make([]gin.H, 0, 7)
	for i := 0; i < 7; i++ {
		day := start.AddDate(0, 0, i)
		key := day.Format("2006-01-02")
		row := rowByDay[key]
		result = append(result, gin.H{
			"date":    key,
			"label":   day.Format("01-02"),
			"total":   row.Total,
			"success": row.Success,
			"failed":  row.Failed,
			"today":   day.Equal(today),
		})
	}
	return result
}

func (h *DashboardHandler) sessionTrend(appID uint64, start time.Time, today time.Time) []gin.H {
	createdByDay := h.countByDay(
		scopeModel(
			h.db.Model(&model.LicenseSession{}).
				Select("DATE_FORMAT(created_at, '%Y-%m-%d') AS day, COUNT(*) AS total").
				Where("created_at >= ?", start).
				Group("DATE_FORMAT(created_at, '%Y-%m-%d')"),
			appID,
			"app_id",
		),
	)
	heartbeatByDay := h.countByDay(
		scopeModel(
			h.db.Model(&model.LicenseSession{}).
				Select("DATE_FORMAT(last_heartbeat_at, '%Y-%m-%d') AS day, COUNT(*) AS total").
				Where("last_heartbeat_at >= ?", start).
				Group("DATE_FORMAT(last_heartbeat_at, '%Y-%m-%d')"),
			appID,
			"app_id",
		),
	)

	result := make([]gin.H, 0, 7)
	for i := 0; i < 7; i++ {
		day := start.AddDate(0, 0, i)
		key := day.Format("2006-01-02")
		created := createdByDay[key]
		active := heartbeatByDay[key]
		result = append(result, gin.H{
			"date":    key,
			"label":   day.Format("01-02"),
			"created": created,
			"active":  active,
			"total":   created + active,
			"today":   day.Equal(today),
		})
	}
	return result
}

func (h *DashboardHandler) countByDay(query *gorm.DB) map[string]int64 {
	var rows []countRow
	if err := query.Order("day ASC").Scan(&rows).Error; err != nil {
		return map[string]int64{}
	}
	result := make(map[string]int64, len(rows))
	for _, row := range rows {
		result[row.Day] = row.Total
	}
	return result
}
