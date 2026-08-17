package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type App struct {
	db     *sql.DB
	router chi.Router
	config Config
}

type Config struct {
	DBHost       string
	DBPort       string
	DBUser       string
	DBPass       string
	DBName       string
	JWTSecret    string
	Port         string
	Environment  string
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

	app := &App{
		db:     db,
		config: config,
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
