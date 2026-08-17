# Deployment Guide - Coolify

This CMS is optimized for **Coolify** deployment. Everything is configured through environment variables and the web admin panel—no file editing needed.

## Prerequisites

- Coolify instance running
- Git repository (push this code to GitHub/GitLab)
- Custom domain (optional but recommended)

## Step 1: Push to Git Repository

```bash
# Initialize git if needed
git init
git remote add origin <your-repo-url>

# Add all files
git add .
git commit -m "Initial CMS deployment"
git push origin main
```

## Step 2: Create Coolify Service

1. **Log in to Coolify Dashboard**
2. **Create New Project** (or use existing)
3. **Add New Service**
4. **Select Docker Compose**
5. **Connect Repository**
   - Choose your Git provider
   - Select this repository
   - Leave dockerfile/docker-compose path as default
6. **Click Create & Deploy**

Coolify will automatically:
- Build the Docker image
- Provision MariaDB database
- Create persistent volumes
- Set up networking

## Step 3: Configure Environment Variables

In Coolify, go to **Service Settings → Environment Variables** and add:

```bash
# Database (Coolify auto-creates these, adjust if needed)
DB_HOST=mariadb
DB_PORT=3306
DB_USER=cms
DB_PASS=<generate-strong-password>
DB_NAME=cms
DB_ROOT_PASSWORD=<generate-strong-password>

# Application
JWT_SECRET=<generate-32-char-random-string>
PORT=3000
ENVIRONMENT=production
SITE_URL=https://yourdomain.com
```

### Generate Secure Values

```bash
# Generate random password (copy output to DB_PASS)
openssl rand -base64 24

# Generate JWT secret (copy output to JWT_SECRET)
openssl rand -base64 32
```

## Step 4: Set Up Reverse Proxy

Coolify uses Traefik. No additional configuration needed if you're using Coolify's built-in SSL.

**If using custom domain:**
1. Add DNS record: `yourdomain.com → your-coolify-ip`
2. Coolify will auto-provision SSL via Let's Encrypt

## Step 5: First Time Setup

1. **Visit your domain** (once deployed)
2. **Create admin user** via `/api/auth/register`:
   ```bash
   curl -X POST https://yourdomain.com/api/auth/register \
     -H "Content-Type: application/json" \
     -d '{
       "email": "admin@example.com",
       "username": "admin",
       "password": "secure-password-min-8-chars"
     }'
   ```
3. **Save the returned JWT token**

3. **Configure site settings** at `/admin/settings` (using token from auth response)

## Monitoring & Logs

### In Coolify Dashboard

1. Go to your service
2. Click **Logs** tab
3. View real-time logs for app and database

### Common Issues

**Database connection error:**
- Check `DB_HOST=mariadb` (not `localhost`)
- Verify DB credentials match environment
- Wait 30 seconds after deployment for DB to be ready

**Admin panel won't load:**
- Verify JWT_SECRET is set and consistent
- Check token expiry (24 hours)
- Re-login at `/api/auth/login`

**Images not uploading:**
- Ensure `static/uploads/` exists (created automatically)
- Check file size < 32MB
- Verify disk space on server

## Auto-Deployments

Coolify can auto-deploy on git push. To enable:

1. Go to **Service Settings → Git**
2. **Enable Auto Deploy**
3. Set deployment branch (usually `main`)
4. Any push to that branch will trigger build & deploy

## Scaling

For high traffic:

1. **Use CDN** (Cloudflare recommended):
   - Point domain to Cloudflare
   - Cloudflare → Coolify server

2. **Database tuning:**
   - Adjust MariaDB max_connections in docker-compose.yml
   - Monitor query performance

3. **Multiple app instances:**
   - Edit docker-compose.yml
   - Run multiple `app` services
   - Use load balancer (Traefik handles this)

## Backups

### Automatic Database Backups

Add to your cron (host machine, not in container):

```bash
# Daily backup at 2 AM
0 2 * * * /home/user/backup-cms.sh
```

**backup-cms.sh:**
```bash
#!/bin/bash
BACKUP_DIR="/backups/cms"
mkdir -p $BACKUP_DIR
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

docker exec cms-db mysqldump -u cms -p${DB_PASS} cms > \
  $BACKUP_DIR/cms_$TIMESTAMP.sql

# Keep only last 30 days
find $BACKUP_DIR -mtime +30 -delete
```

### Manual Backup

```bash
# From Coolify host
docker exec cms-db mysqldump -u cms -p<password> cms > backup.sql
```

### Restore from Backup

```bash
docker exec -i cms-db mysql -u cms -p<password> cms < backup.sql
```

## Maintenance

### Update Application

1. Make changes locally
2. Commit and push to Git
3. Coolify auto-deploys (if enabled)

### Manual Redeploy

In Coolify dashboard → Service → **Redeploy**

### Database Migrations

Handled automatically on startup. No manual action needed.

## Security Checklist

- [ ] JWT_SECRET is random 32+ characters
- [ ] Database password is strong (16+ chars, mixed case)
- [ ] SITE_URL points to your domain
- [ ] ENVIRONMENT=production (not development)
- [ ] SSL certificate auto-provisioned (green lock icon)
- [ ] Backups enabled and tested
- [ ] Regular password rotation for admin users

## Performance Tuning

### Image Optimization

Upload optimized images. CMS stores originals as-is.

```bash
# Optimize before upload
convert input.jpg -quality 80 -resize 1920x1080 output.jpg
```

### Disable Features You Don't Use

In **Admin → Settings**, disable:
- Comments (if not using)
- Comment approval (if you trust commenters)

### Database Indexes

Already optimized with full-text search indexes on posts.

## Troubleshooting

### Deployment Stuck

Check logs in Coolify dashboard. Common issues:
- `docker pull` timeout → increase timeout
- Disk full → free up space
- Memory low → check system resources

### App crashes after deploy

1. Check logs: `docker-compose logs app`
2. Verify all env vars are set
3. Check database is running: `docker-compose logs db`
4. Restart: `docker-compose restart`

### Comments not working

Verify in **Admin → Settings**:
- Comments Enabled: ON
- Require Approval: depends on your preference

### Search returns no results

Full-text search uses MySQL BOOLEAN MODE:
- Use `+word` for must-match
- Use `-word` for exclude
- Use `"phrase"` for exact match

Examples:
- `/search?q=+golang` — find golang
- `/search?q=-draft` — exclude draft
- `/search?q="go programming"` — exact phrase

## Getting Help

1. Check [README.md](README.md) for general info
2. Review Coolify docs: https://coolify.io/docs
3. Check logs for specific errors
4. Common issues resolved in troubleshooting section

## Next Steps

1. Create first post: **Admin → Posts → Create New Post**
2. Set up categories: **Admin → Categories**
3. Invite collaborators (future feature)
4. Set up analytics (external service)
5. Configure email notifications (future feature)

---

**Happy publishing!** 🚀

This CMS is designed to be low-maintenance and production-ready on Coolify. Everything else is just content.
