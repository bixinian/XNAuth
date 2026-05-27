package clientapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"xnauth/internal/config"
	"xnauth/internal/model"
	"xnauth/internal/securetransport"
	"xnauth/internal/signature"
	"xnauth/pkg/response"
	"xnauth/pkg/utils"
)

const (
	maxReportBodyBytes = 64 * 1024
	maxReportFields    = 50
	maxFieldValueLen   = 2048
)

type SecureHandler struct {
	db        *gorm.DB
	secureCfg config.SecureTransportConfig
	auth      config.AuthConfig
}

func NewSecureHandler(db *gorm.DB, secureConfig config.SecureTransportConfig, authConfig config.AuthConfig) *SecureHandler {
	return &SecureHandler{db: db, secureCfg: secureConfig, auth: authConfig}
}

type secureVerifyReq struct {
	AppKey            string `json:"app_key"`
	LicenseKey        string `json:"license_key"`
	MachineCode       string `json:"machine_code"`
	DeviceName        string `json:"device_name"`
	DevicePublicKey   string `json:"device_public_key"`
	ClientVersion     string `json:"client_version"`
	ClientVersionCode int    `json:"client_version_code"`
	Nonce             string `json:"nonce"`
}

type secureHeartbeatReq struct {
	AppKey            string `json:"app_key"`
	LicenseKey        string `json:"license_key"`
	SessionToken      string `json:"session_token"`
	MachineCode       string `json:"machine_code"`
	ClientVersion     string `json:"client_version"`
	ClientVersionCode int    `json:"client_version_code"`
	Nonce             string `json:"nonce"`
}

type secureAnnouncementReq struct {
	AppKey      string `json:"app_key"`
	LicenseKey  string `json:"license_key"`
	MachineCode string `json:"machine_code"`
	Nonce       string `json:"nonce"`
}

type secureUpdateReq struct {
	AppKey            string `json:"app_key"`
	LicenseKey        string `json:"license_key"`
	MachineCode       string `json:"machine_code"`
	ClientVersion     string `json:"client_version"`
	ClientVersionCode int    `json:"client_version_code"`
	Nonce             string `json:"nonce"`
}

type secureReportReq struct {
	AppKey      string         `json:"app_key"`
	LicenseKey  string         `json:"license_key"`
	MachineCode string         `json:"machine_code"`
	Event       string         `json:"event"`
	Data        map[string]any `json:"data"`
	Nonce       string         `json:"nonce"`
}

type secureResult struct {
	Code    int
	Message string
	Data    gin.H
}

type secureClientContext struct {
	app             model.App
	license         model.LicenseCard
	device          model.LicenseDevice
	session         model.LicenseSession
	machineCodeHash string
}

// Config 只返回目标应用的公钥材料。开发阶段可以读取该接口辅助调试；
// 生产客户端应内置预期的密钥 ID 和公钥，避免盲目信任被篡改的配置接口。
func (h *SecureHandler) Config(c *gin.Context) {
	if !h.secureCfg.Enabled {
		response.OK(c, gin.H{"enabled": false})
		return
	}
	appKey := strings.TrimSpace(c.Query("app_key"))
	if appKey == "" {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "app_key_required")
		return
	}
	app, err := h.findEnabledApp(h.db, appKey)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
		return
	}
	keyID, x25519Public, ed25519Public, err := appPublicSecurityConfig(app)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
		return
	}
	response.OK(c, gin.H{
		"enabled":                   true,
		"kid":                       keyID,
		"algorithm":                 "X25519-HKDF-SHA256/AES-256-GCM + Ed25519",
		"app_key":                   app.AppKey,
		"server_x25519_public_key":  x25519Public,
		"server_ed25519_public_key": ed25519Public,
		"timestamp_skew_seconds":    h.timestampSkewSeconds(),
	})
}

// Handle 是客户端加密信封的统一入口。服务端先解开信封，再根据明文动作字段 action
// 分发到授权、心跳、公告、更新和数据上报等业务规则。
func (h *SecureHandler) Handle(c *gin.Context) {
	if !h.secureCfg.Enabled {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "secure_transport_not_configured")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxReportBodyBytes)

	var env securetransport.RequestEnvelope
	if err := c.ShouldBindJSON(&env); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid_secure_envelope")
		return
	}
	manager, err := h.secureManagerForAppKey(env.AppKey)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
		return
	}
	secureCtx, plaintext, err := manager.OpenRequest(env)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "invalid_secure_request")
		return
	}

	result := h.dispatch(c, secureCtx, plaintext)
	h.writeEncrypted(c, manager, secureCtx, result.Code, result.Message, result.Data)
}

