package model

import "time"

const (
	AppStatusEnabled  = 1
	AppStatusDisabled = 2
)

const (
	AdminStatusEnabled  = 1
	AdminStatusDisabled = 2
)

const (
	LicenseStatusInactive = 0
	LicenseStatusActive   = 1
	LicenseStatusExpired  = 2
	LicenseStatusFrozen   = 3
	LicenseStatusBanned   = 4
)

const (
	DeviceStatusNormal   = 1
	DeviceStatusDisabled = 2
	DeviceStatusUnbound  = 3
)

const (
	SessionStatusOnline  = 1
	SessionStatusOffline = 2
	SessionStatusRevoked = 3
	SessionStatusTimeout = 4
)

type App struct {
	ID                   uint64    `gorm:"column:id;primaryKey" json:"id"`
	AppKey               string    `gorm:"column:app_key" json:"app_key"`
	AppName              string    `gorm:"column:app_name" json:"app_name"`
	Status               int       `gorm:"column:status" json:"status"`
	MinLoginVersionCode  *int      `gorm:"column:min_login_version_code" json:"min_login_version_code"`
	ForceUpdate          int       `gorm:"column:force_update" json:"force_update"`
	SecureKeyID          *string   `gorm:"column:secure_key_id" json:"secure_key_id"`
	SecureX25519Private  *string   `gorm:"column:secure_x25519_private_key" json:"secure_x25519_private_key"`
	SecureX25519Public   *string   `gorm:"column:secure_x25519_public_key" json:"secure_x25519_public_key"`
	SecureEd25519Private *string   `gorm:"column:secure_ed25519_private_key" json:"secure_ed25519_private_key"`
	SecureEd25519Public  *string   `gorm:"column:secure_ed25519_public_key" json:"secure_ed25519_public_key"`
	Remark               *string   `gorm:"column:remark" json:"remark"`
	CreatedAt            time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt            time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (App) TableName() string {
	return "apps"
}

type AdminUser struct {
	ID           uint64     `gorm:"column:id;primaryKey" json:"id"`
	Username     string     `gorm:"column:username" json:"username"`
	PasswordHash string     `gorm:"column:password_hash" json:"-"`
	Status       int        `gorm:"column:status" json:"status"`
	LastLoginAt  *time.Time `gorm:"column:last_login_at" json:"last_login_at"`
	CreatedAt    time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (AdminUser) TableName() string {
	return "admin_users"
}

type SystemSetting struct {
	SettingKey   string    `gorm:"column:setting_key;primaryKey" json:"setting_key"`
	SettingValue string    `gorm:"column:setting_value" json:"setting_value"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (SystemSetting) TableName() string {
	return "system_settings"
}

type LicenseCard struct {
	ID          uint64     `gorm:"column:id;primaryKey" json:"id"`
	AppID       uint64     `gorm:"column:app_id" json:"app_id"`
	LicenseKey  string     `gorm:"column:license_key" json:"license_key"`
	Status      int        `gorm:"column:status" json:"status"`
	Remark      *string    `gorm:"column:remark" json:"remark"`
	MaxDevices  int        `gorm:"column:max_devices" json:"max_devices"`
	MaxOnline   int        `gorm:"column:max_online" json:"max_online"`
	ExpireAt    *time.Time `gorm:"column:expire_at" json:"expire_at"`
	ActivatedAt *time.Time `gorm:"column:activated_at" json:"activated_at"`
	CreatedAt   time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (LicenseCard) TableName() string {
	return "license_cards"
}

type LicenseDevice struct {
	ID                uint64     `gorm:"column:id;primaryKey" json:"id"`
	AppID             uint64     `gorm:"column:app_id" json:"app_id"`
	LicenseID         uint64     `gorm:"column:license_id" json:"license_id"`
	MachineCodeHash   string     `gorm:"column:machine_code_hash" json:"machine_code_hash"`
	DeviceName        *string    `gorm:"column:device_name" json:"device_name"`
	DevicePublicKey   *string    `gorm:"column:device_public_key" json:"device_public_key"`
	DeviceKeyBoundAt  *time.Time `gorm:"column:device_key_bound_at" json:"device_key_bound_at"`
	ClientVersion     *string    `gorm:"column:client_version" json:"client_version"`
	ClientVersionCode *int       `gorm:"column:client_version_code" json:"client_version_code"`
	Status            int        `gorm:"column:status" json:"status"`
	FirstSeenAt       time.Time  `gorm:"column:first_seen_at" json:"first_seen_at"`
	LastSeenAt        time.Time  `gorm:"column:last_seen_at" json:"last_seen_at"`
	CreatedAt         time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (LicenseDevice) TableName() string {
	return "license_devices"
}

type ClientNonce struct {
	ID        uint64    `gorm:"column:id;primaryKey" json:"id"`
	AppID     uint64    `gorm:"column:app_id" json:"app_id"`
	DeviceID  uint64    `gorm:"column:device_id" json:"device_id"`
	Nonce     string    `gorm:"column:nonce" json:"nonce"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (ClientNonce) TableName() string {
	return "client_nonces"
}

type LicenseSession struct {
	ID                uint64     `gorm:"column:id;primaryKey" json:"id"`
	AppID             uint64     `gorm:"column:app_id" json:"app_id"`
	LicenseID         uint64     `gorm:"column:license_id" json:"license_id"`
	DeviceID          uint64     `gorm:"column:device_id" json:"device_id"`
	SessionToken      string     `gorm:"column:session_token" json:"session_token"`
	Status            int        `gorm:"column:status" json:"status"`
	ClientIP          *string    `gorm:"column:client_ip" json:"client_ip"`
	ClientVersion     *string    `gorm:"column:client_version" json:"client_version"`
	ClientVersionCode *int       `gorm:"column:client_version_code" json:"client_version_code"`
	StartedAt         time.Time  `gorm:"column:started_at" json:"started_at"`
	LastHeartbeatAt   time.Time  `gorm:"column:last_heartbeat_at" json:"last_heartbeat_at"`
	RevokedAt         *time.Time `gorm:"column:revoked_at" json:"revoked_at"`
	RevokeReason      *string    `gorm:"column:revoke_reason" json:"revoke_reason"`
	CreatedAt         time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (LicenseSession) TableName() string {
	return "license_sessions"
}

type AppAnnouncement struct {
	ID         uint64     `gorm:"column:id;primaryKey" json:"id"`
	AppID      uint64     `gorm:"column:app_id" json:"app_id"`
	Title      string     `gorm:"column:title" json:"title"`
	Content    string     `gorm:"column:content" json:"content"`
	NoticeType *string    `gorm:"column:notice_type" json:"notice_type"`
	Popup      int        `gorm:"column:popup" json:"popup"`
	Enabled    int        `gorm:"column:enabled" json:"enabled"`
	StartAt    *time.Time `gorm:"column:start_at" json:"start_at"`
	EndAt      *time.Time `gorm:"column:end_at" json:"end_at"`
	SortOrder  int        `gorm:"column:sort_order" json:"sort_order"`
	CreatedAt  time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (AppAnnouncement) TableName() string {
	return "app_announcements"
}

type AppVersion struct {
	ID               uint64     `gorm:"column:id;primaryKey" json:"id"`
	AppID            uint64     `gorm:"column:app_id" json:"app_id"`
	VersionName      string     `gorm:"column:version_name" json:"version_name"`
	VersionCode      int        `gorm:"column:version_code" json:"version_code"`
	MinSupportedCode *int       `gorm:"column:min_supported_code" json:"min_supported_code"`
	DownloadURL      *string    `gorm:"column:download_url" json:"download_url"`
	FileHash         *string    `gorm:"column:file_hash" json:"file_hash"`
	FileSize         *int64     `gorm:"column:file_size" json:"file_size"`
	Changelog        *string    `gorm:"column:changelog" json:"changelog"`
	ForceUpdate      int        `gorm:"column:force_update" json:"force_update"`
	Enabled          int        `gorm:"column:enabled" json:"enabled"`
	ReleasedAt       *time.Time `gorm:"column:released_at" json:"released_at"`
	CreatedAt        time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

func (AppVersion) TableName() string {
	return "app_versions"
}

type VerifyLog struct {
	ID                uint64    `gorm:"column:id;primaryKey" json:"id"`
	AppID             uint64    `gorm:"column:app_id" json:"app_id"`
	LicenseID         *uint64   `gorm:"column:license_id" json:"license_id"`
	DeviceID          *uint64   `gorm:"column:device_id" json:"device_id"`
	LicenseKey        *string   `gorm:"column:license_key" json:"license_key"`
	MachineCodeHash   *string   `gorm:"column:machine_code_hash" json:"machine_code_hash"`
	ClientIP          *string   `gorm:"column:client_ip" json:"client_ip"`
	ClientVersion     *string   `gorm:"column:client_version" json:"client_version"`
	ClientVersionCode *int      `gorm:"column:client_version_code" json:"client_version_code"`
	Result            int       `gorm:"column:result" json:"result"`
	FailReason        *string   `gorm:"column:fail_reason" json:"fail_reason"`
	CreatedAt         time.Time `gorm:"column:created_at" json:"created_at"`
}

func (VerifyLog) TableName() string {
	return "verify_logs"
}

type CollectField struct {
	ID            uint64    `gorm:"column:id;primaryKey" json:"id"`
	AppID         uint64    `gorm:"column:app_id" json:"app_id"`
	FieldKey      string    `gorm:"column:field_key" json:"field_key"`
	FieldName     string    `gorm:"column:field_name" json:"field_name"`
	Enabled       int       `gorm:"column:enabled" json:"enabled"`
	ShowInList    int       `gorm:"column:show_in_list" json:"show_in_list"`
	StatEnabled   int       `gorm:"column:stat_enabled" json:"stat_enabled"`
	StatType      string    `gorm:"column:stat_type" json:"stat_type"`
	SearchEnabled int       `gorm:"column:search_enabled" json:"search_enabled"`
	SortOrder     int       `gorm:"column:sort_order" json:"sort_order"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (CollectField) TableName() string {
	return "collect_fields"
}

type CollectRecord struct {
	ID              uint64    `gorm:"column:id;primaryKey" json:"id"`
	AppID           uint64    `gorm:"column:app_id" json:"app_id"`
	LicenseID       *uint64   `gorm:"column:license_id" json:"license_id"`
	DeviceID        *uint64   `gorm:"column:device_id" json:"device_id"`
	LicenseKey      string    `gorm:"column:license_key" json:"license_key"`
	MachineCodeHash *string   `gorm:"column:machine_code_hash" json:"machine_code_hash"`
	Event           *string   `gorm:"column:event" json:"event"`
	ClientIP        *string   `gorm:"column:client_ip" json:"client_ip"`
	UserAgent       *string   `gorm:"column:user_agent" json:"user_agent"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
}

func (CollectRecord) TableName() string {
	return "collect_records"
}

type CollectRecordValue struct {
	ID         uint64    `gorm:"column:id;primaryKey" json:"id"`
	RecordID   uint64    `gorm:"column:record_id" json:"record_id"`
	AppID      uint64    `gorm:"column:app_id" json:"app_id"`
	LicenseID  *uint64   `gorm:"column:license_id" json:"license_id"`
	DeviceID   *uint64   `gorm:"column:device_id" json:"device_id"`
	FieldKey   string    `gorm:"column:field_key" json:"field_key"`
	FieldValue string    `gorm:"column:field_value" json:"field_value"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`
}

func (CollectRecordValue) TableName() string {
	return "collect_record_values"
}

type OperationLog struct {
	ID        uint64    `gorm:"column:id;primaryKey" json:"id"`
	AdminID   *uint64   `gorm:"column:admin_id" json:"admin_id"`
	Module    string    `gorm:"column:module" json:"module"`
	Action    string    `gorm:"column:action" json:"action"`
	TargetID  *uint64   `gorm:"column:target_id" json:"target_id"`
	Content   *string   `gorm:"column:content" json:"content"`
	ClientIP  *string   `gorm:"column:client_ip" json:"client_ip"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (OperationLog) TableName() string {
	return "operation_logs"
}
