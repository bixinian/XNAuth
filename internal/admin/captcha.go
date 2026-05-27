package admin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

type captchaChallenge struct {
	expireAt time.Time
}

type captchaToken struct {
	challengeID string
	expireAt    time.Time
}

type CaptchaStore struct {
	mu         sync.Mutex
	challenges map[string]captchaChallenge
	tokens     map[string]captchaToken
}

func NewCaptchaStore() *CaptchaStore {
	return &CaptchaStore{
		challenges: map[string]captchaChallenge{},
		tokens:     map[string]captchaToken{},
	}
}

func (s *CaptchaStore) Generate() string {
	id := randomHex()
	s.mu.Lock()
	s.pruneLocked(time.Now())
	s.challenges[id] = captchaChallenge{
		expireAt: time.Now().Add(3 * time.Minute),
	}
	s.mu.Unlock()

	return id
}

func (s *CaptchaStore) VerifySlider(id string, sliderValue int) (string, bool) {
	id = strings.TrimSpace(id)
	if id == "" || sliderValue < 98 {
		return "", false
	}

	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	challenge, ok := s.challenges[id]
	delete(s.challenges, id)
	if !ok || challenge.expireAt.Before(now) {
		return "", false
	}
	token := randomHex()
	s.tokens[token] = captchaToken{
		challengeID: id,
		expireAt:    now.Add(2 * time.Minute),
	}
	return token, true
}

func (s *CaptchaStore) VerifyToken(id string, token string) bool {
	id = strings.TrimSpace(id)
	token = strings.TrimSpace(token)
	if id == "" || token == "" {
		return false
	}

	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
	item, ok := s.tokens[token]
	delete(s.tokens, token)
	if !ok || item.challengeID != id || item.expireAt.Before(now) {
		return false
	}
	return true
}

func (s *CaptchaStore) pruneLocked(now time.Time) {
	for id, challenge := range s.challenges {
		if challenge.expireAt.Before(now) {
			delete(s.challenges, id)
		}
	}
	for token, item := range s.tokens {
		if item.expireAt.Before(now) {
			delete(s.tokens, token)
		}
	}
}

func randomHex() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err == nil {
		return hex.EncodeToString(raw)
	}
	return hex.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
}