func (h *SecureHandler) dispatch(c *gin.Context, secureCtx *securetransport.RequestContext, plaintext []byte) secureResult {
	switch strings.TrimSpace(secureCtx.Envelope.Action) {
	case "auth.verify":
		return h.handleVerify(c, secureCtx, plaintext)
	case "auth.heartbeat":
		return h.handleHeartbeat(secureCtx, plaintext)
	case "announcements":
		return h.handleAnnouncements(secureCtx, plaintext)
	case "update.check":
		return h.handleUpdate(secureCtx, plaintext)
	case "collect.report":
		return h.handleCollectReport(c, secureCtx, plaintext)
	default:
		return fail(response.CodeBadRequest, "unsupported_secure_action")
	}
}

func (h *SecureHandler) writeEncrypted(c *gin.Context, manager *securetransport.Manager, secureCtx *securetransport.RequestContext, code int, message string, data gin.H) {
	if message == "" {
		message = "ok"
	}
	_, env, err := manager.SealResponse(secureCtx, http.StatusOK, code, message, data)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.CodeInternalServerError, "secure_response_failed")
		return
	}
	c.JSON(http.StatusOK, env)
}

func (h *SecureHandler) handleVerify(c *gin.Context, secureCtx *securetransport.RequestContext, plaintext []byte) secureResult {
	var req secureVerifyReq
	if err := json.Unmarshal(plaintext, &req); err != nil {
		return fail(response.CodeBadRequest, "invalid_params")
	}
	req.normalize()
	if err := req.ensureRequired(secureCtx.Envelope.Nonce, secureCtx.Envelope.AppKey); err != nil {
		return fail(response.CodeBadRequest, err.Error())
	}
	// 首次验证需要从明文中接收设备公钥，以便新设备完成绑定。
	// 事务内校验会在加载或创建设备记录后，再次使用已存储公钥验签。
	if err := secureCtx.VerifyClientSignature(req.DevicePublicKey); err != nil {
		return fail(response.CodeForbidden, err.Error())
	}

	var result secureClientContext
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var txErr error
		result, txErr = h.verifyInTransaction(tx, c, secureCtx, req)
		return txErr
	})
	if err != nil {
		return fail(response.CodeForbidden, err.Error())
	}

	h.writeVerifyLog(c, result.app.ID, &result.license.ID, &result.device.ID, req.LicenseKey, result.machineCodeHash, req.ClientVersion, req.ClientVersionCode, true, "")
	return ok(h.verifyData(req, result))
}

func (h *SecureHandler) secureManagerForAppKey(appKey string) (*securetransport.Manager, error) {
	appKey = strings.TrimSpace(appKey)
	if appKey == "" {
		return nil, errors.New("app_key_required")
	}
	app, err := h.findEnabledApp(h.db, appKey)
	if err != nil {
		return nil, err
	}
	keyID, x25519Private, ed25519Private, err := appPrivateSecurityConfig(app)
	if err != nil {
		return nil, err
	}
	// 按需使用应用私钥创建传输管理器，确保每个应用的密钥域彼此隔离，
	// 单个应用轮换密钥不会影响其他应用。
	return securetransport.NewManagerFromKeys(keyID, x25519Private, ed25519Private, h.timestampSkewSeconds())
}

func (h *SecureHandler) timestampSkewSeconds() int {
	if h.secureCfg.TimestampSkewSeconds > 0 {
		return h.secureCfg.TimestampSkewSeconds
	}
	return 120
}

func appPublicSecurityConfig(app model.App) (string, string, string, error) {
	keyID := secureDerefString(app.SecureKeyID)
	x25519Public := secureDerefString(app.SecureX25519Public)
	ed25519Public := secureDerefString(app.SecureEd25519Public)
	if keyID == "" || x25519Public == "" || ed25519Public == "" {
		return "", "", "", errors.New("app_secure_keys_not_configured")
	}
	return keyID, x25519Public, ed25519Public, nil
}

func appPrivateSecurityConfig(app model.App) (string, string, string, error) {
	keyID := secureDerefString(app.SecureKeyID)
	x25519Private := secureDerefString(app.SecureX25519Private)
	ed25519Private := secureDerefString(app.SecureEd25519Private)
	if keyID == "" || x25519Private == "" || ed25519Private == "" {
		return "", "", "", errors.New("app_secure_keys_not_configured")
	}
	return keyID, x25519Private, ed25519Private, nil
}

func (h *SecureHandler) handleHeartbeat(secureCtx *securetransport.RequestContext, plaintext []byte) secureResult {
	var req secureHeartbeatReq
	if err := json.Unmarshal(plaintext, &req); err != nil {
		return fail(response.CodeBadRequest, "invalid_params")
	}
	req.normalize()
	if err := req.ensureRequired(secureCtx.Envelope.Nonce, secureCtx.Envelope.AppKey); err != nil {
		return fail(response.CodeBadRequest, err.Error())
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		client, err := h.authorizeDevice(tx, req.AppKey, req.LicenseKey, req.MachineCode, now)
		if err != nil {
			return err
		}
		if err := secureCtx.VerifyClientSignature(secureDerefString(client.device.DevicePublicKey)); err != nil {
			return err
		}
		if err := h.rememberNonce(tx, client.app.ID, client.device.ID, req.Nonce, now); err != nil {
			return err
		}
		return h.refreshHeartbeat(tx, client, req, now)
	})
	if err != nil {
		return fail(response.CodeForbidden, err.Error())
	}
	return ok(gin.H{"server_time": time.Now().Unix(), "client_nonce": req.Nonce})
}

