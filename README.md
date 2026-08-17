# Markdown CMS

A minimalist, production-grade markdown publishing platform built with Go, MariaDB, and vanilla HTML/CSS/JS. Designed for easy deployment on Coolify with zero file-based configuration—everything managed through the web admin panel.

## Features

- ✍️ **Markdown Publishing** — Write in markdown, publish with one click
- 🎨 **Minimalist Design** — Clean, readable interface inspired by Ghost Source and standard.site
- 👤 **User Management** — First user registered becomes admin automatically
- 💬 **Built-in Comments** — Moderated comment system with admin approval
- 📧 **Newsletter System** — Collect and export subscriber emails
- 🏷️ **Tags & Categories** — Organize content with flexible taxonomy
- 🔍 **Full-Text Search** — Fast search across all content
- 🌙 **Dark Mode** — Automatic dark mode toggle with localStorage
- 📱 **Mobile Responsive** — Works seamlessly on all devices
- 🔒 **JWT Authentication** — Secure token-based admin access
- 📄 **SEO Ready** — Auto-generated sitemaps, robots.txt, OG meta tags
- 🚀 **Coolify Ready** — Docker Compose with environment-based config

## Tech Stack

- **Backend**: Go 1.21+ with chi router
- **Database**: MariaDB 11
- **Frontend**: HTML5 + CSS3 + Vanilla JavaScript
- **Deployment**: Docker + Docker Compose
- **Markdown**: goldmark parser

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Git

### Local Development

```bash
# Clone the repository
git clone <repo-url>
cd cms-project

# Copy environment file
cp .env.example .env

# Edit .env with your settings
nano .env

# Start with Docker Compose
docker-compose up --build

# Access the app
# Frontend: http://localhost:3000
# Admin: http://localhost:3000/admin
```

### First-Time Setup

1. **Register Admin User**
   - Visit `http://localhost:3000/api/auth/register`
   - Enter email, username, password (min 8 chars)
   - First user automatically becomes admin
   - Save the JWT token provided

2. **Access Admin Panel**
   - Visit `http://localhost:3000/admin`
   - Use the token as Bearer authentication in Authorization header
   - Or log in via `/api/auth/login`

3. **Configure Site Settings**
   - Go to `/admin/settings`
   - Set site title, description, URL
   - Configure comment moderation
   - Enable/disable features as needed

## Deployment

### Coolify Deployment

1. **Prepare Repository**
   ```bash
   git init
   git add .
   git commit -m "Initial commit"
   git push origin main
   ```

2. **Create Coolify Service**
   - Go to Coolify dashboard
   - Create new service
   - Select "Docker Compose"
   - Point to your repository
   - Add environment variables (see `.env.example`)

3. **Key Environment Variables**
   ```
   DB_HOST=mariadb
   DB_PORT=3306
   DB_USER=cms
   DB_PASS=<strong-password>
   DB_NAME=cms
   DB_ROOT_PASSWORD=<strong-password>
   JWT_SECRET=<random-string>
   PORT=3000
   ENVIRONMENT=production
   SITE_URL=https://yourdomain.com
   ```

4. **Deploy**
   - Coolify will auto-detect docker-compose.yml
   - Provisions MariaDB automatically
   - Builds and deploys Go application
   - Creates persistent volumes for database

### Manual Docker Deployment

```bash
# Build image
docker build -t cms:latest .

# Run with docker-compose
docker-compose up -d

# Or run standalone
docker run -e DB_HOST=<host> -e DB_USER=cms -e DB_PASS=<pass> \
  -p 3000:3000 cms:latest
```

### Production Checklist

- [ ] Change `JWT_SECRET` to a random 32+ character string
- [ ] Use strong database password
- [ ] Set `ENVIRONMENT=production`
- [ ] Point `SITE_URL` to your domain
- [ ] Enable HTTPS on your reverse proxy (Nginx, Caddy)
- [ ] Set up database backups
- [ ] Review and adjust comment moderation settings
- [ ] Test all functionality before making public

## API Endpoints

### Public

- `GET /` — Homepage
- `GET /posts` — List posts
- `GET /posts/{slug}` — View single post
- `GET /category/{slug}` — Posts by category
- `GET /tags/{slug}` — Posts by tag
- `GET /search?q=query` — Search posts
- `GET /sitemap.xml` — SEO sitemap
- `GET /robots.txt` — Robots file
- `GET /subscribe` — Newsletter signup form
- `POST /subscribe` — Subscribe to newsletter
- `POST /api/comments?post_id=<id>` — Create comment
- `GET /api/comments/{postID}` — Get approved comments

### Authentication

- `POST /api/auth/register` — Register new user (first user becomes admin)
- `POST /api/auth/login` — Login with email/password

### Admin (Requires JWT Token)

- `GET /admin` — Dashboard
- `GET /admin/posts` — List all posts
- `POST /admin/posts` — Create post
- `GET /admin/posts/{id}/edit` — Edit form
- `POST /admin/posts/{id}` — Update post
- `POST /admin/posts/{id}/delete` — Delete post
- `POST /admin/posts/{id}/publish` — Toggle publish status
- `GET /admin/categories` — Manage categories
- `POST /admin/categories` — Create category
- `POST /admin/categories/{id}/delete` — Delete category
- `GET /admin/tags` — Manage tags
- `POST /admin/tags` — Create tag
- `POST /admin/tags/{id}/delete` — Delete tag
- `GET /admin/comments` — Moderate comments
- `POST /admin/comments/{id}/approve` — Approve comment
- `POST /admin/comments/{id}/delete` — Delete comment
- `GET /admin/subscribers` — List subscribers
- `GET /admin/subscribers/export` — Export as CSV
- `POST /admin/subscribers/{id}/delete` — Remove subscriber
- `GET /admin/settings` — View settings
- `POST /admin/settings` — Update settings
- `POST /admin/upload` — Upload media file

