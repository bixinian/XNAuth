package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig          `yaml:"server"`
	MySQL  MySQLConfig           `yaml:"mysql"`
	JWT    JWTConfig             `yaml:"jwt"`
	Secure SecureTransportConfig `yaml:"secure_transport"`
	Auth   AuthConfig            `yaml:"auth"`
	Log    LogConfig             `yaml:"log"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
	Mode string `yaml:"mode"`
}

type MySQLConfig struct {
	DSN                    string `yaml:"dsn"`
	MaxOpenConns           int    `yaml:"max_open_conns"`
	MaxIdleConns           int    `yaml:"max_idle_conns"`
	ConnMaxLifetimeMinutes int    `yaml:"conn_max_lifetime_minutes"`
}

type JWTConfig struct {
	Secret      string `yaml:"secret"`
	ExpireHours int    `yaml:"expire_hours"`
}

type SecureTransportConfig struct {
	Enabled              bool `yaml:"enabled"`
	TimestampSkewSeconds int  `yaml:"timestamp_skew_seconds"`
}

type AuthConfig struct {
	HeartbeatIntervalSeconds int `yaml:"heartbeat_interval_seconds"`
	SessionTimeoutSeconds    int `yaml:"session_timeout_seconds"`
}

type LogConfig struct {
	Level string `yaml:"level"`
	Path  string `yaml:"path"`
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			Addr: "0.0.0.0:8080",
			Mode: "release",
		},
		MySQL: MySQLConfig{
			MaxOpenConns:           50,
			MaxIdleConns:           10,
			ConnMaxLifetimeMinutes: 30,
		},
		JWT: JWTConfig{
			ExpireHours: 24,
		},
		Secure: SecureTransportConfig{
			Enabled:              false,
			TimestampSkewSeconds: 120,
		},
		Auth: AuthConfig{
			HeartbeatIntervalSeconds: 30,
			SessionTimeoutSeconds:    90,
		},
		Log: LogConfig{
			Level: "info",
			Path:  "./logs/app.log",
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	cfg.applyEnv()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (cfg *Config) applyEnv() {
	setString("XNAUTH_SERVER_ADDR", &cfg.Server.Addr)
	setString("XNAUTH_SERVER_MODE", &cfg.Server.Mode)
	setString("XNAUTH_MYSQL_DSN", &cfg.MySQL.DSN)
	setString("XNAUTH_JWT_SECRET", &cfg.JWT.Secret)
	setInt("XNAUTH_JWT_EXPIRE_HOURS", &cfg.JWT.ExpireHours)
	setBool("XNAUTH_SECURE_TRANSPORT_ENABLED", &cfg.Secure.Enabled)
	setInt("XNAUTH_SECURE_TRANSPORT_TIMESTAMP_SKEW_SECONDS", &cfg.Secure.TimestampSkewSeconds)
	setString("XNAUTH_LOG_LEVEL", &cfg.Log.Level)
	setString("XNAUTH_LOG_PATH", &cfg.Log.Path)
}

func (cfg Config) validate() error {
	if strings.TrimSpace(cfg.Server.Addr) == "" {
		return fmt.Errorf("server.addr is required")
	}
	if strings.TrimSpace(cfg.MySQL.DSN) == "" {
		return fmt.Errorf("mysql.dsn is required")
	}
	if strings.TrimSpace(cfg.JWT.Secret) == "" {
		return fmt.Errorf("jwt.secret is required")
	}
	if cfg.Auth.HeartbeatIntervalSeconds <= 0 {
		return fmt.Errorf("auth.heartbeat_interval_seconds must be greater than 0")
	}
	if cfg.Auth.SessionTimeoutSeconds <= 0 {
		return fmt.Errorf("auth.session_timeout_seconds must be greater than 0")
	}
	if cfg.Secure.Enabled {
		if cfg.Secure.TimestampSkewSeconds <= 0 {
			return fmt.Errorf("secure_transport.timestamp_skew_seconds must be greater than 0")
		}
	}
	return nil
}

func setString(name string, target *string) {
	if value, ok := os.LookupEnv(name); ok {
		*target = value
	}
}

func setInt(name string, target *int) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return
	}
	parsed, err := strconv.Atoi(value)
	if err == nil {
		*target = parsed
	}
}

func setBool(name string, target *bool) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		*target = true
	case "0", "false", "no", "off":
		*target = false
	}
}
