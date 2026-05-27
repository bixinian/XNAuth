package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func MachineCodeHash(machineCode string) string {
	return SHA256Hex([]byte(strings.TrimSpace(machineCode)))
}

func SHA256Hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
