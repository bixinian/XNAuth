package adminapi

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"xnauth/internal/signature"
	"xnauth/pkg/utils"
)

type appSecurityKeyInput struct {
	KeyID             string
	X25519PrivateKey  string
	X25519PublicKey   string
	Ed25519PrivateKey string
	Ed25519PublicKey  string
}

type appSecurityKeys struct {
	KeyID             string
	X25519PrivateKey  string
	X25519PublicKey   string
	Ed25519PrivateKey string
	Ed25519PublicKey  string
}

func generateAppSecurityKeys() (appSecurityKeys, error) {
	keyID, err := utils.RandomToken("appkey")
	if err != nil {
		return appSecurityKeys{}, err
	}
	xPrivate, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return appSecurityKeys{}, err
	}
	ePublic, ePrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return appSecurityKeys{}, err
	}
	return appSecurityKeys{
		KeyID:             keyID,
		X25519PrivateKey:  base64.StdEncoding.EncodeToString(xPrivate.Bytes()),
		X25519PublicKey:   base64.StdEncoding.EncodeToString(xPrivate.PublicKey().Bytes()),
		Ed25519PrivateKey: base64.StdEncoding.EncodeToString(ePrivate),
		Ed25519PublicKey:  base64.StdEncoding.EncodeToString(ePublic),
	}, nil
}

// normalizeAppSecurityKeys 接受管理员填写的私钥，但服务端会重新推导并校验公钥。
// 这样可以避免保存私钥和公钥不匹配的应用配置。
func normalizeAppSecurityKeys(input appSecurityKeyInput, generateWhenEmpty bool) (appSecurityKeys, error) {
	input.KeyID = strings.TrimSpace(input.KeyID)
	input.X25519PrivateKey = strings.TrimSpace(input.X25519PrivateKey)
	input.X25519PublicKey = strings.TrimSpace(input.X25519PublicKey)
	input.Ed25519PrivateKey = strings.TrimSpace(input.Ed25519PrivateKey)
	input.Ed25519PublicKey = strings.TrimSpace(input.Ed25519PublicKey)

	if input.KeyID == "" && input.X25519PrivateKey == "" && input.X25519PublicKey == "" && input.Ed25519PrivateKey == "" && input.Ed25519PublicKey == "" {
		if !generateWhenEmpty {
			return appSecurityKeys{}, nil
		}
		return generateAppSecurityKeys()
	}
	if input.X25519PrivateKey == "" || input.Ed25519PrivateKey == "" {
		return appSecurityKeys{}, errors.New("invalid_secure_keys")
	}
	if input.KeyID == "" {
		keyID, err := utils.RandomToken("appkey")
		if err != nil {
			return appSecurityKeys{}, err
		}
		input.KeyID = keyID
	}

	xPrivateRaw, err := base64.StdEncoding.DecodeString(input.X25519PrivateKey)
	if err != nil {
		return appSecurityKeys{}, errors.New("invalid_x25519_private_key")
	}
	xPrivate, err := ecdh.X25519().NewPrivateKey(xPrivateRaw)
	if err != nil {
		return appSecurityKeys{}, errors.New("invalid_x25519_private_key")
	}
	xPublic := base64.StdEncoding.EncodeToString(xPrivate.PublicKey().Bytes())
	if input.X25519PublicKey != "" && input.X25519PublicKey != xPublic {
		return appSecurityKeys{}, errors.New("x25519_public_key_mismatch")
	}

	eSigner, err := signature.NewSigner(input.Ed25519PrivateKey)
	if err != nil {
		return appSecurityKeys{}, errors.New("invalid_ed25519_private_key")
	}
	ePublic := eSigner.PublicKeyBase64()
	if input.Ed25519PublicKey != "" && input.Ed25519PublicKey != ePublic {
		return appSecurityKeys{}, errors.New("ed25519_public_key_mismatch")
	}

	return appSecurityKeys{
		KeyID:             input.KeyID,
		X25519PrivateKey:  input.X25519PrivateKey,
		X25519PublicKey:   xPublic,
		Ed25519PrivateKey: input.Ed25519PrivateKey,
		Ed25519PublicKey:  ePublic,
	}, nil
}

func appSecurityKeyUpdates(keys appSecurityKeys) map[string]any {
	return map[string]any{
		"secure_key_id":              keys.KeyID,
		"secure_x25519_private_key":  keys.X25519PrivateKey,
		"secure_x25519_public_key":   keys.X25519PublicKey,
		"secure_ed25519_private_key": keys.Ed25519PrivateKey,
		"secure_ed25519_public_key":  keys.Ed25519PublicKey,
	}
}
