package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/ahror/royal-move/internal/auth"
	chessparser "github.com/ahror/royal-move/internal/chess"
	"github.com/ahror/royal-move/internal/config"
	"github.com/ahror/royal-move/internal/domain"
	"github.com/ahror/royal-move/internal/jobs"
	"github.com/ahror/royal-move/internal/platform"
	"github.com/ahror/royal-move/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/crypto/bcrypt"
)

type API struct {
	cfg       config.Config
	store     *store.Store
	queue     *jobs.Runner
	platforms *platform.Client
}

func New(cfg config.Config, s *store.Store, q *jobs.Runner) http.Handler {
	a := &API{cfg: cfg, store: s, queue: q, platforms: platform.New()}
	r := chi.NewRouter()
	limiter := newRateLimiter(180, time.Minute)
	authLimiter := newRateLimiter(12, 5*time.Minute)
	r.Use(middleware.RequestID, middleware.RealIP, requestLogger, middleware.Recoverer, middleware.Compress(5), middleware.Timeout(35*time.Second), a.securityHeaders, a.cors, limiter.middleware)
	r.Get("/health", a.health)
	r.Route("/api", func(r chi.Router) {
		r.With(authLimiter.middleware).Post("/auth/register", a.register)
		r.With(authLimiter.middleware).Post("/auth/login", a.login)
		r.Get("/platforms/{source}/{username}/games", a.platformGames)
		r.Group(func(r chi.Router) {
			r.Use(func(next http.Handler) http.Handler { return auth.Middleware(cfg.JWTSecret, next) }, a.requireActiveUser)
			r.Post("/analyses", a.createAnalysis)
			r.Get("/analyses", a.listAnalyses)
			r.Get("/analyses/{id}", a.getAnalysis)
			r.Post("/analyses/{id}/retry", a.retry)
			r.Get("/me", a.me)
			r.Route("/admin", func(r chi.Router) {
				r.Use(a.requireAdmin)
				r.Get("/users", a.listUsers)
				r.Patch("/users/{id}/status", a.updateUserStatus)
			})
		})
	})
	return r
}

func (a *API) requireActiveUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := a.store.User(r.Context(), auth.FromContext(r.Context()).UserID)
		if err != nil {
			problem(w, http.StatusUnauthorized, "Session is no longer valid")
			return
		}
		if user.Status != "active" {
			problem(w, http.StatusForbidden, "This account has been blocked")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *API) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := a.store.User(r.Context(), auth.FromContext(r.Context()).UserID)
		if err != nil || user.Role != "admin" {
			problem(w, http.StatusForbidden, "Administrator access is required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *API) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.store.ListUsers(r.Context())
	if err != nil {
		problem(w, http.StatusInternalServerError, "Users could not be loaded")
		return
	}
	write(w, http.StatusOK, map[string]any{"users": users})
}

func (a *API) updateUserStatus(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Status string `json:"status"`
	}
	if decode(w, r, &input) != nil {
		return
	}
	claims := auth.FromContext(r.Context())
	id := chi.URLParam(r, "id")
	if id == claims.UserID && input.Status == "blocked" {
		problem(w, http.StatusConflict, "You cannot block your own administrator account")
		return
	}
	if err := a.store.UpdateUserStatus(r.Context(), id, input.Status); err != nil {
		if store.IsNotFound(err) {
			problem(w, http.StatusNotFound, "User not found")
			return
		}
		problem(w, http.StatusUnprocessableEntity, "User status must be active or blocked")
		return
	}
	write(w, http.StatusOK, map[string]string{"status": input.Status})
}