## Database Schema

### Users
- `id` (UUID)
- `email` (unique)
- `username` (unique)
- `password_hash`
- `is_admin` (boolean)
- `created_at`, `updated_at`

### Posts
- `id` (UUID)
- `title`, `slug` (unique)
- `content` (markdown)
- `excerpt`
- `featured_image`
- `author_id` (FK users)
- `category_id` (FK categories, nullable)
- `is_published` (boolean)
- `published_at` (nullable)
- `created_at`, `updated_at`
- **Full-text index on title + content**

### Categories
- `id` (UUID)
- `name`, `slug` (unique)
- `description`
- `created_at`

### Tags
- `id` (UUID)
- `name`, `slug` (unique)
- `created_at`

### Post_Tags (Junction)
- `post_id` (FK posts)
- `tag_id` (FK tags)

### Comments
- `id` (UUID)
- `post_id` (FK posts)
- `author_name`, `author_email`, `author_url`
- `content`
- `is_approved` (boolean)
- `created_at`

### Subscribers
- `id` (UUID)
- `email` (unique)
- `subscribed_at`
- `unsubscribed_at` (nullable)
- `is_active` (boolean)

### Settings
- `key` (primary, string)
- `value` (longtext)
- `updated_at`

## Configuration via Admin Panel

No config files to edit! Everything is managed through the web admin interface:

1. **Site Settings**
   - Site title & description
   - Site URL
   - Comment moderation policy
   - Dark mode default

2. **Post Management**
   - Create, edit, delete posts
   - Publish/unpublish without rebuilding
   - Drag-drop media uploads
   - Tag and categorize automatically

3. **User Management**
   - First user is automatically admin
   - Admin-only access to /admin routes

4. **Content Management**
   - Categories (create, edit, delete)
   - Tags (auto-created from posts)
   - Full-text search

5. **Moderation**
   - Approve/reject comments
   - Export subscribers as CSV
   - Remove subscribers

## Development

### Building from Source

```bash
# Download dependencies
go mod download

# Run locally
go run .

# Or build binary
go build -o cms

# Run binary
./cms
```

### Environment Variables

```bash
DB_HOST=localhost          # Database host
DB_PORT=3306              # Database port
DB_USER=cms               # Database user
DB_PASS=password          # Database password
DB_NAME=cms               # Database name
JWT_SECRET=secret         # JWT signing secret (min 16 chars)
PORT=3000                 # Application port
ENVIRONMENT=development   # development or production
SITE_URL=http://localhost:3000
```

### Project Structure

```
.
├── main.go                # Application entry point
├── models.go              # Data models
├── handlers.go            # Public route handlers
├── admin.go               # Admin panel handlers
├── middleware.go          # Authentication & rendering
├── migrations.go          # Database schema
├── go.mod, go.sum         # Dependencies
├── Dockerfile             # Docker build configuration
├── docker-compose.yml     # Multi-container setup
├── static/
│   ├── style.css         # Global styles
│   ├── main.js           # Client-side JavaScript
│   └── uploads/          # Media uploads (volume)
├── templates/
│   ├── base.html         # Base layout
│   ├── home.html         # Homepage
│   ├── post.html         # Single post
│   ├── posts-list.html   # Post listing
│   └── admin/
│       ├── dashboard.html
│       ├── posts-list.html
│       ├── post-form.html
│       └── ...
├── .env.example          # Environment template
├── .dockerignore         # Docker build ignore
├── .gitignore            # Git ignore patterns
└── README.md             # This file
```

## Troubleshooting

### Database Connection Error
- Check `DB_HOST` matches service name in docker-compose
- Verify database credentials in `.env`
- Ensure MariaDB is running: `docker-compose ps`

### Admin Panel Access Denied
- Verify JWT token in Authorization header
- Check token hasn't expired (24 hours)
- Re-login via `/api/auth/login`

### Images Not Uploading
- Check `static/uploads/` directory exists and is writable
- Verify file size under 32MB limit
- Supported formats: jpg, jpeg, png, gif, webp

### Comments Not Appearing
- Verify `is_approved` is true (if moderation enabled)
- Check post is published
- Comments require approval before display

## Performance Tips

1. **Database Optimization**
   - MariaDB full-text search indexes pre-created
   - Pagination defaults to 10 posts/page
   - Add database backups

2. **Caching**
   - Static assets served with proper cache headers
   - Consider Cloudflare or similar CDN
   - Published posts are read-only (no real-time queries)

3. **Scaling**
   - Stateless app (can run multiple instances)
   - Use load balancer (Nginx, Caddy)
   - Database is the bottleneck at scale

## Security

- Passwords hashed with bcrypt (10 rounds)
- JWT tokens expire after 7 days
- CSRF protection via SameSite cookies
- SQL injection protection via parameterized queries
- XSS protection via HTML escaping in templates
- Only first user becomes admin (no public registration)

## License

MIT License — feel free to modify and redistribute.

## Support

For issues, questions, or contributions:
1. Check existing GitHub issues
2. Review configuration documentation
3. Ensure environment variables are set correctly
4. Check MariaDB logs: `docker-compose logs db`
5. Check app logs: `docker-compose logs app`

## Contributing

This project welcomes contributions! Please:
1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push and create a pull request
5. Ensure all tests pass

---

**Built with ❤️ for minimalist publishing.**

Designed to work with Coolify for one-click deployment. No file editing needed post-launch—everything configurable through the web admin panel.
