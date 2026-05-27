package router

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	adminpkg "xnauth/internal/admin"
	"xnauth/internal/adminapi"
	"xnauth/internal/auth"
	"xnauth/internal/clientapi"
	"xnauth/internal/collect"
	"xnauth/internal/config"
	"xnauth/internal/middleware"
	"xnauth/pkg/response"
)

type Dependencies struct {
	Config *config.Config
	Logger *zap.Logger
	DB     *gorm.DB
}

func New(deps Dependencies) *gin.Engine {
	if deps.Config != nil && deps.Config.Server.Mode != "" {
		gin.SetMode(deps.Config.Server.Mode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	if deps.Logger != nil {
		engine.Use(requestLogger(deps.Logger))
	}

	engine.GET("/health", healthHandler(deps))

	api := engine.Group("/api")
	{
		api.GET("/health", healthHandler(deps))
		systemHandler := adminapi.NewSystemHandler(deps.DB)

		// 公开接口不能依赖后台身份令牌中间件。这里保持最小化，因为新增的任何路由
		// 都会暴露给未登录访客和已部署客户端。
		public := api.Group("/public")
		{
			public.GET("/site-settings", systemHandler.PublicSiteSettings)
		}

		// 客户端业务接口统一走 /secure 加密信封入口，避免再暴露明文兼容链路。
		client := api.Group("/client")
		{
			client.GET("/health", healthHandler(deps))
			secureHandler := clientapi.NewSecureHandler(deps.DB, deps.Config.Secure, deps.Config.Auth)
			client.GET("/secure/config", secureHandler.Config)
			client.POST("/secure", secureHandler.Handle)
		}

		adminGroup := api.Group("/admin")
		{
			adminGroup.GET("/health", healthHandler(deps))
			jwtManager := auth.NewManager(deps.Config.JWT)
			adminHandler := adminpkg.NewHandler(deps.DB, jwtManager)
			adminGroup.GET("/captcha", adminHandler.Captcha)
			adminGroup.POST("/captcha/verify", adminHandler.VerifyCaptcha)
			adminGroup.POST("/login", adminHandler.Login)

			protected := adminGroup.Group("")
			protected.Use(middleware.AdminAuth(jwtManager))
			{
				// 后台受保护路由统一在这里挂载，便于确认哪些接口只能登录后访问。
				protected.GET("/profile", adminHandler.Profile)
				protected.PUT("/profile", adminHandler.UpdateProfile)
				protected.GET("/settings/security", systemHandler.SecuritySettings)
				protected.PUT("/settings/security", systemHandler.UpdateSecuritySettings)
				protected.GET("/settings/site", systemHandler.SiteSettings)
				protected.PUT("/settings/site", systemHandler.UpdateSiteSettings)
				protected.GET("/settings/status", systemHandler.Status)
				protected.POST("/settings/cleanup", systemHandler.CleanupData)

				appHandler := adminapi.NewAppHandler(deps.DB)
				protected.GET("/apps", appHandler.List)
				protected.POST("/apps", appHandler.Create)
				protected.GET("/apps/:id", appHandler.Detail)
				protected.PUT("/apps/:id", appHandler.Update)
				protected.PUT("/apps/:id/security-keys/generate", appHandler.GenerateSecurityKeys)
				protected.DELETE("/apps/:id", appHandler.Delete)

				sessionTimeoutSeconds := 90
				if deps.Config != nil && deps.Config.Auth.SessionTimeoutSeconds > 0 {
					sessionTimeoutSeconds = deps.Config.Auth.SessionTimeoutSeconds
				}

				dashboardHandler := adminapi.NewDashboardHandler(deps.DB, sessionTimeoutSeconds)
				protected.GET("/dashboard/summary", dashboardHandler.Summary)

				licenseHandler := adminapi.NewLicenseHandler(deps.DB, sessionTimeoutSeconds)
				protected.GET("/licenses", licenseHandler.List)
				protected.POST("/licenses", licenseHandler.Create)
				protected.DELETE("/licenses/expired", licenseHandler.DeleteExpired)
				protected.GET("/licenses/:id", licenseHandler.Detail)
				protected.PUT("/licenses/:id", licenseHandler.Update)
				protected.PUT("/licenses/:id/status", licenseHandler.UpdateStatus)
				protected.DELETE("/licenses/:id", licenseHandler.Delete)

				deviceHandler := adminapi.NewDeviceHandler(deps.DB)
				protected.GET("/devices", deviceHandler.List)
				protected.GET("/devices/:id", deviceHandler.Detail)
				protected.PUT("/devices/:id/status", deviceHandler.UpdateStatus)
				protected.PUT("/devices/:id/unbind", deviceHandler.Unbind)

				sessionHandler := adminapi.NewSessionHandler(deps.DB, sessionTimeoutSeconds)
				protected.GET("/sessions", sessionHandler.List)
				protected.PUT("/sessions/:id/revoke", sessionHandler.Revoke)

				announcementHandler := adminapi.NewAnnouncementHandler(deps.DB)
				protected.GET("/announcements", announcementHandler.List)
				protected.POST("/announcements", announcementHandler.Create)
				protected.GET("/announcements/:id", announcementHandler.Detail)
				protected.PUT("/announcements/:id", announcementHandler.Update)
				protected.DELETE("/announcements/:id", announcementHandler.Delete)

				versionHandler := adminapi.NewVersionHandler(deps.DB)
				protected.GET("/versions", versionHandler.List)
				protected.POST("/versions", versionHandler.Create)
				protected.GET("/versions/:id", versionHandler.Detail)
				protected.PUT("/versions/:id", versionHandler.Update)
				protected.DELETE("/versions/:id", versionHandler.Delete)

				collectHandler := collect.NewHandler(deps.DB, sessionTimeoutSeconds)
				protected.GET("/collect/fields", collectHandler.ListFields)
				protected.POST("/collect/fields", collectHandler.CreateField)
				protected.PUT("/collect/fields/:id", collectHandler.UpdateField)
				protected.DELETE("/collect/fields/:id", collectHandler.DeleteField)
				protected.GET("/collect/stats", collectHandler.Stats)
				protected.GET("/collect/summary", collectHandler.Stats)
				protected.GET("/collect/records", collectHandler.ListRecords)
				protected.GET("/collect/records/:id", collectHandler.RecordDetail)
				protected.GET("/licenses/:id/collect/latest", collectHandler.LatestByLicense)
				protected.GET("/licenses/:id/collect/records", collectHandler.RecordsByLicense)

				logHandler := adminapi.NewAuditLogHandler(deps.DB)
				protected.GET("/verify-logs", logHandler.VerifyLogs)
				protected.GET("/operation-logs", logHandler.OperationLogs)
			}
		}
	}

	return engine
}

func requestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		logger.Info("http request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.String("client_ip", c.ClientIP()),
			zap.Duration("latency", time.Since(startedAt)),
		)
	}
}

func healthHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		data := gin.H{
			"service":   "XNAuth",
			"status":    "ok",
			"timestamp": time.Now().Unix(),
			"checks": gin.H{
				"mysql":  checkMySQL(c.Request.Context(), deps.DB),
				"secure": deps.Config != nil && deps.Config.Secure.Enabled,
			},
		}
		response.JSON(c, http.StatusOK, response.CodeOK, "ok", data)
	}
}

func checkMySQL(ctx context.Context, db *gorm.DB) string {
	if db == nil {
		return "not_configured"
	}
	sqlDB, err := db.DB()
	if err != nil {
		return "error"
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return "error"
	}
	return "ok"
}