func (a *API) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Origin", a.cfg.WebOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Expose-Headers", "X-RateLimit-Limit, X-RateLimit-Remaining, X-Request-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (a *API) health(w http.ResponseWriter, r *http.Request) {
	if err := a.store.DB.PingContext(r.Context()); err != nil {
		problem(w, 503, "database unavailable")
		return
	}
	write(w, 200, map[string]any{"status": "healthy", "time": time.Now()})
}

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var input struct{ Email, Password, DisplayName string }
	if decode(w, r, &input) != nil {
		return
	}
	input.Email = strings.TrimSpace(input.Email)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	address, emailErr := mail.ParseAddress(input.Email)
	if emailErr != nil || address.Address != input.Email || len(input.Password) < 8 || len(input.Password) > 72 || len([]rune(input.DisplayName)) < 2 || len([]rune(input.DisplayName)) > 80 {
		problem(w, 422, "Enter your name, a valid email, and a password of at least 8 characters")
		return
	}
	role := "user"
	if a.cfg.AdminEmail != "" && strings.EqualFold(input.Email, a.cfg.AdminEmail) {
		role = "admin"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		problem(w, 500, "The account could not be created")
		return
	}
	user, err := a.store.CreateUser(r.Context(), input.Email, input.DisplayName, string(hash), role)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			problem(w, 409, "An account with this email already exists")
			return
		}
		problem(w, 500, "The account could not be created")
		return
	}
	token, _ := auth.Token(a.cfg.JWTSecret, user)
	write(w, 201, map[string]any{"token": token, "user": user})
}
func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var input struct{ Email, Password string }
	if decode(w, r, &input) != nil {
		return
	}
	user, hash, err := a.store.UserCredentials(r.Context(), input.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(input.Password)) != nil {
		problem(w, 401, "Incorrect email or password")
		return
	}
	if user.Status != "active" {
		problem(w, 403, "This account has been blocked")
		return
	}
	token, _ := auth.Token(a.cfg.JWTSecret, user)
	write(w, 200, map[string]any{"token": token, "user": user})
}
func (a *API) me(w http.ResponseWriter, r *http.Request) {
	user, err := a.store.User(r.Context(), auth.FromContext(r.Context()).UserID)
	if err != nil {
		problem(w, http.StatusUnauthorized, "Session is no longer valid")
		return
	}
	write(w, 200, user)
}
func (a *API) platformGames(w http.ResponseWriter, r *http.Request) {
	items, err := a.platforms.Games(r.Context(), chi.URLParam(r, "source"), chi.URLParam(r, "username"))
	if err != nil {
		problem(w, 400, err.Error())
		return
	}
	write(w, 200, map[string]any{"games": items})
}

func (a *API) createAnalysis(w http.ResponseWriter, r *http.Request) {
	var input struct{ Source, ExternalID, PGN string }
	if decode(w, r, &input) != nil {
		return
	}
	if input.Source != "pgn" && input.Source != "chesscom" && input.Source != "lichess" {
		problem(w, 422, "Invalid analysis source")
		return
	}
	parsed, err := chessparser.ParsePGN(input.PGN)
	if err != nil {
		problem(w, 422, err.Error())
		return
	}
	claims := auth.FromContext(r.Context())
	uid := claims.UserID
	var external *string
	if input.ExternalID != "" {
		external = &input.ExternalID
	}
	item, err := a.store.CreateAnalysis(r.Context(), domain.Analysis{UserID: &uid, Source: input.Source, ExternalID: external, WhiteName: parsed.White, BlackName: parsed.Black, Result: parsed.Result, PGN: parsed.PGN})
	if err != nil {
		problem(w, 500, "The analysis could not be created")
		return
	}
	if err = a.queue.Enqueue(item.ID); err != nil {
		a.store.FailAnalysis(r.Context(), item.ID, err)
		problem(w, 503, "The analysis queue is unavailable")
		return
	}
	write(w, 201, item)
}
func (a *API) getAnalysis(w http.ResponseWriter, r *http.Request) {
	item, err := a.store.Analysis(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if store.IsNotFound(err) {
			problem(w, 404, "Analysis not found")
			return
		}
		problem(w, 500, "The analysis could not be loaded")
		return
	}
	claims := auth.FromContext(r.Context())
	if claims.Role != "admin" && (item.UserID == nil || *item.UserID != claims.UserID) {
		problem(w, 403, "You do not have permission to access this analysis")
		return
	}
	write(w, 200, item)
}
func (a *API) listAnalyses(w http.ResponseWriter, r *http.Request) {
	claims := auth.FromContext(r.Context())
	items, err := a.store.ListAnalyses(r.Context(), claims.UserID, claims.Role == "admin")
	if err != nil {
		problem(w, 500, "Analysis history could not be loaded")
		return
	}
	write(w, 200, map[string]any{"analyses": items})
}
func (a *API) retry(w http.ResponseWriter, r *http.Request) {
	item, err := a.store.Analysis(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		problem(w, 404, "Analysis not found")
		return
	}
	claims := auth.FromContext(r.Context())
	if claims.Role != "admin" && (item.UserID == nil || *item.UserID != claims.UserID) {
		problem(w, 403, "You do not have permission to access this analysis")
		return
	}
	if item.Status != "failed" {
		problem(w, 409, "Only failed analyses can be retried")
		return
	}
	_, err = a.store.DB.ExecContext(r.Context(), `UPDATE analyses SET status='queued',progress=0,error=NULL WHERE id=?`, item.ID)
	if err == nil {
		err = a.queue.Enqueue(item.ID)
	}
	if err != nil {
		problem(w, 500, "The analysis could not be restarted")
		return
	}
	write(w, 202, map[string]string{"status": "queued"})
}

func decode(w http.ResponseWriter, r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		problem(w, 400, "Invalid request format")
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		problem(w, 400, "The request body must contain a single JSON object")
		return err
	}
	return nil
}
func write(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func problem(w http.ResponseWriter, status int, message string) {
	write(w, status, map[string]any{"error": message, "status": status})
}
