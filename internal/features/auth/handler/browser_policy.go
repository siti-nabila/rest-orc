package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"io"
	"mime"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/siti-nabila/api-contracts/pkg/dictionary/common"
	"github.com/siti-nabila/rest-orc/internal/config"
	"github.com/siti-nabila/rest-orc/internal/features/auth/domain"
	"github.com/siti-nabila/rest-orc/pkg/dictionary"
)

const RefreshCookie = "__Secure-refresh"
const CSRFCookie = "__Secure-csrf"
const CSRFHeader = "X-CSRF-Token"

// BrowserPolicy owns only HTTP credential transport, not session validity.
type BrowserPolicy struct {
	origins   map[string]bool
	secret    []byte
	path      string
	now       func() time.Time
	entropy   io.Reader
	entropyMu sync.Mutex
}

func NewBrowserPolicy(cfg config.BrowserConfig, cookiePath string, now func() time.Time, entropy io.Reader) (*BrowserPolicy, error) {
	if err := cfg.Validate(""); err != nil {
		return nil, err
	}
	if now == nil || entropy == nil || !strings.HasPrefix(cookiePath, "/") || strings.ContainsAny(cookiePath, ";\r\n") {
		return nil, dictionary.ErrBrowserPolicyRequired
	}
	secret, _ := base64.RawURLEncoding.DecodeString(cfg.CSRFSecret)
	origins := make(map[string]bool, len(cfg.AllowedOrigins))
	for _, origin := range cfg.AllowedOrigins {
		origins[origin] = true
	}
	return &BrowserPolicy{origins: origins, secret: secret, path: cookiePath, now: now, entropy: entropy}, nil
}

func (p *BrowserPolicy) ValidateOrigin(ctx fiber.Ctx, required bool) error {
	values := ctx.Request().Header.PeekAll(fiber.HeaderOrigin)
	if len(values) == 0 && !required {
		return nil
	}
	if len(values) != 1 || !p.origins[string(values[0])] {
		return common.ErrForbidden
	}
	return nil
}

func (p *BrowserPolicy) ValidateLogin(ctx fiber.Ctx) error {
	if err := p.ValidateOrigin(ctx, true); err != nil {
		return err
	}
	values := ctx.Request().Header.PeekAll(fiber.HeaderContentType)
	if len(values) != 1 {
		return fiber.ErrUnsupportedMediaType
	}
	media, _, err := mime.ParseMediaType(string(values[0]))
	if err != nil || media != fiber.MIMEApplicationJSON {
		return fiber.ErrUnsupportedMediaType
	}
	return nil
}

func (p *BrowserPolicy) PrepareNonce() ([]byte, error) {
	nonce := make([]byte, 32)
	p.entropyMu.Lock()
	_, err := io.ReadFull(p.entropy, nonce)
	p.entropyMu.Unlock()
	if err != nil {
		return nil, dictionary.ErrCSRFGenerationFailed
	}
	return nonce, nil
}

func (p *BrowserPolicy) signature(refresh string, nonce []byte) []byte {
	digest := sha256.Sum256([]byte(refresh))
	mac := hmac.New(sha256.New, p.secret)
	for _, part := range [][]byte{[]byte("v1"), nonce, digest[:]} {
		var size [4]byte
		binary.BigEndian.PutUint32(size[:], uint32(len(part)))
		mac.Write(size[:])
		mac.Write(part)
	}
	return mac.Sum(nil)
}

func (p *BrowserPolicy) csrfToken(refresh string, nonce []byte) string {
	return "v1." + base64.RawURLEncoding.EncodeToString(nonce) + "." + base64.RawURLEncoding.EncodeToString(p.signature(refresh, nonce))
}

func (p *BrowserPolicy) verifyCSRF(ctx fiber.Ctx, refresh string) error {
	cookie, ok := uniqueCookie(ctx, CSRFCookie)
	values := ctx.Request().Header.PeekAll(CSRFHeader)
	if !ok || len(values) != 1 || subtle.ConstantTimeCompare([]byte(cookie), values[0]) != 1 {
		return common.ErrForbidden
	}
	parts := strings.Split(cookie, ".")
	if len(parts) != 3 || parts[0] != "v1" || len(parts[1]) != 43 || len(parts[2]) != 43 {
		return common.ErrForbidden
	}
	nonce, e1 := base64.RawURLEncoding.Strict().DecodeString(parts[1])
	mac, e2 := base64.RawURLEncoding.Strict().DecodeString(parts[2])
	if e1 != nil || e2 != nil || len(nonce) != 32 || len(mac) != 32 || base64.RawURLEncoding.EncodeToString(nonce) != parts[1] || base64.RawURLEncoding.EncodeToString(mac) != parts[2] || !hmac.Equal(mac, p.signature(refresh, nonce)) {
		return common.ErrForbidden
	}
	return nil
}

func uniqueCookie(ctx fiber.Ctx, name string) (string, bool) {
	count := 0
	value := ""
	for key, raw := range ctx.Request().Header.Cookies() {
		if string(key) == name {
			count++
			value = string(raw)
		}
	}
	return value, count == 1 && value != ""
}

func (p *BrowserPolicy) RefreshCredential(ctx fiber.Ctx) (string, error) {
	if err := p.ValidateOrigin(ctx, true); err != nil {
		return "", err
	}
	refresh, ok := uniqueCookie(ctx, RefreshCookie)
	if !ok {
		return "", common.ErrUnauthorized
	}
	if err := p.verifyCSRF(ctx, refresh); err != nil {
		return "", err
	}
	return refresh, nil
}

