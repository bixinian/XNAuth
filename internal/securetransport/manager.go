package securetransport

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/crypto/hkdf"

	"xnauth/internal/signature"
)

const (
	protocolVersion = 1
	keyLength       = 32
	transportInfo   = "xnauth secure transport v1"
)

type Manager struct {
	keyID         string
	privateKey    *ecdh.PrivateKey
	publicKey     *ecdh.PublicKey
	publicKeyText string
	signer        *signature.Signer
	timestampSkew time.Duration
}

type RequestEnvelope struct {
	Version                  int    `json:"v" binding:"required"`
	KeyID                    string `json:"kid" binding:"required"`
	Action                   string `json:"action" binding:"required"`
	AppKey                   string `json:"app_key"`
	DeviceID                 uint64 `json:"device_id"`
	Timestamp                int64  `json:"timestamp" binding:"required"`
	Nonce                    string `json:"nonce" binding:"required"`
	BodyNonce                string `json:"body_nonce" binding:"required"`
	ClientEphemeralPublicKey string `json:"client_ephemeral_public_key" binding:"required"`
	Ciphertext               string `json:"ciphertext" binding:"required"`
	Sign                     string `json:"sign" binding:"required"`
}

type ResponseEnvelope struct {
	Version      int    `json:"v"`
	KeyID        string `json:"kid"`
	RequestNonce string `json:"request_nonce"`
	Nonce        string `json:"nonce"`
	Timestamp    int64  `json:"timestamp"`
	BodyNonce    string `json:"body_nonce"`
	Ciphertext   string `json:"ciphertext"`
	Sign         string `json:"sign"`
}

type PlainResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

type RequestContext struct {
	Envelope         RequestEnvelope
	responseKey      []byte
	ciphertextDigest string
}

// NewManagerFromKeys 使用单个应用的服务端密钥对创建传输管理器。
// 私钥必须从服务端存储读取，不能通过公开配置接口下发或回查。
func NewManagerFromKeys(keyID string, x25519PrivateKey string, ed25519PrivateKey string, timestampSkewSeconds int) (*Manager, error) {
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return nil, fmt.Errorf("secure_key_id_required")
	}
	if timestampSkewSeconds <= 0 {
		timestampSkewSeconds = 120
	}
	signer, err := signature.NewSigner(ed25519PrivateKey)
	if err != nil {
		return nil, err
	}

	privateKeyBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(x25519PrivateKey))
	if err != nil {
		return nil, fmt.Errorf("decode x25519 private key: %w", err)
	}
	privateKey, err := ecdh.X25519().NewPrivateKey(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("load x25519 private key: %w", err)
	}
	publicKey := privateKey.PublicKey()

	return &Manager{
		keyID:         keyID,
		privateKey:    privateKey,
		publicKey:     publicKey,
		publicKeyText: base64.StdEncoding.EncodeToString(publicKey.Bytes()),
		signer:        signer,
		timestampSkew: time.Duration(timestampSkewSeconds) * time.Second,
	}, nil
}

func (m *Manager) Enabled() bool {
	return m != nil && m.privateKey != nil && m.publicKey != nil
}

func (m *Manager) KeyID() string {
	if m == nil {
		return ""
	}
	return m.keyID
}

func (m *Manager) PublicKeyBase64() string {
	if m == nil {
		return ""
	}
	return m.publicKeyText
}

func (m *Manager) TimestampSkewSeconds() int {
	if m == nil {
		return 0
	}
	return int(m.timestampSkew / time.Second)
}

