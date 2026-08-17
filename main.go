package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/microcosm-cc/bluemonday"
)

type App struct {
	db            *sql.DB
	router        chi.Router
	config        Config
	settingsCache map[string]string
	settingsMu    sync.RWMutex
	templateMu    sync.Mutex
	templateCache map[string]*template.Template
	sanitizer     *bluemonday.Policy
}

type Config struct {
	DBHost      string
	DBPort      string
	DBUser      string
	DBPass      string
	DBName      string
	JWTSecret   string
	Port        string
	Environment string
}

func init() {
	// Ensure uploads directory exists
	if err := os.MkdirAll("./static/uploads", 0755); err != nil {
		log.Printf("Warning: could not create uploads directory: %v", err)
	}
}

func main() {
	config := loadConfig()

	db, err := initDatabase(config)
	if err != nil {
		log.Fatalf("Database init failed: %v", err)
	}
	defer db.Close()

	// Create sanitizer policy
	policy := bluemonday.UGCPolicy()
	// Allow tables, but we can be more permissive
	policy.AllowElements("table", "thead", "tbody", "tr", "th", "td")
	policy.AllowElements("figure", "figcaption")
	policy.AllowElements("hr")
	policy.AllowElements("pre", "code")
	policy.AllowAttrs("class").OnElements("code", "pre")

	app := &App{
		db:            db,
		config:        config,
		settingsCache: make(map[string]string),
		templateCache: make(map[string]*template.Template),
		sanitizer:     policy,
	}

	app.router = setupRoutes(app)

	server := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      app.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting server on %s (env: %s)", server.Addr, config.Environment)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func loadConfig() Config {
	return Config{
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "3306"),
		DBUser:      getEnv("DB_USER", "cms"),
		DBPass:      getEnv("DB_PASS", ""),
		DBName:      getEnv("DB_NAME", "cms"),
		JWTSecret:   getEnv("JWT_SECRET", "change-me-in-production"),
		Port:        getEnv("PORT", "3000"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return defaultValue
}

func initDatabase(config Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		config.DBUser,
		config.DBPass,
		config.DBHost,
		config.DBPort,
		config.DBName,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := runMigrations(db); err != nil {
		return nil, err
	}

	return db, nil
}

func setupRoutes(app *App) chi.Router {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(corsMiddleware)

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Public routes
	r.Get("/", app.homePage)
	r.Get("/posts", app.listPosts)
	r.Get("/posts/{slug}", app.viewPost)
	r.Get("/category/{slug}", app.categoryPosts)
	r.Get("/tags/{slug}", app.tagPosts)
	r.Get("/search", app.searchPosts)
	r.Get("/subscribe", app.subscribeForm)
	r.Post("/subscribe", app.subscribeHandler)
	r.Get("/sitemap.xml", app.sitemap)
	r.Get("/robots.txt", app.robots)

	// Comments
	r.Post("/api/comments", app.createComment)
	r.Get("/api/comments/{postID}", app.getComments)

	// Auth routes
	r.Post("/api/auth/register", app.registerUser)
	r.Post("/api/auth/login", app.loginUser)

	// Admin routes (protected)
	r.Route("/admin", func(r chi.Router) {
		r.Use(app.authMiddleware)

		// Dashboard
		r.Get("/", app.adminDashboard)

		// Posts
		r.Get("/posts", app.adminPostsList)
		r.Get("/posts/new", app.newPostForm)
		r.Post("/posts", app.createPost)
		r.Get("/posts/{id}/edit", app.editPostForm)
		r.Post("/posts/{id}", app.updatePost)
		r.Post("/posts/{id}/delete", app.deletePost)
		r.Post("/posts/{id}/publish", app.togglePublish)

		// Categories
		r.Get("/categories", app.adminCategories)
		r.Post("/categories", app.createCategory)
		r.Post("/categories/{id}/delete", app.deleteCategory)

		// Tags
		r.Get("/tags", app.adminTags)
		r.Post("/tags", app.createTag)
		r.Post("/tags/{id}/delete", app.deleteTag)

		// Comments moderation
		r.Get("/comments", app.adminComments)
		r.Post("/comments/{id}/approve", app.approveComment)
		r.Post("/comments/{id}/delete", app.deleteCommentAdmin)

		// Subscribers
		r.Get("/subscribers", app.adminSubscribers)
		r.Get("/subscribers/export", app.exportSubscribers)
		r.Post("/subscribers/{id}/delete", app.deleteSubscriber)

		// Settings
		r.Get("/settings", app.adminSettings)
		r.Post("/settings", app.updateSettings)

		// Media upload
		r.Post("/upload", app.uploadMedia)
	})

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		next.ServeHTTP(w, r)
	})
}

