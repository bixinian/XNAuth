package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"xnauth/internal/config"
)

type Claims struct {
	AdminID  uint64 `json:"admin_id"`
	Username string `json:"username"`
	IssuedAt int64  `json:"iat"`
	ExpireAt int64  `json:"exp"`
}

type Manager struct {
	secret []byte
	expire time.Duration
}

func NewManager(cfg config.JWTConfig) *Manager {
	expireHours := cfg.ExpireHours
	if expireHours <= 0 {
		expireHours = 24
	}
	return &Manager{
		secret: []byte(cfg.Secret),
		expire: time.Duration(expireHours) * time.Hour,
	}
}

func (m *Manager) Generate(adminID uint64, username string) (string, int64, error) {
	now := time.Now()
	expireAt := now.Add(m.expire).Unix()
	claims := Claims{
		AdminID:  adminID,
		Username: username,
		IssuedAt: now.Unix(),
		ExpireAt: expireAt,
	}

	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	headerRaw, err := json.Marshal(header)
	if err != nil {
		return "", 0, err
	}
	payloadRaw, err := json.Marshal(claims)
	if err != nil {
		return "", 0, err
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerRaw)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadRaw)
	signingInput := encodedHeader + "." + encodedPayload
	signature := m.sign(signingInput)

	return signingInput + "." + signature, expireAt, nil
}

func (m *Manager) Parse(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	signingInput := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(m.sign(signingInput)), []byte(parts[2])) {
		return nil, fmt.Errorf("invalid token signature")
	}

	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode token payload: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(payloadRaw, &claims); err != nil {
		return nil, fmt.Errorf("parse token payload: %w", err)
	}
	if claims.AdminID == 0 || strings.TrimSpace(claims.Username) == "" {
		return nil, fmt.Errorf("invalid token claims")
	}
	if claims.ExpireAt <= time.Now().Unix() {
		return nil, fmt.Errorf("token expired")
	}
	return &claims, nil
}

func (m *Manager) sign(input string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func AdminIDFromAny(value any) (uint64, bool) {
	switch v := value.(type) {
	case uint64:
		return v, true
	case uint:
		return uint64(v), true
	case int:
		if v > 0 {
			return uint64(v), true
		}
	case int64:
		if v > 0 {
			return uint64(v), true
		}
	case string:
		parsed, err := strconv.ParseUint(v, 10, 64)
		return parsed, err == nil
	}
	return 0, false
}
