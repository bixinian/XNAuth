package operationlog

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"xnauth/internal/auth"
	"xnauth/internal/middleware"
	"xnauth/internal/model"
)

func Write(db *gorm.DB, c *gin.Context, module string, action string, targetID *uint64, content any) {
	if db == nil {
		return
	}

	var adminID *uint64
	if value, exists := c.Get(middleware.ContextAdminID); exists {
		if parsed, ok := auth.AdminIDFromAny(value); ok {
			adminID = &parsed
		}
	}

	var contentText *string
	if content != nil {
		raw, err := json.Marshal(content)
		if err == nil {
			text := string(raw)
			contentText = &text
		}
	}

	clientIP := c.ClientIP()
	record := model.OperationLog{
		AdminID:   adminID,
		Module:    module,
		Action:    action,
		TargetID:  targetID,
		Content:   contentText,
		ClientIP:  &clientIP,
		CreatedAt: time.Now(),
	}
	_ = db.Create(&record).Error
}
