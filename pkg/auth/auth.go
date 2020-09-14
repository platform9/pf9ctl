package auth

import (
	"time"
)

type Authenticator interface {
	Auth(credentials Credentials) (TokenInfo, error)
}

type Credentials struct {
	Username string
	Password string
	// TODO add field to capture tenant
}

type TokenInfo struct {
	Token     string
	ExpiresAt time.Time
}