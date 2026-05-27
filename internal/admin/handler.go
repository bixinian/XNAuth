package admin

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"xnauth/internal/auth"
	"xnauth/internal/middleware"
	"xnauth/internal/model"
	"xnauth/internal/operationlog"
	"xnauth/internal/systemconfig"
	"xnauth/pkg/response"
	"xnauth/pkg/utils"
)

type Handler struct {
	db      *gorm.DB
	jwt     *auth.Manager
	captcha *CaptchaStore
}

func NewHandler(db *gorm.DB, jwtManager *auth.Manager) *Handler {
	return &Handler{db: db, jwt: jwtManager, captcha: NewCaptchaStore()}
}

type loginReq struct {
	Username      string `json:"username" binding:"required"`
	Password      string `json:"password" binding:"required"`
	CaptchaID     string `json:"captcha_id"`
	CaptchaAnswer string `json:"captcha_answer"`
	CaptchaToken  string `json:"captcha_token"`
}

type verifyCaptchaReq struct {
	CaptchaID   string `json:"captcha_id" binding:"required"`
	SliderValue int    `json:"slider_value" binding:"required"`
}

type updateProfileReq struct {
	Username        *string `json:"username"`
	CurrentPassword string  `json:"current_password"`
	NewPassword     string  `json:"new_password"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}

	if h.loginCaptchaEnabled() {
		token := strings.TrimSpace(req.CaptchaToken)
		if token == "" {
			token = req.CaptchaAnswer
		}
		if !h.captcha.VerifyToken(req.CaptchaID, token) {
			response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid_captcha")
			return
		}
	}

	var user model.AdminUser
	err := h.db.Where("username = ?", strings.TrimSpace(req.Username)).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "invalid username or password")
		return
	}
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}
	if user.Status != model.AdminStatusEnabled || !utils.CheckPassword(req.Password, user.PasswordHash) {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "invalid username or password")
		return
	}

	now := time.Now()
	if err := h.db.Model(&user).Update("last_login_at", now).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	token, expireAt, err := h.jwt.Generate(user.ID, user.Username)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}

	response.OK(c, gin.H{
		"token":     token,
		"expire_at": expireAt,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (h *Handler) Captcha(c *gin.Context) {
	if !h.loginCaptchaEnabled() {
		response.OK(c, gin.H{"enabled": false})
		return
	}
	id := h.captcha.Generate()
	response.OK(c, gin.H{
		"enabled":        true,
		"type":           "slider",
		"captcha_id":     id,
		"expire_seconds": 180,
	})
}

func (h *Handler) VerifyCaptcha(c *gin.Context) {
	if !h.loginCaptchaEnabled() {
		response.OK(c, gin.H{"enabled": false, "verified": true})
		return
	}
	var req verifyCaptchaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}
	token, ok := h.captcha.VerifySlider(req.CaptchaID, req.SliderValue)
	if !ok {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid_captcha")
		return
	}
	response.OK(c, gin.H{
		"enabled":       true,
		"verified":      true,
		"captcha_id":    req.CaptchaID,
		"captcha_token": token,
	})
}

func (h *Handler) Profile(c *gin.Context) {
	user, ok := h.currentAdminUser(c)
	if !ok {
		return
	}
	response.OK(c, user)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	var req updateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid params")
		return
	}

	user, ok := h.currentAdminUser(c)
	if !ok {
		return
	}

	updates := map[string]any{}
	if req.Username != nil {
		username := strings.TrimSpace(*req.Username)
		if username == "" {
			response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid username")
			return
		}
		if username != user.Username {
			var exists model.AdminUser
			err := h.db.Where("username = ? AND id <> ?", username, user.ID).First(&exists).Error
			if err == nil {
				response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "username_exists")
				return
			}
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
				return
			}
			updates["username"] = username
		}
	}
	newPassword := strings.TrimSpace(req.NewPassword)
	if newPassword != "" {
		if len(newPassword) < 6 {
			response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "password_too_short")
			return
		}
		passwordHash, err := utils.HashPassword(newPassword)
		if err != nil {
			response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
			return
		}
		updates["password_hash"] = passwordHash
	}

	if len(updates) == 0 {
		response.OK(c, user)
		return
	}
	if !utils.CheckPassword(req.CurrentPassword, user.PasswordHash) {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid_current_password")
		return
	}

	updates["updated_at"] = time.Now()
	if err := h.db.Model(&user).Updates(updates).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}
	operationlog.Write(h.db, c, "admin_user", "update_profile", &user.ID, operationLogUserUpdates(updates))
	if err := h.db.First(&user, user.ID).Error; err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return
	}
	response.OK(c, user)
}

func (h *Handler) currentAdminUser(c *gin.Context) (model.AdminUser, bool) {
	value, exists := c.Get(middleware.ContextAdminID)
	adminID, ok := auth.AdminIDFromAny(value)
	if !exists || !ok {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "invalid token")
		return model.AdminUser{}, false
	}
	var user model.AdminUser
	if err := h.db.First(&user, adminID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "invalid token")
			return model.AdminUser{}, false
		}
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "server error")
		return model.AdminUser{}, false
	}
	return user, true
}

func (h *Handler) loginCaptchaEnabled() bool {
	return systemconfig.Bool(h.db, systemconfig.LoginCaptchaEnabledKey, false)
}

func operationLogUserUpdates(updates map[string]any) gin.H {
	content := gin.H{}
	for key, value := range updates {
		if key == "password_hash" {
			content["password_changed"] = true
			continue
		}
		if key == "updated_at" {
			continue
		}
		content[key] = value
	}
	return content
}
