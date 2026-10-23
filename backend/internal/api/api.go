package api

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/a-chris/trackete/backend/internal/auth"
	"github.com/a-chris/trackete/backend/internal/config"
	"github.com/a-chris/trackete/backend/internal/store"
)

const (
	sessionCookie = "trackete_session"
	ctxUserKey    = "trackete.userID"
	maxBodyBytes  = 1 << 20 // 1 MiB
)

type API struct {
	cfg     config.Config
	store   *store.Store
	limiter *attemptLimiter
}

func NewRouter(cfg config.Config, st *store.Store) *gin.Engine {
	a := &API{
		cfg:     cfg,
		store:   st,
		limiter: newAttemptLimiter(10, 5*time.Minute),
	}

	r := gin.New()
	// Trust no proxy by default: otherwise any client can spoof
	// X-Forwarded-For and slip past the per-IP auth rate limiter. Behind a
	// reverse proxy the operator lists its addresses in TRUSTED_PROXIES.
	if len(cfg.TrustedProxies) == 0 {
		if err := r.SetTrustedProxies(nil); err != nil {
			log.Printf("WARN: could not disable trusted proxies: %v", err)
		}
	} else if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Printf("WARN: invalid TRUSTED_PROXIES %v: %v (trusting no proxy)", cfg.TrustedProxies, err)
		if err := r.SetTrustedProxies(nil); err != nil {
			log.Printf("WARN: could not disable trusted proxies: %v", err)
		}
	}
	r.RedirectTrailingSlash = false
	r.Use(gin.Logger(), gin.Recovery(), securityHeaders(), cors(cfg.CORSOrigins), limitBody())

	r.GET("/api/health", a.health)
	r.GET("/api/meta", a.meta)

	apiGroup := r.Group("/api")

	apiGroup.POST("/auth/register", a.limiter.guard(a.register))
	apiGroup.POST("/auth/login", a.limiter.guard(a.login))
	apiGroup.POST("/auth/logout", a.logout)

	protected := apiGroup.Group("", a.requireAuth())
	{
		protected.GET("/auth/me", a.me)
		protected.PATCH("/auth/me", a.updateMe)
		protected.POST("/auth/password", a.changePassword)

		protected.GET("/overview", a.overview)

		protected.GET("/trackers", a.listTrackers)
		protected.POST("/trackers", a.createTracker)
		protected.GET("/trackers/:id", a.getTracker)
		protected.PUT("/trackers/:id", a.updateTracker)
		protected.DELETE("/trackers/:id", a.deleteTracker)
		protected.GET("/trackers/:id/data", a.trackerData)
		protected.GET("/trackers/:id/entries", a.listEntries)
		protected.PUT("/trackers/:id/entries/:day", a.upsertEntry)
		protected.DELETE("/trackers/:id/entries/:day", a.deleteEntry)

		protected.GET("/dashboards", a.listDashboards)
		protected.POST("/dashboards", a.createDashboard)
		protected.GET("/dashboards/:id", a.getDashboard)
		protected.PATCH("/dashboards/:id", a.renameDashboard)
		protected.DELETE("/dashboards/:id", a.deleteDashboard)
		protected.GET("/dashboards/:id/data", a.dashboardData)
		protected.PUT("/dashboards/:id/layout", a.updateLayouts)
		protected.POST("/dashboards/:id/widgets", a.createWidget)

		protected.PUT("/widgets/:id", a.updateWidget)
		protected.DELETE("/widgets/:id", a.deleteWidget)
	}

	a.registerStatic(r)
	return r
}

func (a *API) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "version": a.cfg.Version, "time": time.Now().UTC()})
}

func (a *API) meta(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":           a.cfg.Version,
		"allowRegistration": a.cfg.AllowRegistration,
	})
}

func userID(c *gin.Context) string {
	if v, ok := c.Get(ctxUserKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ---- responses -------------------------------------------------------------

func fail(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}

func failStore(c *gin.Context, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		fail(c, http.StatusNotFound, "not found")
	case errors.Is(err, store.ErrEmailTaken):
		fail(c, http.StatusConflict, "that email is already registered")
	default:
		log.Printf("store error: %v", err)
		fail(c, http.StatusInternalServerError, "something went wrong")
	}
}

// ---- middleware ------------------------------------------------------------

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy",
			"default-src 'self'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; "+
				"script-src 'self'; font-src 'self' data:; connect-src 'self'; object-src 'none'; "+
				"base-uri 'self'; form-action 'self'; frame-ancestors 'none'")
		c.Next()
	}
}

func limitBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
		}
		c.Next()
	}
}

func cors(origins []string) gin.HandlerFunc {
	if len(origins) == 0 {
		return func(c *gin.Context) { c.Next() }
	}
	allowed := make(map[string]bool, len(origins))
	wildcard := false
	for _, o := range origins {
		if o == "*" {
			wildcard = true
		}
		allowed[o] = true
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && (wildcard || allowed[origin]) {
			h := c.Writer.Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Add("Vary", "Origin")
			h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Max-Age", "600")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (a *API) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ""
		if cookie, err := c.Cookie(sessionCookie); err == nil {
			token = cookie
		}
		if token == "" {
			if header := c.GetHeader("Authorization"); strings.HasPrefix(header, "Bearer ") {
				token = strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
			}
		}
		if token == "" {
			fail(c, http.StatusUnauthorized, "authentication required")
			return
		}

		uid, ver, err := auth.ParseToken(a.cfg.JWTSecret, token)
		if err != nil {
			fail(c, http.StatusUnauthorized, "your session has expired, please sign in again")
			return
		}
		user, err := a.store.UserByID(c.Request.Context(), uid)
		if err != nil {
			a.clearSession(c)
			fail(c, http.StatusUnauthorized, "your session has expired, please sign in again")
			return
		}
		// Changing the password bumps token_version, which invalidates every
		// session issued before the change.
		if user.TokenVersion != ver {
			a.clearSession(c)
			fail(c, http.StatusUnauthorized, "your session has expired, please sign in again")
			return
		}

		c.Set(ctxUserKey, uid)
		c.Next()
	}
}

// ---- static frontend -------------------------------------------------------

func (a *API) registerStatic(r *gin.Engine) {
	dir := a.cfg.StaticDir
	index := filepath.Join(dir, "index.html")
	if dir == "" {
		r.NoRoute(func(c *gin.Context) { fail(c, http.StatusNotFound, "not found") })
		return
	}
	if _, err := os.Stat(index); err != nil {
		log.Printf("WARN: frontend build not found at %s (API-only mode)", index)
		r.NoRoute(func(c *gin.Context) { fail(c, http.StatusNotFound, "not found") })
		return
	}

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/api" || strings.HasPrefix(path, "/api/") {
			fail(c, http.StatusNotFound, "not found")
			return
		}
		clean := filepath.Clean("/" + strings.TrimPrefix(path, "/"))
		full := filepath.Join(dir, strings.TrimPrefix(clean, "/"))
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			c.File(full)
			return
		}
		c.File(index)
	})
}