func (h *SecureHandler) handleAnnouncements(secureCtx *securetransport.RequestContext, plaintext []byte) secureResult {
	var req secureAnnouncementReq
	if err := json.Unmarshal(plaintext, &req); err != nil {
		return fail(response.CodeBadRequest, "invalid_params")
	}
	req.normalize()
	if err := req.ensureRequired(secureCtx.Envelope.Nonce, secureCtx.Envelope.AppKey); err != nil {
		return fail(response.CodeBadRequest, err.Error())
	}

	var records []model.AppAnnouncement
	var client secureClientContext
	err := h.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		var err error
		client, err = h.authorizeDevice(tx, req.AppKey, req.LicenseKey, req.MachineCode, now)
		if err != nil {
			return err
		}
		if err := secureCtx.VerifyClientSignature(secureDerefString(client.device.DevicePublicKey)); err != nil {
			return err
		}
		if err := h.rememberNonce(tx, client.app.ID, client.device.ID, req.Nonce, now); err != nil {
			return err
		}
		return tx.Where("app_id = ? AND enabled = 1 AND (start_at IS NULL OR start_at <= ?) AND (end_at IS NULL OR end_at >= ?)", client.app.ID, now, now).
			Order("sort_order DESC, id DESC").
			Find(&records).Error
	})
	if err != nil {
		return fail(response.CodeForbidden, normalizeDBError(err))
	}

	items := make([]gin.H, 0, len(records))
	for _, record := range records {
		items = append(items, gin.H{
			"title":       record.Title,
			"content":     record.Content,
			"notice_type": secureDerefString(record.NoticeType),
			"popup":       record.Popup == 1,
		})
	}
	digest, err := signature.PayloadDigest(items)
	if err != nil {
		return fail(response.CodeInternalServerError, "server_error")
	}
	return ok(gin.H{
		"announcements":  items,
		"payload_digest": digest,
		"server_time":    time.Now().Unix(),
		"client_nonce":   req.Nonce,
	})
}

func (h *SecureHandler) handleUpdate(secureCtx *securetransport.RequestContext, plaintext []byte) secureResult {
	var req secureUpdateReq
	if err := json.Unmarshal(plaintext, &req); err != nil {
		return fail(response.CodeBadRequest, "invalid_params")
	}
	req.normalize()
	if err := req.ensureRequired(secureCtx.Envelope.Nonce, secureCtx.Envelope.AppKey); err != nil {
		return fail(response.CodeBadRequest, err.Error())
	}

	var latest model.AppVersion
	var client secureClientContext
	err := h.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		var err error
		client, err = h.authorizeDevice(tx, req.AppKey, req.LicenseKey, req.MachineCode, now)
		if err != nil {
			return err
		}
		if err := secureCtx.VerifyClientSignature(secureDerefString(client.device.DevicePublicKey)); err != nil {
			return err
		}
		if err := h.rememberNonce(tx, client.app.ID, client.device.ID, req.Nonce, now); err != nil {
			return err
		}
		return tx.Where("app_id = ? AND enabled = 1", client.app.ID).Order("version_code DESC").First(&latest).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		update := gin.H{
			"has_update":          false,
			"latest_version":      req.ClientVersion,
			"latest_version_code": req.ClientVersionCode,
			"force_update":        false,
			"download_url":        "",
			"file_hash":           "",
			"file_size":           0,
			"changelog":           "",
		}
		return okPayload(req.Nonce, "update", update)
	}
	if err != nil {
		return fail(response.CodeForbidden, normalizeDBError(err))
	}

	forceUpdate := client.app.ForceUpdate == 1
	if client.app.MinLoginVersionCode != nil && req.ClientVersionCode < *client.app.MinLoginVersionCode {
		forceUpdate = true
	}
	update := gin.H{
		"has_update":          latest.VersionCode > req.ClientVersionCode,
		"latest_version":      latest.VersionName,
		"latest_version_code": latest.VersionCode,
		"force_update":        forceUpdate,
		"download_url":        secureDerefString(latest.DownloadURL),
		"file_hash":           secureDerefString(latest.FileHash),
		"file_size":           secureDerefInt64(latest.FileSize),
		"changelog":           secureDerefString(latest.Changelog),
	}
	return okPayload(req.Nonce, "update", update)
}

