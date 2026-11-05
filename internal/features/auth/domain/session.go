package domain

import (
	"context"
	"strings"
	"time"
)

type RegisterResponse struct {
	Token              string `json:"token"`
	RegistrationID     uint64 `json:"registration_id"`
	RegistrationStatus string `json:"registration_status"`
}

type LoginResponse struct {
	Token              string `json:"token"`
	TokenType          string `json:"token_type"`
	SessionID          string `json:"session_id"`
	AccessExpiresAt    string `json:"access_expires_at"`
	SessionExpiresAt   string `json:"session_expires_at"`
	MustChangePassword bool   `json:"must_change_password"`
}

// SessionResult is internal orchestration output, never an HTTP response.
type SessionResult struct {
	Token              string    `json:"-"`
	RefreshToken       string    `json:"-"`
	SessionID          string    `json:"-"`
	AccessExpiresAt    time.Time `json:"-"`
	SessionExpiresAt   time.Time `json:"-"`
	MustChangePassword bool      `json:"-"`
}

type LogoutResponse struct {
	Changed bool `json:"changed"`
}
type CSRFResponse struct {
	CSRFToken string `json:"csrf_token"`
}

// LogoutCredential holds exactly one selected proof. The HTTP adapter chooses it.
type LogoutCredential struct{ Authorization, RefreshToken string }

func (credential LogoutCredential) Valid() bool {
	if credential.Authorization != "" {
		return credential.RefreshToken == "" && ValidBearer(credential.Authorization)
	}
	return credential.RefreshToken != ""
}

func ValidBearer(authorization string) bool {
	parts := strings.Split(authorization, " ")
	return len(parts) == 2 && parts[0] == "Bearer" && parts[1] != "" && !strings.ContainsAny(parts[1], "\t\r\n,")
}

type RefreshGateway interface {
	Refresh(context.Context, string) (SessionResult, error)
}
type LogoutGateway interface {
	Logout(context.Context, LogoutCredential) (LogoutResponse, error)
}
