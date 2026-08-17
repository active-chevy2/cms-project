# Quick Start Guide

Get your CMS running in minutes.

## Option 1: Docker (Recommended)

```bash
# 1. Clone or download this repository
cd cms-project

# 2. Copy and edit environment file
cp .env.example .env
# Edit .env with your settings

# 3. Start with docker-compose
docker-compose up -d

# 4. Wait 30 seconds for database to initialize

# 5. Visit http://localhost:3000
```

## Option 2: Local Development

```bash
# 1. Install Go 1.21+
# (https://golang.org/dl)

# 2. Start MariaDB (or use Docker)
# docker run -d -e MYSQL_ROOT_PASSWORD=root -p 3306:3306 mariadb:11

# 3. Clone and setup
cd cms-project
cp .env.example .env

# 4. Edit .env and set DB_HOST=localhost

# 5. Run
go run .

# 6. Visit http://localhost:3000
```

## First Time Setup

### 1. Register Admin User

Visit http://localhost:3000/api/auth/register (or use curl):

```bash
curl -X POST http://localhost:3000/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "username": "admin",
    "password": "securepassword123"
  }'
```

**Response:**
```json
{
  "token": "eyJhbGc...",
  "message": "User registered successfully"
}
```

Save the token for the next step.

### 2. Access Admin Panel

Visit: http://localhost:3000/admin

Add token to request headers:
```
Authorization: Bearer <your-token-from-above>
```

Or use the login endpoint:

```bash
curl -X POST http://localhost:3000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "securepassword123"
  }'
```

### 3. Configure Site Settings

1. Go to **Admin → Settings**
2. Set:
   - Site Title
   - Site Description
   - Site URL (important for SEO)
   - Comment moderation preferences
3. Click **Save**

### 4. Create Your First Post

1. Go to **Admin → Posts**
2. Click **Create New Post**
3. Write in markdown:
   ```markdown
   # My First Post

   This is **bold** and this is *italic*.

   ## Subheading

   - List item 1
   - List item 2

   [Link to example.com](https://example.com)
   ```
4. Set category and tags
5. Click **Publish**
6. Visit your domain to see it live!

## Common Tasks

### Add a Category

1. **Admin → Categories**
2. Enter name and description
3. Click **Add Category**

### Add a Tag

Tags are created automatically when you publish posts, or:

1. **Admin → Tags**
2. Enter tag name
3. Click **Add Tag**

### Upload Images

In post editor:
1. Click **Upload Image** button
2. Select JPG/PNG/GIF/WebP
3. Copy the returned URL into the featured image field

### Moderate Comments

1. **Admin → Comments**
2. Review pending comments
3. Click **Approve** or **Delete**

### Export Subscribers

1. **Admin → Subscribers**
2. Click **Export as CSV**
3. Use CSV with your email service (Mailchimp, SendGrid, etc.)

## Deployment to Coolify

See [DEPLOYMENT.md](DEPLOYMENT.md) for full instructions.

Quick version:

1. Push to Git repository
2. Create Coolify service (Docker Compose)
3. Set environment variables
4. Done! Auto-deploys on git push

## Troubleshooting

**Can't connect to admin panel?**
- Check token is valid (24 hour expiry)
- Re-login with `/api/auth/login`
- Verify Authorization header format: `Bearer <token>`

**Database error?**
- Check MariaDB is running
- Verify DB credentials in `.env`
- Docker: `docker-compose logs db`

**Images not uploading?**
- Supported formats: JPG, PNG, GIF, WebP
- Max size: 32MB
- Directory must exist: `static/uploads/`

**Search not working?**
- Full-text search requires MariaDB
- Try simpler queries first
- Use `+keyword` syntax for exact match

## Next Steps

- [ ] Configure site settings
- [ ] Create first post
- [ ] Add categories/tags
- [ ] Test comments
- [ ] Deploy to Coolify or your server
- [ ] Set up SSL/HTTPS
- [ ] Enable backups

## API Reference

### Public Endpoints

```
GET  /                    # Homepage
GET  /posts               # All posts (paginated)
GET  /posts/{slug}        # Single post
GET  /category/{slug}     # Posts by category
GET  /tags/{slug}         # Posts by tag
GET  /search?q=query      # Search posts
GET  /sitemap.xml         # SEO sitemap
GET  /robots.txt          # Robots file
POST /subscribe           # Subscribe to newsletter
POST /api/comments        # Create comment
GET  /api/comments/{id}   # Get comments
```

### Auth Endpoints

```
POST /api/auth/register   # Register new user (first user is admin)
POST /api/auth/login      # Login
```

### Admin Endpoints (requires Bearer token)

```
GET    /admin/                       # Dashboard
GET    /admin/posts                  # List posts
POST   /admin/posts                  # Create post
GET    /admin/posts/{id}/edit        # Edit form
POST   /admin/posts/{id}             # Update post
POST   /admin/posts/{id}/delete      # Delete post
POST   /admin/posts/{id}/publish     # Toggle publish
GET    /admin/categories             # Manage categories
POST   /admin/categories             # Create category
GET    /admin/tags                   # Manage tags
POST   /admin/tags                   # Create tag
GET    /admin/comments               # Moderate comments
POST   /admin/comments/{id}/approve  # Approve comment
GET    /admin/subscribers            # List subscribers
GET    /admin/subscribers/export     # Export as CSV
POST   /admin/upload                 # Upload media
GET    /admin/settings               # View settings
POST   /admin/settings               # Update settings
```

## Support

- Full documentation: [README.md](README.md)
- Deployment guide: [DEPLOYMENT.md](DEPLOYMENT.md)
- Issues? Check browser console for errors
- Server logs: `docker-compose logs app`

---

**Ready to publish?** Start creating posts! 🚀
