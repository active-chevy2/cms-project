package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/yuin/goldmark"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================================
// PUBLIC HANDLERS
// ============================================================================

func (app *App) homePage(w http.ResponseWriter, r *http.Request) {
	posts, err := app.getPublishedPosts(0, 10)
	if err != nil {
		app.renderError(w, 500, "Failed to load posts")
		return
	}

	app.renderTemplate(w, "home", map[string]interface{}{
		"posts": posts,
	})
}

func (app *App) listPosts(w http.ResponseWriter, r *http.Request) {
	page := parsePage(r.URL.Query().Get("page"))
	postsPerPage := 10

	posts, err := app.getPublishedPosts((page-1)*postsPerPage, postsPerPage)
	if err != nil {
		app.renderError(w, 500, "Failed to load posts")
		return
	}

	app.renderTemplate(w, "posts-list", map[string]interface{}{
		"posts": posts,
		"page":  page,
	})
}

func (app *App) viewPost(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var post Post
	var authorName string
	var categoryName string

	query := `
		SELECT p.id, p.title, p.slug, p.content, p.excerpt, p.featured_image,
		       p.author_id, u.username, p.category_id, c.name,
		       p.is_published, p.published_at, p.created_at, p.updated_at
		FROM posts p
		LEFT JOIN users u ON p.author_id = u.id
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE p.slug = ? AND p.is_published = true
	`

	err := app.db.QueryRow(query, slug).Scan(
		&post.ID, &post.Title, &post.Slug, &post.Content, &post.Excerpt,
		&post.FeaturedImage, &post.AuthorID, &authorName, &post.CategoryID,
		&categoryName, &post.IsPublished, &post.PublishedAt, &post.CreatedAt,
		&post.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		app.renderError(w, 404, "Post not found")
		return
	}
	if err != nil {
		app.renderError(w, 500, "Failed to load post")
		return
	}

	// Get comments
	comments, _ := app.getApprovedComments(post.ID)

	// Get tags
	tags, _ := app.getPostTags(post.ID)
	post.Tags = tags

	// Parse markdown to HTML
	html := app.markdownToHTML(post.Content)

	app.renderTemplate(w, "post", map[string]interface{}{
		"post":        post,
		"content":     template.HTML(html),
		"author":      authorName,
		"category":    categoryName,
		"comments":    comments,
		"url":         r.URL.String(),
	})
}

func (app *App) categoryPosts(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var category Category
	err := app.db.QueryRow(
		"SELECT id, name, slug, description FROM categories WHERE slug = ?",
		slug,
	).Scan(&category.ID, &category.Name, &category.Slug, &category.Description)

	if err == sql.ErrNoRows {
		app.renderError(w, 404, "Category not found")
		return
	}

	query := `
		SELECT p.id, p.title, p.slug, p.excerpt, p.published_at, p.created_at
		FROM posts p
		WHERE p.category_id = ? AND p.is_published = true
		ORDER BY p.published_at DESC
	`

	rows, err := app.db.Query(query, category.ID)
	if err != nil {
		app.renderError(w, 500, "Failed to load posts")
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.Title, &post.Slug, &post.Excerpt,
			&post.PublishedAt, &post.CreatedAt); err != nil {
			continue
		}
		posts = append(posts, post)
	}

	app.renderTemplate(w, "category", map[string]interface{}{
		"category": category,
		"posts":    posts,
	})
}

func (app *App) tagPosts(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var tag Tag
	err := app.db.QueryRow(
		"SELECT id, name, slug FROM tags WHERE slug = ?",
		slug,
	).Scan(&tag.ID, &tag.Name, &tag.Slug)

	if err == sql.ErrNoRows {
		app.renderError(w, 404, "Tag not found")
		return
	}

	query := `
		SELECT p.id, p.title, p.slug, p.excerpt, p.published_at, p.created_at
		FROM posts p
		JOIN post_tags pt ON p.id = pt.post_id
		WHERE pt.tag_id = ? AND p.is_published = true
		ORDER BY p.published_at DESC
	`

	rows, err := app.db.Query(query, tag.ID)
	if err != nil {
		app.renderError(w, 500, "Failed to load posts")
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.Title, &post.Slug, &post.Excerpt,
			&post.PublishedAt, &post.CreatedAt); err != nil {
			continue
		}
		posts = append(posts, post)
	}

	app.renderTemplate(w, "tag", map[string]interface{}{
		"tag":   tag,
		"posts": posts,
	})
}