func (h *SecureHandler) handleCollectReport(c *gin.Context, secureCtx *securetransport.RequestContext, plaintext []byte) secureResult {
	var req secureReportReq
	if err := json.Unmarshal(plaintext, &req); err != nil {
		return fail(response.CodeBadRequest, "invalid_params")
	}
	req.normalize()
	if err := req.ensureRequired(secureCtx.Envelope.Nonce, secureCtx.Envelope.AppKey); err != nil {
		return fail(response.CodeBadRequest, err.Error())
	}
	if len(req.Data) > maxReportFields {
		return fail(response.CodeBadRequest, "too_many_fields")
	}

	var record model.CollectRecord
	var savedFields int
	err := h.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		client, err := h.authorizeDevice(tx, req.AppKey, req.LicenseKey, req.MachineCode, now)
		if err != nil {
			return err
		}
		if err := secureCtx.VerifyClientSignature(secureDerefString(client.device.DevicePublicKey)); err != nil {
			return err
		}
		if err := h.rememberNonce(tx, client.app.ID, client.device.ID, req.Nonce, now); err != nil {
			return err
		}

		var fields []model.CollectField
		if err := tx.Where("app_id = ? AND enabled = 1", client.app.ID).Find(&fields).Error; err != nil {
			return errors.New("server_error")
		}
		allowed := make(map[string]model.CollectField, len(fields))
		for _, field := range fields {
			allowed[field.FieldKey] = field
		}

		values := make([]model.CollectRecordValue, 0, len(req.Data))
		for key, rawValue := range req.Data {
			if _, ok := allowed[key]; !ok {
				continue
			}
			value := stringify(rawValue)
			if len(value) > maxFieldValueLen {
				value = value[:maxFieldValueLen]
			}
			values = append(values, model.CollectRecordValue{
				AppID:      client.app.ID,
				LicenseID:  &client.license.ID,
				DeviceID:   &client.device.ID,
				FieldKey:   key,
				FieldValue: value,
				CreatedAt:  now,
			})
		}

		event := strings.TrimSpace(req.Event)
		if event == "" {
			event = "custom"
		}
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()
		record = model.CollectRecord{
			AppID:           client.app.ID,
			LicenseID:       &client.license.ID,
			DeviceID:        &client.device.ID,
			LicenseKey:      req.LicenseKey,
			MachineCodeHash: &client.machineCodeHash,
			Event:           &event,
			ClientIP:        &clientIP,
			UserAgent:       &userAgent,
			CreatedAt:       now,
		}
		if err := tx.Create(&record).Error; err != nil {
			return errors.New("server_error")
		}
		for i := range values {
			values[i].RecordID = record.ID
		}
		if len(values) == 0 {
			return nil
		}
		if err := tx.Create(&values).Error; err != nil {
			return errors.New("server_error")
		}
		savedFields = len(values)
		return nil
	})
	if err != nil {
		return fail(response.CodeForbidden, err.Error())
	}
	return ok(gin.H{"record_id": record.ID, "saved_fields": savedFields})
}

func (h *SecureHandler) verifyInTransaction(tx *gorm.DB, c *gin.Context, secureCtx *securetransport.RequestContext, req secureVerifyReq) (secureClientContext, error) {
	now := time.Now()
	result := secureClientContext{machineCodeHash: utils.MachineCodeHash(req.MachineCode)}

	// 加密验证顺序必须稳定：应用 -> 卡密 -> 版本 -> 设备 -> 签名 -> 随机数 -> 会话。
	// 失败请求不应创建会话；随机数也必须在目标设备明确后才有保存意义。
	app, err := h.findEnabledApp(tx, req.AppKey)
	if err != nil {
		return result, err
	}
	result.app = app

	license, err := h.findUsableLicense(tx, app.ID, req.LicenseKey, now)
	if err != nil {
		h.writeVerifyLog(c, app.ID, nil, nil, req.LicenseKey, result.machineCodeHash, req.ClientVersion, req.ClientVersionCode, false, err.Error())
		return result, err
	}
	result.license = license

	if err := h.checkLoginVersion(app, req.ClientVersionCode); err != nil {
		h.writeVerifyLog(c, app.ID, &license.ID, nil, req.LicenseKey, result.machineCodeHash, req.ClientVersion, req.ClientVersionCode, false, err.Error())
		return result, err
	}

	device, err := h.bindOrRefreshDevice(tx, license, result.machineCodeHash, req, now)
	if err != nil {
		h.writeVerifyLog(c, app.ID, &license.ID, nil, req.LicenseKey, result.machineCodeHash, req.ClientVersion, req.ClientVersionCode, false, err.Error())
		return result, err
	}
	result.device = device

	if err := secureCtx.VerifyClientSignature(secureDerefString(device.DevicePublicKey)); err != nil {
		return result, err
	}
	if err := h.rememberNonce(tx, app.ID, device.ID, req.Nonce, now); err != nil {
		return result, err
	}

	session, err := h.createOrRefreshSession(tx, c, license, device, req, now)
	if err != nil {
		h.writeVerifyLog(c, app.ID, &license.ID, &device.ID, req.LicenseKey, result.machineCodeHash, req.ClientVersion, req.ClientVersionCode, false, err.Error())
		return result, err
	}
	result.session = session

	if license.Status == model.LicenseStatusInactive {
		updates := map[string]any{
			"status":       model.LicenseStatusActive,
			"activated_at": now,
			"updated_at":   now,
		}
		if err := tx.Model(&model.LicenseCard{}).Where("id = ?", license.ID).Updates(updates).Error; err != nil {
			return result, errors.New("server_error")
		}
		result.license.Status = model.LicenseStatusActive
		result.license.ActivatedAt = &now
	}
	return result, nil
}

