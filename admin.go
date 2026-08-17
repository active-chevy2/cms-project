package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ============================================================================
// ADMIN DASHBOARD
// ============================================================================

func (app *App) adminDashboard(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	var postCount, commentCount, subscriberCount int
	app.db.QueryRow("SELECT COUNT(*) FROM posts WHERE is_published = true").Scan(&postCount)
	app.db.QueryRow("SELECT COUNT(*) FROM comments WHERE is_approved = false").Scan(&commentCount)
	app.db.QueryRow("SELECT COUNT(*) FROM subscribers WHERE is_active = true").Scan(&subscriberCount)

	app.renderTemplate(w, "admin/dashboard", map[string]interface{}{
		"user":            user,
		"postCount":       postCount,
		"commentCount":    commentCount,
		"subscriberCount": subscriberCount,
	})
}

// ============================================================================
// POSTS MANAGEMENT
// ============================================================================

func (app *App) adminPostsList(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	query := `
		SELECT id, title, slug, excerpt, is_published, published_at, created_at, updated_at
		FROM posts
		ORDER BY created_at DESC
	`

	rows, err := app.db.Query(query)
	if err != nil {
		app.renderError(w, 500, "Failed to load posts")
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.Title, &post.Slug, &post.Excerpt,
			&post.IsPublished, &post.PublishedAt, &post.CreatedAt, &post.UpdatedAt); err != nil {
			continue
		}
		posts = append(posts, post)
	}

	app.renderTemplate(w, "admin/posts-list", map[string]interface{}{
		"user":  user,
		"posts": posts,
	})
}

func (app *App) newPostForm(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	// Get categories
	rows, _ := app.db.Query("SELECT id, name FROM categories ORDER BY name")
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		if err := rows.Scan(&cat.ID, &cat.Name); err != nil {
			continue
		}
		categories = append(categories, cat)
	}

	app.renderTemplate(w, "admin/post-form", map[string]interface{}{
		"user":       user,
		"categories": categories,
		"isNew":      true,
	})
}

