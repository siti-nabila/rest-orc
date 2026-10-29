package test_scenarios

import (
	"encoding/json"

	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
	"testing"
)

func SessionScenarios() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "session DTOs expose only safe exact fields", Run: sessionSerialization},
	}
}

func sessionSerialization(t *testing.T) {
	for _, tc := range []struct {
		value any
		want  string
	}{
		{domain.SessionResult{Token: "access-secret", RefreshToken: "refresh-secret"}, `{}`},
		{domain.RegisterResponse{RegistrationID: 17, RegistrationStatus: "PENDING"}, `{"token":"","registration_id":17,"registration_status":"PENDING"}`},
		{domain.LogoutResponse{}, `{"changed":false}`},
		{domain.LoginResponse{Token: "access", TokenType: "Bearer", SessionID: "session", AccessExpiresAt: "2026-10-29T04:00:00.123Z", SessionExpiresAt: "2026-10-30T04:00:00Z"}, `{"token":"access","token_type":"Bearer","session_id":"session","access_expires_at":"2026-10-29T04:00:00.123Z","session_expires_at":"2026-10-30T04:00:00Z","must_change_password":false}`},
	} {
		b, err := json.Marshal(tc.value)
		if err != nil || string(b) != tc.want {
			t.Fatalf("serialization %s want %s (%v)", b, tc.want, err)
		}
	}
}