func (m *Manager) OpenRequest(env RequestEnvelope) (*RequestContext, []byte, error) {
	if !m.Enabled() {
		return nil, nil, fmt.Errorf("secure transport is not initialized")
	}
	if env.Version != protocolVersion {
		return nil, nil, fmt.Errorf("unsupported secure protocol version")
	}
	if env.KeyID != m.keyID {
		return nil, nil, fmt.Errorf("secure key id mismatch")
	}
	if strings.TrimSpace(env.Action) == "" || strings.TrimSpace(env.Nonce) == "" {
		return nil, nil, fmt.Errorf("invalid secure envelope")
	}
	if err := m.checkTimestamp(env.Timestamp); err != nil {
		return nil, nil, err
	}

	// 客户端每次请求都提交新的临时公钥；服务端用应用私钥与该临时公钥派生
	// 本次请求专用 AES 密钥，抓包得到的密文不能在后续直接复用为明文。
	clientPublicBytes, err := base64.StdEncoding.DecodeString(env.ClientEphemeralPublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("decode client ephemeral public key: %w", err)
	}
	clientPublicKey, err := ecdh.X25519().NewPublicKey(clientPublicBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("load client ephemeral public key: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(env.Ciphertext)
	if err != nil {
		return nil, nil, fmt.Errorf("decode ciphertext: %w", err)
	}
	bodyNonce, err := base64.StdEncoding.DecodeString(env.BodyNonce)
	if err != nil {
		return nil, nil, fmt.Errorf("decode body nonce: %w", err)
	}

	key, err := m.deriveKey(clientPublicKey)
	if err != nil {
		return nil, nil, err
	}
	aead, err := newAEAD(key)
	if err != nil {
		return nil, nil, err
	}
	if len(bodyNonce) != aead.NonceSize() {
		return nil, nil, fmt.Errorf("invalid body nonce size")
	}

	// 认证附加数据覆盖明文路由元数据。动作字段 action、随机数 nonce、时间戳、密钥 ID 或客户端临时公钥
	// 被篡改时，GCM 认证会直接失败。
	plaintext, err := aead.Open(nil, bodyNonce, ciphertext, []byte(requestAAD(env)))
	if err != nil {
		return nil, nil, fmt.Errorf("decrypt secure request failed")
	}

	return &RequestContext{
		Envelope:         env,
		responseKey:      key,
		ciphertextDigest: digestHex(ciphertext),
	}, plaintext, nil
}

// SealResponse 使用请求派生出的密钥加密标准接口响应信封。
// 客户端必须先验证 Ed25519 签名，再信任解密后的响应内容。
func (m *Manager) SealResponse(ctx *RequestContext, httpStatus int, code int, message string, data any) (int, ResponseEnvelope, error) {
	if !m.Enabled() {
		return httpStatus, ResponseEnvelope{}, fmt.Errorf("secure transport is not initialized")
	}
	if ctx == nil || len(ctx.responseKey) != keyLength {
		return httpStatus, ResponseEnvelope{}, fmt.Errorf("secure request context is missing")
	}
	if data == nil {
		data = map[string]any{}
	}

	raw, err := json.Marshal(PlainResponse{
		Code:    code,
		Message: message,
		Data:    data,
	})
	if err != nil {
		return httpStatus, ResponseEnvelope{}, err
	}
	aead, err := newAEAD(ctx.responseKey)
	if err != nil {
		return httpStatus, ResponseEnvelope{}, err
	}
	bodyNonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(bodyNonce); err != nil {
		return httpStatus, ResponseEnvelope{}, err
	}
	responseNonce := randomTextNonce()
	now := time.Now().Unix()
	env := ResponseEnvelope{
		Version:      protocolVersion,
		KeyID:        m.keyID,
		RequestNonce: ctx.Envelope.Nonce,
		Nonce:        responseNonce,
		Timestamp:    now,
		BodyNonce:    base64.StdEncoding.EncodeToString(bodyNonce),
	}
	ciphertext := aead.Seal(nil, bodyNonce, raw, []byte(responseAAD(env)))
	env.Ciphertext = base64.StdEncoding.EncodeToString(ciphertext)

	signPayload := responseSigningPayload(env, digestHex(ciphertext))
	sign, err := m.signer.Sign(signPayload)
	if err != nil {
		return httpStatus, ResponseEnvelope{}, err
	}
	env.Sign = sign
	return httpStatus, env, nil
}

func (ctx *RequestContext) VerifyClientSignature(publicKeyBase64 string) error {
	publicKeyBase64 = strings.TrimSpace(publicKeyBase64)
	if publicKeyBase64 == "" {
		return fmt.Errorf("device_public_key_required")
	}
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil || len(publicKeyBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid_device_public_key")
	}
	signBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(ctx.Envelope.Sign))
	if err != nil || len(signBytes) != ed25519.SignatureSize {
		return fmt.Errorf("invalid_request_signature")
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKeyBytes), []byte(ctx.RequestSigningPayload()), signBytes) {
		return fmt.Errorf("request_signature_invalid")
	}
	return nil
}

