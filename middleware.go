package main

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func (app *App) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			authHeader = r.URL.Query().Get("token")
		}

		if authHeader == "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(app.config.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Get user from database
		var user User
		err = app.db.QueryRow(
			"SELECT id, email, username, is_admin FROM users WHERE id = ?",
			userID,
		).Scan(&user.ID, &user.Email, &user.Username, &user.IsAdmin)

		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Store user in context
		ctx := context.WithValue(r.Context(), "user", &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *App) renderTemplate(w http.ResponseWriter, name string, data map[string]interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}

	templatePath := "./templates/" + name + ".html"

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Printf("Template error: %v", err)
		app.renderError(w, 500, "Template error")
		return
	}

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