func (app *App) searchPosts(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) < 2 {
		app.renderError(w, 400, "Search query too short")
		return
	}

	searchQuery := `
		SELECT p.id, p.title, p.slug, p.excerpt, p.published_at, p.created_at
		FROM posts p
		WHERE p.is_published = true AND MATCH(p.title, p.content) AGAINST(? IN BOOLEAN MODE)
		ORDER BY p.published_at DESC
		LIMIT 50
	`

	rows, err := app.db.Query(searchQuery, query)
	if err != nil {
		app.renderError(w, 500, "Search failed")
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.Title, &post.Slug, &post.Excerpt,
			&post.PublishedAt, &post.CreatedAt); err != nil {
			continue
		}
		posts = append(posts, post)
	}

	app.renderTemplate(w, "search", map[string]interface{}{
		"query": query,
		"posts": posts,
	})
}

func (app *App) subscribeForm(w http.ResponseWriter, r *http.Request) {
	app.renderTemplate(w, "subscribe", nil)
}

func (app *App) subscribeHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.renderError(w, 400, "Invalid form data")
		return
	}

	email := strings.TrimSpace(r.PostFormValue("email"))
	if !isValidEmail(email) {
		app.renderError(w, 400, "Invalid email address")
		return
	}

	id := uuid.New().String()
	_, err := app.db.Exec(
		"INSERT INTO subscribers (id, email) VALUES (?, ?) ON DUPLICATE KEY UPDATE is_active = true, unsubscribed_at = NULL",
		id, email,
	)

	if err != nil {
		app.renderError(w, 500, "Subscription failed")
		return
	}

	app.renderTemplate(w, "subscribe-success", map[string]interface{}{
		"email": email,
	})
}

// ============================================================================
// COMMENT HANDLERS
// ============================================================================

func (app *App) createComment(w http.ResponseWriter, r *http.Request) {
	postID := r.URL.Query().Get("post_id")
	if postID == "" {
		app.jsonError(w, 400, "Post ID required")
		return
	}

	var req CommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.jsonError(w, 400, "Invalid request")
		return
	}

	if len(strings.TrimSpace(req.Content)) < 3 {
		app.jsonError(w, 400, "Comment too short")
		return
	}

	if !isValidEmail(req.AuthorEmail) {
		app.jsonError(w, 400, "Invalid email")
		return
	}

	id := uuid.New().String()
	_, err := app.db.Exec(`
		INSERT INTO comments (id, post_id, author_name, author_email, author_url, content, is_approved)
		VALUES (?, ?, ?, ?, ?, ?, false)
	`, id, postID, req.AuthorName, req.AuthorEmail, req.AuthorURL, req.Content)

	if err != nil {
		app.jsonError(w, 500, "Failed to save comment")
		return
	}

	app.jsonResponse(w, 201, map[string]string{
		"message": "Comment submitted and awaiting approval",
		"id":      id,
	})
}

func (app *App) getComments(w http.ResponseWriter, r *http.Request) {
	postID := chi.URLParam(r, "postID")

	comments, err := app.getApprovedComments(postID)
	if err != nil {
		app.jsonError(w, 500, "Failed to load comments")
		return
	}

	app.jsonResponse(w, 200, comments)
}

// ============================================================================
// AUTH HANDLERS
// ============================================================================

func (app *App) registerUser(w http.ResponseWriter, r *http.Request) {
	// Check if first user exists
	var userCount int
	err := app.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err == nil && userCount > 0 {
		app.jsonError(w, 403, "User registration disabled")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.jsonError(w, 400, "Invalid request")
		return
	}

	if !isValidEmail(req.Email) || len(req.Password) < 8 || len(req.Username) < 3 {
		app.jsonError(w, 400, "Invalid input")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		app.jsonError(w, 500, "Failed to process password")
		return
	}

	id := uuid.New().String()
	_, err = app.db.Exec(`
		INSERT INTO users (id, email, username, password_hash, is_admin)
		VALUES (?, ?, ?, ?, true)
	`, id, req.Email, req.Username, string(hash))

	if err != nil {
		app.jsonError(w, 409, "Email or username already exists")
		return
	}

	token, err := app.generateJWT(id)
	if err != nil {
		app.jsonError(w, 500, "Failed to create token")
		return
	}

	app.jsonResponse(w, 201, map[string]interface{}{
		"token":   token,
		"message": "User registered successfully",
	})
}