func (h *SecureHandler) authorizeDevice(tx *gorm.DB, appKey string, licenseKey string, machineCode string, now time.Time) (secureClientContext, error) {
	client := secureClientContext{machineCodeHash: utils.MachineCodeHash(machineCode)}
	app, err := h.findEnabledApp(tx, appKey)
	if err != nil {
		return client, err
	}
	client.app = app

	license, err := h.findUsableLicense(tx, app.ID, licenseKey, now)
	if err != nil {
		return client, err
	}
	client.license = license

	var device model.LicenseDevice
	err = tx.Where("app_id = ? AND license_id = ? AND machine_code_hash = ?", app.ID, license.ID, client.machineCodeHash).First(&device).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return client, errors.New("device_not_found")
	}
	if err != nil {
		return client, errors.New("server_error")
	}
	if device.Status == model.DeviceStatusDisabled {
		return client, errors.New("device_disabled")
	}
	if device.Status != model.DeviceStatusNormal {
		return client, errors.New("device_invalid")
	}
	if secureDerefString(device.DevicePublicKey) == "" {
		return client, errors.New("device_key_unbound")
	}
	// 后续加密动作只能使用已绑定的设备公钥，不能从请求载荷中接受替换公钥。
	client.device = device
	return client, nil
}

func (h *SecureHandler) refreshHeartbeat(tx *gorm.DB, client secureClientContext, req secureHeartbeatReq, now time.Time) error {
	var session model.LicenseSession
	err := tx.Where("app_id = ? AND session_token = ?", client.app.ID, req.SessionToken).First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("session_not_found")
	}
	if err != nil {
		return errors.New("server_error")
	}
	if session.Status == model.SessionStatusRevoked {
		return errors.New("session_revoked")
	}
	if session.Status != model.SessionStatusOnline {
		return errors.New("session_invalid")
	}
	if session.LastHeartbeatAt.Before(now.Add(-time.Duration(h.auth.SessionTimeoutSeconds) * time.Second)) {
		tx.Model(&session).Updates(map[string]any{"status": model.SessionStatusTimeout, "updated_at": now})
		return errors.New("session_timeout")
	}
	if session.LicenseID != client.license.ID {
		return errors.New("session_license_mismatch")
	}
	if session.DeviceID != client.device.ID {
		return errors.New("session_device_mismatch")
	}
	if err := h.checkLoginVersion(client.app, req.ClientVersionCode); err != nil {
		return err
	}

	clientVersion := req.ClientVersion
	if err := tx.Model(&session).Updates(map[string]any{
		"last_heartbeat_at":   now,
		"client_version":      &clientVersion,
		"client_version_code": &req.ClientVersionCode,
		"updated_at":          now,
	}).Error; err != nil {
		return errors.New("server_error")
	}
	return tx.Model(&model.LicenseDevice{}).Where("id = ?", client.device.ID).Updates(map[string]any{
		"last_seen_at":        now,
		"client_version":      &clientVersion,
		"client_version_code": &req.ClientVersionCode,
		"updated_at":          now,
	}).Error
}

