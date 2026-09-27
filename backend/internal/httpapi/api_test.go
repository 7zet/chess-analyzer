package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ahror/royal-move/internal/config"
	"github.com/ahror/royal-move/internal/jobs"
	"github.com/ahror/royal-move/internal/store"
)

func testAPI(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Load()
	cfg.DatabaseURL = "file:" + t.Name() + "?mode=memory&cache=shared"
	cfg.JWTSecret = "test-secret-with-more-than-thirty-two-characters"
	cfg.AdminEmail = "admin@example.com"
	db, err := store.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.DB.Close() })
	return New(cfg, db, jobs.New(cfg, db))
}

func performJSON(t *testing.T, handler http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestRegisterLoginAndSession(t *testing.T) {
	handler := testAPI(t)
	credentials := map[string]string{"email": "player@example.com", "password": "securepass123", "displayName": "Royal Player"}
	registered := performJSON(t, handler, http.MethodPost, "/api/auth/register", credentials, "")
	if registered.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", registered.Code, registered.Body.String())
	}
	var session struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(registered.Body.Bytes(), &session); err != nil || session.Token == "" {
		t.Fatalf("missing token: %v", err)
	}
	me := performJSON(t, handler, http.MethodGet, "/api/me", nil, session.Token)
	if me.Code != http.StatusOK {
		t.Fatalf("me status=%d body=%s", me.Code, me.Body.String())
	}
	if got := me.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("security header=%q", got)
	}
	login := performJSON(t, handler, http.MethodPost, "/api/auth/login", map[string]string{"email": "player@example.com", "password": "securepass123"}, "")
	if login.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", login.Code, login.Body.String())
	}
}

func TestRegistrationValidationAndAdminBootstrap(t *testing.T) {
	handler := testAPI(t)
	invalid := performJSON(t, handler, http.MethodPost, "/api/auth/register", map[string]string{"email": "invalid", "password": "short", "displayName": "A"}, "")
	if invalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid status=%d", invalid.Code)
	}
	admin := performJSON(t, handler, http.MethodPost, "/api/auth/register", map[string]string{"email": "admin@example.com", "password": "securepass123", "displayName": "Administrator"}, "")
	if admin.Code != http.StatusCreated {
		t.Fatalf("admin status=%d body=%s", admin.Code, admin.Body.String())
	}
	if !bytes.Contains(admin.Body.Bytes(), []byte(`"role":"admin"`)) {
		t.Fatalf("admin role missing: %s", admin.Body.String())
	}
	var session struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(admin.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	users := performJSON(t, handler, http.MethodGet, "/api/admin/users", nil, session.Token)
	if users.Code != http.StatusOK {
		t.Fatalf("admin users status=%d body=%s", users.Code, users.Body.String())
	}
}
