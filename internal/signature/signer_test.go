package signature

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func TestSignerSignAndVerify(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	signer, err := NewSigner(base64.StdEncoding.EncodeToString(privateKey))
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}

	payload := "app_key=demo_app&status=valid"
	sign, err := signer.Sign(payload)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if !signer.Verify(payload, sign) {
		t.Fatal("signature should verify")
	}
	if signer.Verify(payload+"x", sign) {
		t.Fatal("modified payload should not verify")
	}
}

func TestBuildCanonicalString(t *testing.T) {
	got := BuildCanonicalString(map[string]any{
		"server_time": 1710000000,
		"app_key":     "demo app",
		"status":      "valid",
	})
	want := "app_key=demo+app&server_time=1710000000&status=valid"
	if got != want {
		t.Fatalf("canonical mismatch: got %q want %q", got, want)
	}
}
