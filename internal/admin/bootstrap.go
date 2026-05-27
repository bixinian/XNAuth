package admin

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"xnauth/internal/model"
	"xnauth/pkg/utils"
)

func EnsureAdminUser(db *gorm.DB, username string, password string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("admin username is required")
	}
	if len(password) < 6 {
		return fmt.Errorf("admin password must be at least 6 characters")
	}

	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	now := time.Now()
	var user model.AdminUser
	err = db.Where("username = ?", username).First(&user).Error
	if err == nil {
		return db.Model(&user).Updates(map[string]any{
			"password_hash": passwordHash,
			"status":        model.AdminStatusEnabled,
			"updated_at":    now,
		}).Error
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	user = model.AdminUser{
		Username:     username,
		PasswordHash: passwordHash,
		Status:       model.AdminStatusEnabled,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	return db.Create(&user).Error
}