func (h *SecureHandler) bindOrRefreshDevice(tx *gorm.DB, license model.LicenseCard, machineCodeHash string, req secureVerifyReq, now time.Time) (model.LicenseDevice, error) {
	var device model.LicenseDevice
	err := tx.Where("license_id = ? AND machine_code_hash = ?", license.ID, machineCodeHash).First(&device).Error
	if err == nil {
		if device.Status == model.DeviceStatusDisabled {
			return device, errors.New("device_disabled")
		}
		if device.Status != model.DeviceStatusNormal {
			var count int64
			if err := tx.Model(&model.LicenseDevice{}).Where("license_id = ? AND status = ?", license.ID, model.DeviceStatusNormal).Count(&count).Error; err != nil {
				return device, errors.New("server_error")
			}
			if count >= int64(license.MaxDevices) {
				return device, errors.New("max_devices_exceeded")
			}
			device.Status = model.DeviceStatusNormal
		}
		storedPublicKey := secureDerefString(device.DevicePublicKey)
		if storedPublicKey != "" && storedPublicKey != req.DevicePublicKey {
			return device, errors.New("device_key_mismatch")
		}

		deviceName := secureCleanStringPtr(req.DeviceName)
		clientVersion := req.ClientVersion
		updates := map[string]any{
			"status":              device.Status,
			"device_name":         deviceName,
			"client_version":      &clientVersion,
			"client_version_code": &req.ClientVersionCode,
			"last_seen_at":        now,
			"updated_at":          now,
		}
		if storedPublicKey == "" {
			// 只有管理员解绑清空该设备公钥后，才允许该设备重新绑定。
			// 同一卡密下的其他设备密钥不受影响。
			updates["device_public_key"] = req.DevicePublicKey
			updates["device_key_bound_at"] = now
		}
		if err := tx.Model(&device).Updates(updates).Error; err != nil {
			return device, errors.New("server_error")
		}
		tx.First(&device, device.ID)
		return device, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return device, errors.New("server_error")
	}

	var count int64
	if err := tx.Model(&model.LicenseDevice{}).Where("license_id = ? AND status = ?", license.ID, model.DeviceStatusNormal).Count(&count).Error; err != nil {
		return device, errors.New("server_error")
	}
	if count >= int64(license.MaxDevices) {
		return device, errors.New("max_devices_exceeded")
	}

	deviceName := secureCleanStringPtr(req.DeviceName)
	clientVersion := req.ClientVersion
	devicePublicKey := req.DevicePublicKey
	device = model.LicenseDevice{
		AppID:             license.AppID,
		LicenseID:         license.ID,
		MachineCodeHash:   machineCodeHash,
		DeviceName:        deviceName,
		DevicePublicKey:   &devicePublicKey,
		DeviceKeyBoundAt:  &now,
		ClientVersion:     &clientVersion,
		ClientVersionCode: &req.ClientVersionCode,
		Status:            model.DeviceStatusNormal,
		FirstSeenAt:       now,
		LastSeenAt:        now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := tx.Create(&device).Error; err != nil {
		return device, errors.New("server_error")
	}
	return device, nil
}

func (h *SecureHandler) createOrRefreshSession(tx *gorm.DB, c *gin.Context, license model.LicenseCard, device model.LicenseDevice, req secureVerifyReq, now time.Time) (model.LicenseSession, error) {
	cutoff := now.Add(-time.Duration(h.auth.SessionTimeoutSeconds) * time.Second)
	if err := tx.Model(&model.LicenseSession{}).
		Where("license_id = ? AND status = ? AND last_heartbeat_at < ?", license.ID, model.SessionStatusOnline, cutoff).
		Updates(map[string]any{"status": model.SessionStatusTimeout, "updated_at": now}).Error; err != nil {
		return model.LicenseSession{}, errors.New("server_error")
	}

	var session model.LicenseSession
	err := tx.Where("license_id = ? AND device_id = ? AND status = ? AND last_heartbeat_at >= ?", license.ID, device.ID, model.SessionStatusOnline, cutoff).
		Order("id DESC").
		First(&session).Error
	if err == nil {
		clientVersion := req.ClientVersion
		if err := tx.Model(&session).Updates(map[string]any{
			"client_ip":           c.ClientIP(),
			"client_version":      &clientVersion,
			"client_version_code": &req.ClientVersionCode,
			"last_heartbeat_at":   now,
			"updated_at":          now,
		}).Error; err != nil {
			return session, errors.New("server_error")
		}
		tx.First(&session, session.ID)
		return session, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return session, errors.New("server_error")
	}

	var count int64
	if err := tx.Model(&model.LicenseSession{}).Where("license_id = ? AND status = ? AND last_heartbeat_at >= ?", license.ID, model.SessionStatusOnline, cutoff).Count(&count).Error; err != nil {
		return session, errors.New("server_error")
	}
	if count >= int64(license.MaxOnline) {
		return session, errors.New("max_online_exceeded")
	}

	token, err := utils.RandomToken("session")
	if err != nil {
		return session, errors.New("server_error")
	}
	clientIP := c.ClientIP()
	clientVersion := req.ClientVersion
	session = model.LicenseSession{
		AppID:             license.AppID,
		LicenseID:         license.ID,
		DeviceID:          device.ID,
		SessionToken:      token,
		Status:            model.SessionStatusOnline,
		ClientIP:          &clientIP,
		ClientVersion:     &clientVersion,
		ClientVersionCode: &req.ClientVersionCode,
		StartedAt:         now,
		LastHeartbeatAt:   now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := tx.Create(&session).Error; err != nil {
		return session, errors.New("server_error")
	}
	return session, nil
}

func (h *SecureHandler) findEnabledApp(tx *gorm.DB, appKey string) (model.App, error) {
	var app model.App
	err := tx.Where("app_key = ?", appKey).First(&app).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return app, errors.New("app_not_found")
	}
	if err != nil {
		return app, errors.New("server_error")
	}
	if app.Status != model.AppStatusEnabled {
		return app, errors.New("app_disabled")
	}
	return app, nil
}

func (h *SecureHandler) findUsableLicense(tx *gorm.DB, appID uint64, licenseKey string, now time.Time) (model.LicenseCard, error) {
	var license model.LicenseCard
	err := tx.Where("app_id = ? AND license_key = ?", appID, licenseKey).First(&license).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return license, errors.New("license_not_found")
	}
	if err != nil {
		return license, errors.New("server_error")
	}
	if license.ExpireAt != nil && license.ExpireAt.Before(now) {
		tx.Model(&license).Updates(map[string]any{"status": model.LicenseStatusExpired, "updated_at": now})
		return license, errors.New("license_expired")
	}
	switch license.Status {
	case model.LicenseStatusInactive, model.LicenseStatusActive:
		return license, nil
	case model.LicenseStatusExpired:
		return license, errors.New("license_expired")
	case model.LicenseStatusFrozen:
		return license, errors.New("license_frozen")
	case model.LicenseStatusBanned:
		return license, errors.New("license_banned")
	default:
		return license, errors.New("license_invalid")
	}
}

