package signature

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Signer struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

func NewSigner(privateKeyBase64 string) (*Signer, error) {
	privateKeyBase64 = strings.TrimSpace(privateKeyBase64)
	if privateKeyBase64 == "" {
		return nil, fmt.Errorf("ed25519_private_key is empty")
	}

	raw, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode ed25519 private key: %w", err)
	}

	var privateKey ed25519.PrivateKey
	switch len(raw) {
	case ed25519.PrivateKeySize:
		privateKey = ed25519.PrivateKey(raw)
	case ed25519.SeedSize:
		privateKey = ed25519.NewKeyFromSeed(raw)
	default:
		return nil, fmt.Errorf("invalid ed25519 private key length: %d", len(raw))
	}

	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("derive ed25519 public key failed")
	}

	return &Signer{
		privateKey: privateKey,
		publicKey:  publicKey,
	}, nil
}

func (s *Signer) Enabled() bool {
	return s != nil && len(s.privateKey) == ed25519.PrivateKeySize
}

func (s *Signer) Sign(payload string) (string, error) {
	if !s.Enabled() {
		return "", fmt.Errorf("signature signer is not initialized")
	}
	signature := ed25519.Sign(s.privateKey, []byte(payload))
	return base64.StdEncoding.EncodeToString(signature), nil
}

func (s *Signer) Verify(payload string, signBase64 string) bool {
	if s == nil || len(s.publicKey) != ed25519.PublicKeySize {
		return false
	}
	signature, err := base64.StdEncoding.DecodeString(signBase64)
	if err != nil {
		return false
	}
	return ed25519.Verify(s.publicKey, []byte(payload), signature)
}

func (s *Signer) PublicKeyBase64() string {
	if s == nil || len(s.publicKey) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(s.publicKey)
}

// BuildCanonicalString 构造稳定的 key=value 签名原文，用于 Ed25519 签名。
// 字段名按字典序排序，字段名和字段值都会做查询参数转义，避免拼接歧义。
func BuildCanonicalString(data map[string]any) string {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(formatValue(data[key])))
	}
	return strings.Join(parts, "&")
}

func PayloadDigest(payload any) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func formatValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case []byte:
		return string(v)
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(raw)
	}
}