// getSiteSettings retrieves all settings from the database, caching them in memory.
func (app *App) getSiteSettings() map[string]string {
	app.settingsMu.RLock()
	if len(app.settingsCache) > 0 {
		defer app.settingsMu.RUnlock()
		return app.settingsCache
	}
	app.settingsMu.RUnlock()

	app.settingsMu.Lock()
	defer app.settingsMu.Unlock()

	// Double-check after acquiring write lock
	if len(app.settingsCache) > 0 {
		return app.settingsCache
	}

	rows, err := app.db.Query("SELECT key, value FROM settings")
	if err != nil {
		log.Printf("Failed to load settings: %v", err)
		return map[string]string{}
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		settings[key] = value
	}
	app.settingsCache = settings
	return settings
}

func (app *App) renderTemplate(w http.ResponseWriter, name string, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}

	// Merge site settings
	settings := app.getSiteSettings()
	for k, v := range settings {
		if _, exists := data[k]; !exists {
			data[k] = v
		}
	}
	// Provide default site title if not set
	if _, ok := data["siteTitle"]; !ok {
		data["siteTitle"] = "CMS"
	}
	if _, ok := data["siteDescription"]; !ok {
		data["siteDescription"] = "A minimalist markdown publishing platform"
	}

	// Add template functions
	funcMap := template.FuncMap{
		"date": func(t interface{}, layout string) string {
			var tm time.Time
			switch v := t.(type) {
			case time.Time:
				tm = v
			case *time.Time:
				if v == nil {
					return ""
				}
				tm = *v
			default:
				return ""
			}
			if tm.IsZero() {
				return ""
			}
			return tm.Format(layout)
		},
		"truncate": func(s string, n int) string {
			runes := []rune(s)
			if len(runes) <= n {
				return s
			}
			return string(runes[:n]) + "..."
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"default": func(def, val interface{}) interface{} {
			if val == nil || val == "" {
				return def
			}
			return val
		},
		"now": func() time.Time {
			return time.Now()
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	}

	templatePath := "./templates/" + name + ".html"

	app.templateMu.Lock()
	tmpl, ok := app.templateCache[name]
	if !ok {
		var err error
		tmpl, err = template.New(name).Funcs(funcMap).ParseFiles(templatePath)
		if err != nil {
			app.templateMu.Unlock()
			log.Printf("Template error: %v", err)
			app.renderError(w, 500, "Template error")
			return
		}
		app.templateCache[name] = tmpl
	}
	app.templateMu.Unlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Render error: %v", err)
	}
}

func (app *App) renderError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Error</title>
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; padding: 2rem; background: #f5f5f5; }
		.container { max-width: 600px; margin: 0 auto; background: white; padding: 2rem; border-radius: 8px; }
		h1 { margin-top: 0; color: #333; }
		p { color: #666; }
	</style>
</head>
<body>
	<div class="container">
		<h1>Error</h1>
		<p>` + message + `</p>
	</div>
</body>
</html>`))
}

func (app *App) jsonResponse(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func (app *App) jsonError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