func (h *SecureHandler) checkLoginVersion(app model.App, clientVersionCode int) error {
	if clientVersionCode <= 0 {
		return errors.New("version_not_supported")
	}
	if app.MinLoginVersionCode != nil && clientVersionCode < *app.MinLoginVersionCode {
		return errors.New("version_not_supported")
	}
	return nil
}

func (h *SecureHandler) rememberNonce(tx *gorm.DB, appID uint64, deviceID uint64, nonce string, now time.Time) error {
	nonce = strings.TrimSpace(nonce)
	if nonce == "" {
		return errors.New("invalid_nonce")
	}
	record := model.ClientNonce{
		AppID:     appID,
		DeviceID:  deviceID,
		Nonce:     nonce,
		CreatedAt: now,
	}
	if err := tx.Create(&record).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return errors.New("replay_request")
		}
		return errors.New("server_error")
	}
	// 随机数 nonce 只需要覆盖重放窗口；顺手清理旧记录，避免表无限增长。
	_ = tx.Where("created_at < ?", now.Add(-24*time.Hour)).Delete(&model.ClientNonce{}).Error
	return nil
}

func (h *SecureHandler) verifyData(req secureVerifyReq, result secureClientContext) gin.H {
	expireAt := secureFormatTime(result.license.ExpireAt)
	return gin.H{
		"license": gin.H{
			"status":            "valid",
			"license_id":        result.license.ID,
			"device_id":         result.device.ID,
			"machine_code_hash": result.machineCodeHash,
			"expire_at":         expireAt,
			"max_devices":       result.license.MaxDevices,
			"max_online":        result.license.MaxOnline,
		},
		"session": gin.H{
			"session_token":      result.session.SessionToken,
			"heartbeat_interval": h.auth.HeartbeatIntervalSeconds,
		},
		"server_time":  time.Now().Unix(),
		"client_nonce": req.Nonce,
	}
}

func (h *SecureHandler) writeVerifyLog(c *gin.Context, appID uint64, licenseID *uint64, deviceID *uint64, licenseKey string, machineCodeHash string, clientVersion string, clientVersionCode int, success bool, failReason string) {
	result := 1
	var reason *string
	if !success {
		result = 2
		reason = &failReason
	}
	clientIP := c.ClientIP()
	record := model.VerifyLog{
		AppID:             appID,
		LicenseID:         licenseID,
		DeviceID:          deviceID,
		LicenseKey:        secureCleanStringPtr(licenseKey),
		MachineCodeHash:   secureCleanStringPtr(machineCodeHash),
		ClientIP:          &clientIP,
		ClientVersion:     secureCleanStringPtr(clientVersion),
		ClientVersionCode: &clientVersionCode,
		Result:            result,
		FailReason:        reason,
		CreatedAt:         time.Now(),
	}
	_ = h.db.Create(&record).Error
}

func ok(data gin.H) secureResult {
	return secureResult{Code: response.CodeOK, Message: "ok", Data: data}
}

func okPayload(nonce string, key string, payload gin.H) secureResult {
	digest, err := signature.PayloadDigest(payload)
	if err != nil {
		sum := sha256.Sum256([]byte("digest_error"))
		digest = hex.EncodeToString(sum[:])
	}
	return ok(gin.H{
		key:              payload,
		"payload_digest": digest,
		"server_time":    time.Now().Unix(),
		"client_nonce":   nonce,
	})
}

func fail(code int, message string) secureResult {
	return secureResult{Code: code, Message: message, Data: gin.H{}}
}

func normalizeDBError(err error) string {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "not_found"
	}
	if err == nil {
		return ""
	}
	if err.Error() == "" {
		return "server_error"
	}
	return err.Error()
}

func (req *secureVerifyReq) normalize() {
	req.AppKey = strings.TrimSpace(req.AppKey)
	req.LicenseKey = strings.TrimSpace(req.LicenseKey)
	req.MachineCode = strings.TrimSpace(req.MachineCode)
	req.DeviceName = strings.TrimSpace(req.DeviceName)
	req.DevicePublicKey = strings.TrimSpace(req.DevicePublicKey)
	req.ClientVersion = strings.TrimSpace(req.ClientVersion)
	req.Nonce = strings.TrimSpace(req.Nonce)
}

