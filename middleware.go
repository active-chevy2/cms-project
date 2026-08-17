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
			// If request expects JSON, return 401 JSON
			if strings.Contains(r.Header.Get("Accept"), "application/json") || r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
				app.jsonError(w, 401, "Unauthorized")
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(app.config.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			if strings.Contains(r.Header.Get("Accept"), "application/json") || r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
				app.jsonError(w, 401, "Invalid token")
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			if strings.Contains(r.Header.Get("Accept"), "application/json") || r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
				app.jsonError(w, 401, "Invalid token claims")
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok {
			if strings.Contains(r.Header.Get("Accept"), "application/json") || r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
				app.jsonError(w, 401, "Invalid user ID")
				return
			}
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
			if strings.Contains(r.Header.Get("Accept"), "application/json") || r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
				app.jsonError(w, 401, "User not found")
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Store user in context
		ctx := context.WithValue(r.Context(), "user", &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
