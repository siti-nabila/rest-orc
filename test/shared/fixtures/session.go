package fixtures

import (
	"bytes"
	"encoding/base64"

	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"github.com/siti-nabila/rest-orc/internal/config"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

const BrowserOrigin = "https://app.example.test"

func BrowserConfig() config.BrowserConfig {
	return config.BrowserConfig{AllowedOrigins: []string{BrowserOrigin}, CSRFSecret: base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{19}, 32))}
}
func SessionNow() time.Time { return time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC) }
func LoginResult() domain.SessionResult {
	return domain.SessionResult{Token: "login-token", RefreshToken: "private-refresh", SessionID: "00000000-0000-4000-8000-000000000017", AccessExpiresAt: SessionNow().Add(15 * time.Minute), SessionExpiresAt: SessionNow().Add(24 * time.Hour)}
}
func LoginTokenResponse() *userv1.UserTokenResponse {
	r := LoginResult()
	return &userv1.UserTokenResponse{Token: r.Token, RefreshToken: r.RefreshToken, SessionId: r.SessionID, AccessExpiresAt: timestamppb.New(r.AccessExpiresAt), SessionExpiresAt: timestamppb.New(r.SessionExpiresAt)}
}
func RegistrationResponse() *userv1.UserTokenResponse {
	return &userv1.UserTokenResponse{RegistrationId: 17, RegistrationStatus: "PENDING"}
}