func (req *secureVerifyReq) ensureRequired(envelopeNonce string, envelopeAppKey string) error {
	if req.Nonce == "" {
		req.Nonce = envelopeNonce
	}
	if req.Nonce != envelopeNonce {
		return errors.New("nonce_mismatch")
	}
	if req.AppKey != envelopeAppKey {
		return errors.New("app_key_mismatch")
	}
	if req.AppKey == "" || req.LicenseKey == "" || req.MachineCode == "" || req.ClientVersion == "" || req.ClientVersionCode <= 0 || req.DevicePublicKey == "" {
		return errors.New("invalid_params")
	}
	return nil
}

func (req *secureHeartbeatReq) normalize() {
	req.AppKey = strings.TrimSpace(req.AppKey)
	req.LicenseKey = strings.TrimSpace(req.LicenseKey)
	req.SessionToken = strings.TrimSpace(req.SessionToken)
	req.MachineCode = strings.TrimSpace(req.MachineCode)
	req.ClientVersion = strings.TrimSpace(req.ClientVersion)
	req.Nonce = strings.TrimSpace(req.Nonce)
}

func (req *secureHeartbeatReq) ensureRequired(envelopeNonce string, envelopeAppKey string) error {
	if req.Nonce == "" {
		req.Nonce = envelopeNonce
	}
	if req.Nonce != envelopeNonce {
		return errors.New("nonce_mismatch")
	}
	if req.AppKey != envelopeAppKey {
		return errors.New("app_key_mismatch")
	}
	if req.AppKey == "" || req.LicenseKey == "" || req.SessionToken == "" || req.MachineCode == "" || req.ClientVersion == "" || req.ClientVersionCode <= 0 {
		return errors.New("invalid_params")
	}
	return nil
}

func (req *secureAnnouncementReq) normalize() {
	req.AppKey = strings.TrimSpace(req.AppKey)
	req.LicenseKey = strings.TrimSpace(req.LicenseKey)
	req.MachineCode = strings.TrimSpace(req.MachineCode)
	req.Nonce = strings.TrimSpace(req.Nonce)
}

func (req *secureAnnouncementReq) ensureRequired(envelopeNonce string, envelopeAppKey string) error {
	if req.Nonce == "" {
		req.Nonce = envelopeNonce
	}
	if req.Nonce != envelopeNonce {
		return errors.New("nonce_mismatch")
	}
	if req.AppKey != envelopeAppKey {
		return errors.New("app_key_mismatch")
	}
	if req.AppKey == "" || req.LicenseKey == "" || req.MachineCode == "" {
		return errors.New("invalid_params")
	}
	return nil
}

func (req *secureUpdateReq) normalize() {
	req.AppKey = strings.TrimSpace(req.AppKey)
	req.LicenseKey = strings.TrimSpace(req.LicenseKey)
	req.MachineCode = strings.TrimSpace(req.MachineCode)
	req.ClientVersion = strings.TrimSpace(req.ClientVersion)
	req.Nonce = strings.TrimSpace(req.Nonce)
}

func (req *secureUpdateReq) ensureRequired(envelopeNonce string, envelopeAppKey string) error {
	if req.Nonce == "" {
		req.Nonce = envelopeNonce
	}
	if req.Nonce != envelopeNonce {
		return errors.New("nonce_mismatch")
	}
	if req.AppKey != envelopeAppKey {
		return errors.New("app_key_mismatch")
	}
	if req.AppKey == "" || req.LicenseKey == "" || req.MachineCode == "" || req.ClientVersion == "" || req.ClientVersionCode <= 0 {
		return errors.New("invalid_params")
	}
	return nil
}

func (req *secureReportReq) normalize() {
	req.AppKey = strings.TrimSpace(req.AppKey)
	req.LicenseKey = strings.TrimSpace(req.LicenseKey)
	req.MachineCode = strings.TrimSpace(req.MachineCode)
	req.Event = strings.TrimSpace(req.Event)
	req.Nonce = strings.TrimSpace(req.Nonce)
}

func (req *secureReportReq) ensureRequired(envelopeNonce string, envelopeAppKey string) error {
	if req.Nonce == "" {
		req.Nonce = envelopeNonce
	}
	if req.Nonce != envelopeNonce {
		return errors.New("nonce_mismatch")
	}
	if req.AppKey != envelopeAppKey {
		return errors.New("app_key_mismatch")
	}
	if req.AppKey == "" || req.LicenseKey == "" || req.MachineCode == "" || req.Data == nil {
		return errors.New("invalid_params")
	}
	return nil
}

func secureDerefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func secureDerefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func secureCleanStringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func secureFormatTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02 15:04:05")
}

func stringify(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case float64, bool:
		return fmt.Sprint(v)
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(raw)
	}
}