func (app *App) loginUser(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.jsonError(w, 400, "Invalid request")
		return
	}

	var user User
	err := app.db.QueryRow(
		"SELECT id, email, username, password_hash, is_admin FROM users WHERE email = ?",
		req.Email,
	).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.IsAdmin)

	if err == sql.ErrNoRows {
		app.jsonError(w, 401, "Invalid credentials")
		return
	}
	if err != nil {
		app.jsonError(w, 500, "Database error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		app.jsonError(w, 401, "Invalid credentials")
		return
	}

	token, err := app.generateJWT(user.ID)
	if err != nil {
		app.jsonError(w, 500, "Failed to create token")
		return
	}

	app.jsonResponse(w, 200, AuthResponse{
		Token: token,
		User: &User{
			ID:       user.ID,
			Email:    user.Email,
			Username: user.Username,
			IsAdmin:  user.IsAdmin,
		},
	})
}

// ============================================================================
// SEO HANDLERS
// ============================================================================

func (app *App) sitemap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")

	rows, err := app.db.Query(`
		SELECT slug, updated_at FROM posts WHERE is_published = true
		ORDER BY updated_at DESC
	`)
	if err != nil {
		http.Error(w, "Internal error", 500)
		return
	}
	defer rows.Close()

	siteURL := os.Getenv("SITE_URL")
	if siteURL == "" {
		siteURL = "http://localhost:3000"
	}

	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
`)

	for rows.Next() {
		var slug string
		var updatedAt time.Time
		if err := rows.Scan(&slug, &updatedAt); err != nil {
			continue
		}
		fmt.Fprintf(w, `  <url>
    <loc>%s/posts/%s</loc>
    <lastmod>%s</lastmod>
  </url>
`, siteURL, slug, updatedAt.Format(time.RFC3339))
	}

	fmt.Fprint(w, `</urlset>`)
}

func (app *App) robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	siteURL := os.Getenv("SITE_URL")
	if siteURL == "" {
		siteURL = "http://localhost:3000"
	}

	fmt.Fprintf(w, `User-agent: *
Allow: /

Sitemap: %s/sitemap.xml
`, siteURL)
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

func (app *App) getPublishedPosts(offset, limit int) ([]Post, error) {
	query := `
		SELECT id, title, slug, excerpt, featured_image, published_at, created_at
		FROM posts
		WHERE is_published = true
		ORDER BY published_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := app.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.Title, &post.Slug, &post.Excerpt,
			&post.FeaturedImage, &post.PublishedAt, &post.CreatedAt); err != nil {
			continue
		}
		posts = append(posts, post)
	}

	return posts, nil
}

func (app *App) getApprovedComments(postID string) ([]Comment, error) {
	query := `
		SELECT id, post_id, author_name, author_email, author_url, content, created_at
		FROM comments
		WHERE post_id = ? AND is_approved = true
		ORDER BY created_at DESC
	`

	rows, err := app.db.Query(query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.AuthorName, &c.AuthorEmail,
			&c.AuthorURL, &c.Content, &c.CreatedAt); err != nil {
			continue
		}
		comments = append(comments, c)
	}

	return comments, nil
}

func (app *App) getPostTags(postID string) ([]Tag, error) {
	query := `
		SELECT t.id, t.name, t.slug
		FROM tags t
		JOIN post_tags pt ON t.id = pt.tag_id
		WHERE pt.post_id = ?
		ORDER BY t.name
	`

	rows, err := app.db.Query(query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Slug); err != nil {
			continue
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

func (app *App) generateJWT(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(),
	})

	return token.SignedString([]byte(app.config.JWTSecret))
}

func (app *App) markdownToHTML(md string) string {
	var buf strings.Builder
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		return fmt.Sprintf("<p>%s</p>", md)
	}
	return buf.String()
}

func (app *App) slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, s)
	return strings.Trim(s, "-")
}

func isValidEmail(email string) bool {
	return strings.Contains(email, "@") && strings.Contains(email, ".")
}

func parsePage(p string) int {
	if p == "" {
		return 1
	}
	page := 1
	fmt.Sscanf(p, "%d", &page)
	if page < 1 {
		page = 1
	}
	return page
}
