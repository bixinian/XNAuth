package systemconfig

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"xnauth/internal/model"
)

const (
	LoginCaptchaEnabledKey = "login_captcha_enabled"
	SiteNameKey            = "site_name"
	ICPNumberKey           = "icp_number"
	FooterLinksKey         = "footer_links"
)

func String(db *gorm.DB, key string, fallback string) string {
	value, err := Value(db, key)
	if err != nil {
		return fallback
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func Bool(db *gorm.DB, key string, fallback bool) bool {
	value, err := Value(db, key)
	if err != nil {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off", "":
		return false
	default:
		return fallback
	}
}

func Value(db *gorm.DB, key string) (string, error) {
	var setting model.SystemSetting
	if err := db.Where("setting_key = ?", key).First(&setting).Error; err != nil {
		return "", err
	}
	return setting.SettingValue, nil
}

func SetBool(db *gorm.DB, key string, enabled bool) error {
	if enabled {
		return SetValue(db, key, "1")
	}
	return SetValue(db, key, "0")
}

func SetValue(db *gorm.DB, key string, value string) error {
	now := time.Now()
	var setting model.SystemSetting
	err := db.Where("setting_key = ?", key).First(&setting).Error
	if err == nil {
		return db.Model(&setting).Updates(map[string]any{
			"setting_value": value,
			"updated_at":    now,
		}).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return db.Create(&model.SystemSetting{
		SettingKey:   key,
		SettingValue: value,
		CreatedAt:    now,
		UpdatedAt:    now,
	}).Error
}
