package api

import (
	"context"
	"net/http"
	"time"

	"container-updater/backend/internal/config"
	"container-updater/backend/internal/logger"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserContextKey contextKey = "user"

type UserClaims struct {
	UserID   string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

func NewRouter() http.Handler {
	r := chi.NewRouter()

	// Base middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(RequestLogger)
	r.Use(middleware.Recoverer)

	// Dynamic CORS config
	r.Use(cors.Handler(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			return true
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Public Routes
	r.Group(func(r chi.Router) {
		r.Get("/api/auth/login", HandleLogin)
		r.Get("/api/auth/callback", HandleCallback)
	})

	// Protected Routes (Require Session Cookie)
	r.Group(func(r chi.Router) {
		r.Use(RequireAuth)

		r.Get("/api/auth/session", HandleSession)
		
		// Workloads
		r.Get("/api/workloads", HandleListWorkloads)
		r.Post("/api/workloads/{id}/update", HandleTriggerUpdate)
		r.Get("/api/audit-logs", HandleListAuditLogs)
		r.Get("/api/stats", HandleGetStats)

		// Notifications
		r.Get("/api/notifications", HandleListNotifications)
		r.Post("/api/notifications", HandleSaveNotification)

		// WebSocket
		r.Get("/api/ws", HandleWebSocket)
	})

	return r
}

// RequestLogger middleware integrates chi request logging with our slog logger
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		logger.Log.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration", time.Since(start).String(),
			"bytes", ww.BytesWritten(),
			"ip", r.RemoteAddr,
		)
	})
}

// RequireAuth middleware verifies the JWT token stored in cookie "session_token"
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			logger.Log.Warn("authentication failed: missing session_token cookie")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tokenString := cookie.Value
		claims := &UserClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.GlobalConfig.OIDC.CookieSecret), nil
		})

		if err != nil || !token.Valid {
			logger.Log.Warn("authentication failed: invalid or expired session_token", "error", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Inject user info into request context
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Handlers are implemented in other files of the api package

// Helper to generate a token for user sessions
func GenerateToken(userID, username, email string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &UserClaims{
		UserID:   userID,
		Username: username,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.GlobalConfig.OIDC.CookieSecret))
}
