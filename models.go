package main

import "time"

// User represents a user in the system
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Post represents a blog post
type Post struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Slug           string    `json:"slug"`
	Content        string    `json:"content"`
	Excerpt        string    `json:"excerpt"`
	FeaturedImage  string    `json:"featured_image"`
	AuthorID       string    `json:"author_id"`
	Author         *User     `json:"author,omitempty"`
	CategoryID     string    `json:"category_id,omitempty"`
	Category       *Category `json:"category,omitempty"`
	Tags           []Tag     `json:"tags,omitempty"`
	IsPublished    bool      `json:"is_published"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CommentCount   int       `json:"comment_count,omitempty"`
}

// Category represents a post category
type Category struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Tag represents a post tag
type Tag struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

// Comment represents a post comment
type Comment struct {
	ID         string    `json:"id"`
	PostID     string    `json:"post_id"`
	AuthorName string    `json:"author_name"`
	AuthorEmail string   `json:"author_email"`
	AuthorURL  string    `json:"author_url,omitempty"`
	Content    string    `json:"content"`
	IsApproved bool      `json:"is_approved"`
	CreatedAt  time.Time `json:"created_at"`
}

// Subscriber represents an email subscriber
type Subscriber struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	SubscribedAt   time.Time `json:"subscribed_at"`
	UnsubscribedAt *time.Time `json:"unsubscribed_at,omitempty"`
	IsActive      bool      `json:"is_active"`
}

// Settings represents site configuration
type Settings struct {
	SiteTitle       string `json:"site_title"`
	SiteDescription string `json:"site_description"`
	SiteURL         string `json:"site_url"`
	PostsPerPage    int    `json:"posts_per_page"`
	CommentsEnabled bool   `json:"comments_enabled"`
	ApprovalRequired bool  `json:"approval_required"`
	DarkModeDefault bool   `json:"dark_mode_default"`
}

// AuthResponse is returned after login
type AuthResponse struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

// RegisterRequest is for new user registration
type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginRequest is for user authentication
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// CommentRequest is for creating comments
type CommentRequest struct {
	AuthorName string `json:"author_name"`
	AuthorEmail string `json:"author_email"`
	AuthorURL  string `json:"author_url,omitempty"`
	Content    string `json:"content"`
}

// CreatePostRequest is for creating/updating posts
type CreatePostRequest struct {
	Title          string `json:"title"`
	Content        string `json:"content"`
	Excerpt        string `json:"excerpt"`
	FeaturedImage  string `json:"featured_image,omitempty"`
	CategoryID     string `json:"category_id,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	IsPublished    bool   `json:"is_published"`
}

// CreateCategoryRequest is for creating categories
type CreateCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CreateTagRequest is for creating tags
type CreateTagRequest struct {
	Name string `json:"name"`
}
