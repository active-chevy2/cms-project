package main

import (
	"database/sql"
	"log"
)

func runMigrations(db *sql.DB) error {
	migrations := []string{
		createUsersTable,
		createPostsTable,
		createCategoriesTable,
		createTagsTable,
		createPostTagsTable,
		createCommentsTable,
		createSubscribersTable,
		createSettingsTable,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			log.Printf("Migration error: %v", err)
			return err
		}
	}

	return nil
}

const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
	id VARCHAR(36) PRIMARY KEY,
	email VARCHAR(255) UNIQUE NOT NULL,
	password_hash VARCHAR(255) NOT NULL,
	username VARCHAR(100) UNIQUE NOT NULL,
	is_admin BOOLEAN DEFAULT false,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	INDEX idx_email (email),
	INDEX idx_username (username)
)
`

const createPostsTable = `
CREATE TABLE IF NOT EXISTS posts (
	id VARCHAR(36) PRIMARY KEY,
	title VARCHAR(255) NOT NULL,
	slug VARCHAR(255) UNIQUE NOT NULL,
	content LONGTEXT NOT NULL,
	excerpt VARCHAR(500),
	featured_image VARCHAR(255),
	author_id VARCHAR(36) NOT NULL,
	category_id VARCHAR(36),
	is_published BOOLEAN DEFAULT false,
	published_at TIMESTAMP NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL,
	INDEX idx_slug (slug),
	INDEX idx_published (is_published),
	INDEX idx_published_at (published_at),
	FULLTEXT INDEX idx_fulltext (title, content)
)
`

const createCategoriesTable = `
CREATE TABLE IF NOT EXISTS categories (
	id VARCHAR(36) PRIMARY KEY,
	name VARCHAR(100) NOT NULL UNIQUE,
	slug VARCHAR(100) UNIQUE NOT NULL,
	description TEXT,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	INDEX idx_slug (slug)
)
`

const createTagsTable = `
CREATE TABLE IF NOT EXISTS tags (
	id VARCHAR(36) PRIMARY KEY,
	name VARCHAR(100) NOT NULL UNIQUE,
	slug VARCHAR(100) UNIQUE NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	INDEX idx_slug (slug)
)
`

const createPostTagsTable = `
CREATE TABLE IF NOT EXISTS post_tags (
	post_id VARCHAR(36) NOT NULL,
	tag_id VARCHAR(36) NOT NULL,
	PRIMARY KEY (post_id, tag_id),
	FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
	FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
)
`

const createCommentsTable = `
CREATE TABLE IF NOT EXISTS comments (
	id VARCHAR(36) PRIMARY KEY,
	post_id VARCHAR(36) NOT NULL,
	author_name VARCHAR(100) NOT NULL,
	author_email VARCHAR(255) NOT NULL,
	author_url VARCHAR(255),
	content TEXT NOT NULL,
	is_approved BOOLEAN DEFAULT false,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
	INDEX idx_post_id (post_id),
	INDEX idx_approved (is_approved)
)
`

const createSubscribersTable = `
CREATE TABLE IF NOT EXISTS subscribers (
	id VARCHAR(36) PRIMARY KEY,
	email VARCHAR(255) UNIQUE NOT NULL,
	subscribed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	unsubscribed_at TIMESTAMP NULL,
	is_active BOOLEAN DEFAULT true,
	INDEX idx_email (email),
	INDEX idx_active (is_active)
)
`

const createSettingsTable = `
CREATE TABLE IF NOT EXISTS settings (
	key VARCHAR(100) PRIMARY KEY,
	value LONGTEXT NOT NULL,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
)
`