func (ctx *RequestContext) RequestSigningPayload() string {
	return requestSigningPayload(ctx.Envelope, ctx.ciphertextDigest)
}

func (m *Manager) checkTimestamp(ts int64) error {
	if ts <= 0 {
		return fmt.Errorf("invalid secure timestamp")
	}
	delta := time.Since(time.Unix(ts, 0))
	if delta < 0 {
		delta = -delta
	}
	if delta > m.timestampSkew {
		return fmt.Errorf("secure timestamp expired")
	}
	return nil
}

func (m *Manager) deriveKey(clientPublicKey *ecdh.PublicKey) ([]byte, error) {
	shared, err := m.privateKey.ECDH(clientPublicKey)
	if err != nil {
		return nil, fmt.Errorf("derive shared key failed: %w", err)
	}
	// HKDF 盐值纳入密钥 ID 和双方公钥，避免同一份原始 ECDH 结果跨应用或跨密钥轮换复用。
	saltSource := make([]byte, 0, len(clientPublicKey.Bytes())+len(m.publicKey.Bytes())+len(m.keyID)+32)
	saltSource = append(saltSource, []byte("xnauth-secure-transport-salt:")...)
	saltSource = append(saltSource, []byte(m.keyID)...)
	saltSource = append(saltSource, clientPublicKey.Bytes()...)
	saltSource = append(saltSource, m.publicKey.Bytes()...)
	salt := sha256.Sum256(saltSource)

	key := make([]byte, keyLength)
	reader := hkdf.New(sha256.New, shared, salt[:], []byte(transportInfo))
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func newAEAD(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func requestAAD(env RequestEnvelope) string {
	return signature.BuildCanonicalString(map[string]any{
		"v":                           env.Version,
		"kid":                         env.KeyID,
		"action":                      env.Action,
		"app_key":                     env.AppKey,
		"device_id":                   env.DeviceID,
		"timestamp":                   env.Timestamp,
		"nonce":                       env.Nonce,
		"body_nonce":                  env.BodyNonce,
		"client_ephemeral_public_key": env.ClientEphemeralPublicKey,
	})
}

func requestSigningPayload(env RequestEnvelope, ciphertextDigest string) string {
	// Ed25519 签名覆盖密文摘要而不是明文；服务端可以在暴露或记录明文前拒绝篡改请求。
	data := map[string]any{
		"v":                           env.Version,
		"kid":                         env.KeyID,
		"action":                      env.Action,
		"app_key":                     env.AppKey,
		"device_id":                   env.DeviceID,
		"timestamp":                   env.Timestamp,
		"nonce":                       env.Nonce,
		"body_nonce":                  env.BodyNonce,
		"client_ephemeral_public_key": env.ClientEphemeralPublicKey,
		"ciphertext_sha256":           ciphertextDigest,
	}
	return signature.BuildCanonicalString(data)
}

func responseAAD(env ResponseEnvelope) string {
	return signature.BuildCanonicalString(map[string]any{
		"v":             env.Version,
		"kid":           env.KeyID,
		"request_nonce": env.RequestNonce,
		"nonce":         env.Nonce,
		"timestamp":     env.Timestamp,
		"body_nonce":    env.BodyNonce,
	})
}

func responseSigningPayload(env ResponseEnvelope, ciphertextDigest string) string {
	data := map[string]any{
		"v":                 env.Version,
		"kid":               env.KeyID,
		"request_nonce":     env.RequestNonce,
		"nonce":             env.Nonce,
		"timestamp":         env.Timestamp,
		"body_nonce":        env.BodyNonce,
		"ciphertext_sha256": ciphertextDigest,
	}
	return signature.BuildCanonicalString(data)
}

func digestHex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func randomTextNonce() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("server-%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}