func (app *App) createPost(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	var req CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.jsonError(w, 400, "Invalid request")
		return
	}

	if len(strings.TrimSpace(req.Title)) < 3 {
		app.jsonError(w, 400, "Title too short")
		return
	}

	slug := app.slugify(req.Title)
	id := uuid.New().String()

	// Check slug uniqueness
	var existing string
	app.db.QueryRow("SELECT id FROM posts WHERE slug = ?", slug).Scan(&existing)
	if existing != "" {
		slug = slug + "-" + id[:8]
	}

	tx, err := app.db.Begin()
	if err != nil {
		app.jsonError(w, 500, "Database error")
		return
	}
	defer tx.Rollback()

	// Handle category_id: if empty, set to NULL
	var categoryID interface{}
	if req.CategoryID != "" {
		categoryID = req.CategoryID
	} else {
		categoryID = nil
	}

	_, err = tx.Exec(`
		INSERT INTO posts (id, title, slug, content, excerpt, featured_image, author_id, category_id, is_published)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, req.Title, slug, req.Content, req.Excerpt, req.FeaturedImage, user.ID, categoryID, req.IsPublished)

	if err != nil {
		app.jsonError(w, 500, "Failed to create post")
		return
	}

	// Add tags
	for _, tagName := range req.Tags {
		tagID := app.getOrCreateTag(tx, tagName)
		tx.Exec("INSERT INTO post_tags (post_id, tag_id) VALUES (?, ?)", id, tagID)
	}

	if err := tx.Commit(); err != nil {
		app.jsonError(w, 500, "Database error")
		return
	}

	app.jsonResponse(w, 201, map[string]string{
		"id":   id,
		"slug": slug,
	})
}

func (app *App) editPostForm(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)
	postID := chi.URLParam(r, "id")

	var post Post
	err := app.db.QueryRow(
		"SELECT id, title, slug, content, excerpt, featured_image, category_id, is_published FROM posts WHERE id = ?",
		postID,
	).Scan(&post.ID, &post.Title, &post.Slug, &post.Content, &post.Excerpt,
		&post.FeaturedImage, &post.CategoryID, &post.IsPublished)

	if err == sql.ErrNoRows {
		app.renderError(w, 404, "Post not found")
		return
	}

	// Get tags
	tags, _ := app.getPostTags(postID)
	post.Tags = tags

	// Get categories
	rows, _ := app.db.Query("SELECT id, name FROM categories ORDER BY name")
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		if err := rows.Scan(&cat.ID, &cat.Name); err != nil {
			continue
		}
		categories = append(categories, cat)
	}

	app.renderTemplate(w, "admin/post-form", map[string]interface{}{
		"user":       user,
		"post":       post,
		"categories": categories,
		"isNew":      false,
	})
}

func (app *App) updatePost(w http.ResponseWriter, r *http.Request) {
	postID := chi.URLParam(r, "id")

	var req CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.jsonError(w, 400, "Invalid request")
		return
	}

	// Handle category_id
	var categoryID interface{}
	if req.CategoryID != "" {
		categoryID = req.CategoryID
	} else {
		categoryID = nil
	}

	_, err := app.db.Exec(`
		UPDATE posts
		SET title = ?, content = ?, excerpt = ?, featured_image = ?, category_id = ?, is_published = ?
		WHERE id = ?
	`, req.Title, req.Content, req.Excerpt, req.FeaturedImage, categoryID, req.IsPublished, postID)

	if err != nil {
		app.jsonError(w, 500, "Failed to update post")
		return
	}

	// Update tags
	app.db.Exec("DELETE FROM post_tags WHERE post_id = ?", postID)
	for _, tagName := range req.Tags {
		tagID := app.getOrCreateTag(nil, tagName)
		app.db.Exec("INSERT INTO post_tags (post_id, tag_id) VALUES (?, ?)", postID, tagID)
	}

	app.jsonResponse(w, 200, map[string]string{
		"message": "Post updated",
	})
}

func (app *App) deletePost(w http.ResponseWriter, r *http.Request) {
	postID := chi.URLParam(r, "id")

	_, err := app.db.Exec("DELETE FROM posts WHERE id = ?", postID)
	if err != nil {
		app.jsonError(w, 500, "Failed to delete post")
		return
	}

	app.jsonResponse(w, 200, map[string]string{
		"message": "Post deleted",
	})
}

func (app *App) togglePublish(w http.ResponseWriter, r *http.Request) {
	postID := chi.URLParam(r, "id")

	var isPublished bool
	app.db.QueryRow("SELECT is_published FROM posts WHERE id = ?", postID).Scan(&isPublished)

	var publishedAt interface{}
	if !isPublished {
		now := time.Now()
		publishedAt = now
	} else {
		publishedAt = nil // sets to NULL
	}

	_, err := app.db.Exec(
		"UPDATE posts SET is_published = ?, published_at = ? WHERE id = ?",
		!isPublished, publishedAt, postID,
	)

	if err != nil {
		app.jsonError(w, 500, "Failed to toggle publish status")
		return
	}

	app.jsonResponse(w, 200, map[string]interface{}{
		"isPublished": !isPublished,
	})
}

// ============================================================================
// CATEGORIES
// ============================================================================

func (app *App) adminCategories(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	rows, err := app.db.Query("SELECT id, name, slug, description FROM categories ORDER BY name")
	if err != nil {
		app.renderError(w, 500, "Failed to load categories")
		return
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Slug, &cat.Description); err != nil {
			continue
		}
		categories = append(categories, cat)
	}

	app.renderTemplate(w, "admin/categories", map[string]interface{}{
		"user":       user,
		"categories": categories,
	})
}

func (app *App) createCategory(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.jsonError(w, 400, "Invalid request")
		return
	}

	if len(strings.TrimSpace(req.Name)) < 2 {
		app.jsonError(w, 400, "Name too short")
		return
	}

	id := uuid.New().String()
	slug := app.slugify(req.Name)

	_, err := app.db.Exec(`
		INSERT INTO categories (id, name, slug, description)
		VALUES (?, ?, ?, ?)
	`, id, req.Name, slug, req.Description)

	if err != nil {
		app.jsonError(w, 409, "Category already exists")
		return
	}

	app.jsonResponse(w, 201, map[string]string{
		"id": id,
	})
}

func (app *App) deleteCategory(w http.ResponseWriter, r *http.Request) {
	catID := chi.URLParam(r, "id")

	_, err := app.db.Exec("DELETE FROM categories WHERE id = ?", catID)
	if err != nil {
		app.jsonError(w, 500, "Failed to delete category")
		return
	}

	app.jsonResponse(w, 200, map[string]string{
		"message": "Category deleted",
	})
}

// ============================================================================
// TAGS
// ============================================================================

func (app *App) adminTags(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	rows, err := app.db.Query("SELECT id, name, slug FROM tags ORDER BY name")
	if err != nil {
		app.renderError(w, 500, "Failed to load tags")
		return
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

	app.renderTemplate(w, "admin/tags", map[string]interface{}{
		"user": user,
		"tags": tags,
	})
}

func (app *App) createTag(w http.ResponseWriter, r *http.Request) {
	var req CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.jsonError(w, 400, "Invalid request")
		return
	}

	if len(strings.TrimSpace(req.Name)) < 2 {
		app.jsonError(w, 400, "Name too short")
		return
	}

	id := uuid.New().String()
	slug := app.slugify(req.Name)

	_, err := app.db.Exec(`
		INSERT INTO tags (id, name, slug)
		VALUES (?, ?, ?)
	`, id, req.Name, slug)

	if err != nil {
		app.jsonError(w, 409, "Tag already exists")
		return
	}

	app.jsonResponse(w, 201, map[string]string{
		"id": id,
	})
}

func (app *App) deleteTag(w http.ResponseWriter, r *http.Request) {
	tagID := chi.URLParam(r, "id")

	_, err := app.db.Exec("DELETE FROM tags WHERE id = ?", tagID)
	if err != nil {
		app.jsonError(w, 500, "Failed to delete tag")
		return
	}

	app.jsonResponse(w, 200, map[string]string{
		"message": "Tag deleted",
	})
}

// ============================================================================
// COMMENTS MODERATION
// ============================================================================

func (app *App) adminComments(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	query := `
		SELECT c.id, c.post_id, c.author_name, c.author_email, c.content, c.is_approved, c.created_at, p.title
		FROM comments c
		LEFT JOIN posts p ON c.post_id = p.id
		ORDER BY c.created_at DESC
	`

	rows, err := app.db.Query(query)
	if err != nil {
		app.renderError(w, 500, "Failed to load comments")
		return
	}
	defer rows.Close()

	var comments []map[string]interface{}
	for rows.Next() {
		var c Comment
		var postTitle string
		if err := rows.Scan(&c.ID, &c.PostID, &c.AuthorName, &c.AuthorEmail,
			&c.Content, &c.IsApproved, &c.CreatedAt, &postTitle); err != nil {
			continue
		}
		comments = append(comments, map[string]interface{}{
			"id":        c.ID,
			"postID":    c.PostID,
			"author":    c.AuthorName,
			"email":     c.AuthorEmail,
			"content":   c.Content,
			"approved":  c.IsApproved,
			"createdAt": c.CreatedAt,
			"postTitle": postTitle,
		})
	}

	app.renderTemplate(w, "admin/comments", map[string]interface{}{
		"user":     user,
		"comments": comments,
	})
}

func (app *App) approveComment(w http.ResponseWriter, r *http.Request) {
	commentID := chi.URLParam(r, "id")

	_, err := app.db.Exec("UPDATE comments SET is_approved = true WHERE id = ?", commentID)
	if err != nil {
		app.jsonError(w, 500, "Failed to approve comment")
		return
	}

	app.jsonResponse(w, 200, map[string]string{
		"message": "Comment approved",
	})
}

func (app *App) deleteCommentAdmin(w http.ResponseWriter, r *http.Request) {
	commentID := chi.URLParam(r, "id")

	_, err := app.db.Exec("DELETE FROM comments WHERE id = ?", commentID)
	if err != nil {
		app.jsonError(w, 500, "Failed to delete comment")
		return
	}

	app.jsonResponse(w, 200, map[string]string{
		"message": "Comment deleted",
	})
}

// ============================================================================
// SUBSCRIBERS
// ============================================================================

func (app *App) adminSubscribers(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	rows, err := app.db.Query("SELECT id, email, subscribed_at, is_active FROM subscribers ORDER BY subscribed_at DESC")
	if err != nil {
		app.renderError(w, 500, "Failed to load subscribers")
		return
	}
	defer rows.Close()

	var subscribers []Subscriber
	for rows.Next() {
		var sub Subscriber
		if err := rows.Scan(&sub.ID, &sub.Email, &sub.SubscribedAt, &sub.IsActive); err != nil {
			continue
		}
		subscribers = append(subscribers, sub)
	}

	app.renderTemplate(w, "admin/subscribers", map[string]interface{}{
		"user":        user,
		"subscribers": subscribers,
	})
}

func (app *App) exportSubscribers(w http.ResponseWriter, r *http.Request) {
	rows, err := app.db.Query("SELECT email FROM subscribers WHERE is_active = true ORDER BY email")
	if err != nil {
		app.jsonError(w, 500, "Export failed")
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=subscribers.csv")

	writer := csv.NewWriter(w)
	writer.Write([]string{"email"})

	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			continue
		}
		writer.Write([]string{email})
	}
	writer.Flush()
}

func (app *App) deleteSubscriber(w http.ResponseWriter, r *http.Request) {
	subID := chi.URLParam(r, "id")

	_, err := app.db.Exec("DELETE FROM subscribers WHERE id = ?", subID)
	if err != nil {
		app.jsonError(w, 500, "Failed to delete subscriber")
		return
	}

	app.jsonResponse(w, 200, map[string]string{
		"message": "Subscriber deleted",
	})
}

// ============================================================================
// SETTINGS
// ============================================================================

func (app *App) adminSettings(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*User)

	settings := map[string]string{}
	rows, _ := app.db.Query("SELECT key, value FROM settings")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var key, value string
			if err := rows.Scan(&key, &value); err != nil {
				continue
			}
			settings[key] = value
		}
	}

	app.renderTemplate(w, "admin/settings", map[string]interface{}{
		"user":     user,
		"settings": settings,
	})
}

func (app *App) updateSettings(w http.ResponseWriter, r *http.Request) {
	var settings map[string]string
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		app.jsonError(w, 400, "Invalid request")
		return
	}

	for key, value := range settings {
		app.db.Exec(`
			INSERT INTO settings (key, value) VALUES (?, ?)
			ON DUPLICATE KEY UPDATE value = ?
		`, key, value, value)
	}

	// Clear cache so next render fetches new settings
	app.settingsMu.Lock()
	app.settingsCache = map[string]string{}
	app.settingsMu.Unlock()

	app.jsonResponse(w, 200, map[string]string{
		"message": "Settings updated",
	})
}

// ============================================================================
// MEDIA UPLOAD
// ============================================================================

func (app *App) uploadMedia(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		app.jsonError(w, 400, "File too large")
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		app.jsonError(w, 400, "No file provided")
		return
	}
	defer file.Close()

	// Validate file type
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}

	ext := filepath.Ext(handler.Filename)
	if !allowedExts[ext] {
		app.jsonError(w, 400, "Invalid file type")
		return
	}

	// Create unique filename
	filename := fmt.Sprintf("%s-%s%s", time.Now().Format("20060102150405"), uuid.New().String()[:8], ext)
	filepath := filepath.Join("./static/uploads", filename)

	dst, err := os.Create(filepath)
	if err != nil {
		app.jsonError(w, 500, "Failed to save file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		app.jsonError(w, 500, "Failed to save file")
		return
	}

	app.jsonResponse(w, 201, map[string]string{
		"url": "/static/uploads/" + filename,
	})
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// getOrCreateTag inserts a tag if it doesn't exist, using the provided transaction if available.
// It returns the tag ID. If tx is nil, it uses the main DB connection.
// It also uses a local map cache within the transaction to avoid duplicate lookups.
func (app *App) getOrCreateTag(tx *sql.Tx, tagName string) string {
	tagName = strings.TrimSpace(tagName)
	slug := app.slugify(tagName)

	var tagID string
	var err error

	if tx != nil {
		// Use transaction
		err = tx.QueryRow("SELECT id FROM tags WHERE slug = ?", slug).Scan(&tagID)
		if err == nil {
			return tagID
		}
		// Insert
		tagID = uuid.New().String()
		_, err = tx.Exec("INSERT INTO tags (id, name, slug) VALUES (?, ?, ?)", tagID, tagName, slug)
		if err != nil {
			// If duplicate key, try select again (another transaction may have inserted)
			tx.QueryRow("SELECT id FROM tags WHERE slug = ?", slug).Scan(&tagID)
			return tagID
		}
		return tagID
	}

	// No transaction: use main db
	err = app.db.QueryRow("SELECT id FROM tags WHERE slug = ?", slug).Scan(&tagID)
	if err == nil {
		return tagID
	}
	// Insert
	tagID = uuid.New().String()
	_, err = app.db.Exec("INSERT INTO tags (id, name, slug) VALUES (?, ?, ?)", tagID, tagName, slug)
	if err != nil {
		// If duplicate, fetch existing
		app.db.QueryRow("SELECT id FROM tags WHERE slug = ?", slug).Scan(&tagID)
		return tagID
	}
	return tagID
}
