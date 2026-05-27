package utils

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
)

func RandomLicenseKey() (string, error) {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	out := make([]byte, len(raw))
	for index, value := range raw {
		out[index] = letters[int(value)%len(letters)]
	}
	return string(out), nil
}

func RandomToken(prefix string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	if strings.TrimSpace(prefix) == "" {
		return token, nil
	}
	return prefix + "_" + token, nil
}