func (p *BrowserPolicy) LogoutCredential(ctx fiber.Ctx) (domain.LogoutCredential, error) {
	if err := p.ValidateOrigin(ctx, false); err != nil {
		return domain.LogoutCredential{}, err
	}
	values := ctx.Request().Header.PeekAll(fiber.HeaderAuthorization)
	if len(values) > 0 {
		if len(values) != 1 || !domain.ValidBearer(string(values[0])) {
			return domain.LogoutCredential{}, common.ErrUnauthorized
		}
		return domain.LogoutCredential{Authorization: string(values[0])}, nil
	}
	refresh, err := p.RefreshCredential(ctx)
	if err != nil {
		return domain.LogoutCredential{}, err
	}
	return domain.LogoutCredential{RefreshToken: refresh}, nil
}

func (p *BrowserPolicy) cookie(name, value string) *fiber.Cookie {
	return &fiber.Cookie{Name: name, Value: value, Path: p.path, Secure: true, HTTPOnly: name == RefreshCookie, SameSite: fiber.CookieSameSiteLaxMode}
}

func (p *BrowserPolicy) Issue(ctx fiber.Ctx, result domain.SessionResult, nonce []byte) (domain.LoginResponse, error) {
	now := p.now()
	expiry := result.SessionExpiresAt.UTC().Truncate(time.Second)
	age := int(expiry.Sub(now) / time.Second)
	if result.Token == "" || result.RefreshToken == "" || result.SessionID == "" || len(nonce) != 32 {
		return domain.LoginResponse{}, dictionary.ErrSessionResultInvalid
	}
	if !result.AccessExpiresAt.After(now) || !result.SessionExpiresAt.After(now) || result.AccessExpiresAt.After(result.SessionExpiresAt) {
		return domain.LoginResponse{}, dictionary.ErrSessionResultInvalid
	}
	if result.AccessExpiresAt.Year() > 9999 || result.SessionExpiresAt.Year() > 9999 || age < 1 {
		return domain.LoginResponse{}, dictionary.ErrSessionResultInvalid
	}
	// Reject values that cannot be transported faithfully as a cookie.
	for _, b := range []byte(result.RefreshToken) {
		if b < 0x21 || b > 0x7e || strings.ContainsRune("\";,\\", rune(b)) {
			return domain.LoginResponse{}, dictionary.ErrSessionResultInvalid
		}
	}
	csrf := p.csrfToken(result.RefreshToken, nonce)
	refreshCookie, csrfCookie := p.cookie(RefreshCookie, result.RefreshToken), p.cookie(CSRFCookie, csrf)
	for _, cookie := range []*fiber.Cookie{refreshCookie, csrfCookie} {
		cookie.Expires = expiry
		cookie.MaxAge = age
	}
	data := domain.LoginResponse{
		Token:              result.Token,
		TokenType:          "Bearer",
		SessionID:          result.SessionID,
		AccessExpiresAt:    result.AccessExpiresAt.UTC().Format(time.RFC3339Nano),
		SessionExpiresAt:   result.SessionExpiresAt.UTC().Format(time.RFC3339Nano),
		MustChangePassword: result.MustChangePassword,
	}
	ctx.Cookie(refreshCookie)
	ctx.Cookie(csrfCookie)
	return data, nil
}

func (p *BrowserPolicy) Bootstrap(ctx fiber.Ctx) (domain.CSRFResponse, error) {
	if err := p.ValidateOrigin(ctx, true); err != nil {
		return domain.CSRFResponse{}, err
	}
	refresh, ok := uniqueCookie(ctx, RefreshCookie)
	if !ok {
		return domain.CSRFResponse{}, common.ErrUnauthorized
	}
	nonce, err := p.PrepareNonce()
	if err != nil {
		return domain.CSRFResponse{}, err
	}
	token := p.csrfToken(refresh, nonce)
	cookie := p.cookie(CSRFCookie, token)
	cookie.SessionOnly = true
	ctx.Cookie(cookie)
	return domain.CSRFResponse{CSRFToken: token}, nil
}

func (p *BrowserPolicy) Clear(ctx fiber.Ctx) {
	for _, name := range []string{RefreshCookie, CSRFCookie} {
		cookie := p.cookie(name, "")
		cookie.Expires = time.Unix(1, 0).UTC()
		cookie.MaxAge = -1
		ctx.Cookie(cookie)
	}
}

// LifecycleNoStore runs before CORS/auth so early failures are not cacheable.
func LifecycleNoStore(apiPrefix string) fiber.Handler {
	paths := map[string]bool{apiPrefix + "/users/create": true, apiPrefix + "/users/login": true, apiPrefix + "/auth/refresh": true, apiPrefix + "/auth/logout": true, apiPrefix + "/auth/csrf": true, apiPrefix + "/me/password": true}
	return func(ctx fiber.Ctx) error {
		path := strings.TrimRight(strings.ToLower(ctx.Path()), "/")
		sessions := strings.ToLower(apiPrefix) + "/me/sessions"
		if paths[path] || path == sessions || strings.HasPrefix(path, sessions+"/") {
			ctx.Set(fiber.HeaderCacheControl, "no-store")
		}
		return ctx.Next()
	}
}
