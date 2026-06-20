package models

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// ShareLink represents a generated share link for a file or folder.
type ShareLink struct {
	Token     string    `json:"token"`
	Path      string    `json:"path"`      // relative path within shared dir
	IsDir     bool      `json:"is_dir"`
	Password  string    `json:"password,omitempty"` // empty = no password
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ShareStore is a thread-safe in-memory store for share links.
type ShareStore struct {
	mu     sync.RWMutex
	links  map[string]*ShareLink
	tokens map[string]string // path -> token (reverse lookup)
}

// NewShareStore creates a new share store.
func NewShareStore() *ShareStore {
	return &ShareStore{
		links:  make(map[string]*ShareLink),
		tokens: make(map[string]string),
	}
}

// GenerateToken creates a random hex token.
func GenerateToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Create creates a new share link.
func (s *ShareStore) Create(path string, isDir bool, password string, expiresIn time.Duration) (*ShareLink, error) {
	token, err := GenerateToken(16)
	if err != nil {
		return nil, err
	}

	link := &ShareLink{
		Token:     token,
		Path:      path,
		IsDir:     isDir,
		Password:  password,
		CreatedAt: time.Now(),
	}
	if expiresIn > 0 {
		link.ExpiresAt = time.Now().Add(expiresIn)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.links[token] = link
	s.tokens[path] = token
	return link, nil
}

// Get retrieves a share link by token.
func (s *ShareStore) Get(token string) (*ShareLink, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	link, ok := s.links[token]
	return link, ok
}

// Delete removes a share link by token.
func (s *ShareStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if link, ok := s.links[token]; ok {
		delete(s.tokens, link.Path)
		delete(s.links, token)
	}
}

// List returns all active (non-expired) share links.
func (s *ShareStore) List() []*ShareLink {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	var result []*ShareLink
	for _, link := range s.links {
		if !link.ExpiresAt.IsZero() && now.After(link.ExpiresAt) {
			continue
		}
		result = append(result, link)
	}
	return result
}

// Cleanup removes expired links.
func (s *ShareStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for token, link := range s.links {
		if !link.ExpiresAt.IsZero() && now.After(link.ExpiresAt) {
			delete(s.tokens, link.Path)
			delete(s.links, token)
		}
	}
}
