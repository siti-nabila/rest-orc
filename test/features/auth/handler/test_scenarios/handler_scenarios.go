package test_scenarios

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	requestauth "github.com/siti-nabila/rest-orc/internal/auth"
	authhandler "github.com/siti-nabila/rest-orc/internal/features/auth/handler"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
	"github.com/siti-nabila/rest-orc/test/shared/fixtures"
	"github.com/siti-nabila/rest-orc/test/shared/testutils"
)

func All() []testutils.Scenario {
	return []testutils.Scenario{
		{Name: "Register public route maps request and response", Run: registerPublicRouteMapsRequestAndResponse},
		{Name: "Login public route maps request and response", Run: loginPublicRouteMapsRequestAndResponse},
		{Name: "Register rejects malformed JSON before usecase", Run: registerRejectsMalformedJSON},
		{Name: "Login rejects malformed JSON before usecase", Run: loginRejectsMalformedJSON},
		{Name: "auth constructor rejects nil usecases", Run: constructorRejectsNilAuthUseCases},
		{Name: "Me writes only the complete REST contract", Run: completeMeResponse},
		{Name: "Me empty and restricted snapshots have stable JSON collections", Run: emptyMeResponse},
		{Name: "Me missing principal is unauthorized", Run: missingPrincipal},
		{Name: "Me constructor rejects nil writer", Run: nilWriter},
	}
}

func completeMeResponse(t *testing.T) {
	p := fixtures.MePrincipal()
	local := p.CreatedAt.In(time.FixedZone("UTC+7", 7*60*60))
	p.CreatedAt = &local
	got := meBody(t, &p, http.StatusOK)
	want := `{"code":"SS","message":"Profile retrieved successfully.","data":{
		"id":17,"email":"member@example.com","fullname":"Member User","address":"Jakarta",
		"phone":"+62+628123","country_code":"+62","created_at":"2026-10-20T08:09:10.123456789Z",
		"status":"ACTIVE","version":"9007199254740993","must_change_password":false,
		"role_codes":[3],"role_names":["member"],
		"roles":[
			{"role_id":9,"code":"clm.member","name":"Member","status":"ACTIVE","role_status":"ACTIVE"},
			{"role_id":11,"code":"clm.reader","name":"Reader","status":"ACTIVE","role_status":"ACTIVE"}],
		"application_access":[{"application_id":5,"code":"clm","name":"CLM","status":"ACTIVE","application_status":"ACTIVE"}],
		"permissions":{"clm.document.read":true,"clm.document.delete":false}
	}}`
	assertJSON(t, got, want)
}

func emptyMeResponse(t *testing.T) {
	for _, restricted := range []bool{false, true} {
		p := requestauth.Principal{ID: 17, MustChangePassword: restricted}
		body := meBody(t, &p, http.StatusOK)
		var envelope struct {
			Code string                     `json:"code"`
			Data map[string]json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal([]byte(body), &envelope); err != nil {
			t.Fatal(err)
		}
		for field, want := range map[string]string{
			"role_codes": "[]", "role_names": "[]", "roles": "[]", "application_access": "[]", "permissions": "{}",
			"created_at": "null", "version": `"0"`,
		} {
			if got := string(envelope.Data[field]); got != want {
				t.Errorf("restricted=%v %s=%s, want %s", restricted, field, got, want)
			}
		}
		var flag bool
		if err := json.Unmarshal(envelope.Data["must_change_password"], &flag); err != nil || flag != restricted || envelope.Code != "SS" {
			t.Fatalf("restricted marker/envelope = %s", body)
		}
	}
}

func missingPrincipal(t *testing.T) {
	meBody(t, nil, http.StatusUnauthorized)
}

func nilWriter(t *testing.T) {
	handler, err := authhandler.New(unexpectedAuthUseCase(t), unexpectedLoginUseCase(t), unexpectedPasswordUseCase(t), nil, fixtures.NewBrowserPolicy(t))
	if handler != nil || !errors.Is(err, dictionary.ErrAuthHandlerResponseWriterRequired) {
		t.Fatalf("New(nil) = (%v, %v)", handler, err)
	}
}

func meBody(t *testing.T, principal *requestauth.Principal, status int) string {
	t.Helper()
	app, writer := testutils.NewHTTPApp()
	handler, err := authhandler.New(unexpectedAuthUseCase(t), unexpectedLoginUseCase(t), unexpectedPasswordUseCase(t), writer, fixtures.NewBrowserPolicy(t))
	if err != nil {
		t.Fatal(err)
	}
	app.Use(func(ctx fiber.Ctx) error {
		if principal != nil {
			ctx.SetContext(requestauth.WithPrincipal(context.Background(), *principal))
		}
		return ctx.Next()
	})
	handler.RegisterAuthenticatedRoutes(app.Group(authhandler.MeRoute))
	response, err := app.Test(httptest.NewRequest(http.MethodGet, authhandler.MeRoute, nil))
	body := testutils.ResponseBody(t, response, err)
	if response.StatusCode != status {
		t.Fatalf("status=%d, want %d; body=%s", response.StatusCode, status, body)
	}
	return body
}

func assertJSON(t *testing.T, got, want string) {
	t.Helper()
	var actual, expected any
	if err := json.Unmarshal([]byte(got), &actual); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(want), &expected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("JSON = %s, want %s", got, want)
	}
}
