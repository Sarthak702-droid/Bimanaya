package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"bimanyaya/api/internal/db"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type AuthService struct {
	db        *db.DB
	jwtSecret []byte
	// Memory store for OTPs for local development. In production, Redis would store this.
	otpStore map[string]string 
}

func NewAuthService(database *db.DB, jwtSecret string) *AuthService {
	return &AuthService{
		db:        database,
		jwtSecret: []byte(jwtSecret),
		otpStore:  make(map[string]string),
	}
}

type OTPRequest struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type OTPVerifyRequest struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type User struct {
	ID                string `json:"id"`
	Email             string `json:"email"`
	Phone             string `json:"phone"`
	Role              string `json:"role"`
	Status            string `json:"status"`
	PreferredLanguage string `json:"preferred_language"`
}

func (s *AuthService) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req OTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	identifier := req.Email
	if identifier == "" {
		identifier = req.Phone
	}

	if identifier == "" {
		http.Error(w, "Email or phone is required", http.StatusBadRequest)
		return
	}

	// Generate 6 digit code
	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	s.otpStore[identifier] = code

	// In real application, send via email or SMS. For demo, we return it or log it.
	fmt.Printf("[OTP DEMO] Sent OTP %s to %s\n", code, identifier)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "OTP sent successfully (check backend log for code in demo mode)",
		"code_preview_demo": code, // Returning code directly for ease of API demonstration
	})
}

func (s *AuthService) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req OTPVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	identifier := req.Email
	if identifier == "" {
		identifier = req.Phone
	}

	expectedCode, ok := s.otpStore[identifier]
	if !ok || expectedCode != req.Code {
		http.Error(w, "Invalid or expired OTP", http.StatusUnauthorized)
		return
	}

	// Clear OTP
	delete(s.otpStore, identifier)

	ctx := r.Context()
	var user User

	// Check if user exists, else create
	var err error
	if req.Email != "" && req.Phone != "" {
		err = s.db.Pool.QueryRow(ctx, 
			"SELECT id, email, COALESCE(phone, ''), role, status, preferred_language FROM users WHERE email = $1 OR phone = $2", 
			req.Email, req.Phone).Scan(&user.ID, &user.Email, &user.Phone, &user.Role, &user.Status, &user.PreferredLanguage)
	} else if req.Email != "" {
		err = s.db.Pool.QueryRow(ctx, 
			"SELECT id, email, COALESCE(phone, ''), role, status, preferred_language FROM users WHERE email = $1", 
			req.Email).Scan(&user.ID, &user.Email, &user.Phone, &user.Role, &user.Status, &user.PreferredLanguage)
	} else {
		err = s.db.Pool.QueryRow(ctx, 
			"SELECT id, email, COALESCE(phone, ''), role, status, preferred_language FROM users WHERE phone = $1", 
			req.Phone).Scan(&user.ID, &user.Email, &user.Phone, &user.Role, &user.Status, &user.PreferredLanguage)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		// Register new user
		user.ID = uuid.New().String()
		user.Email = req.Email
		user.Phone = req.Phone
		user.Role = "POLICYHOLDER"
		user.Status = "ACTIVE"
		user.PreferredLanguage = "en"

		var emailVal *string
		if req.Email != "" {
			emailVal = &req.Email
		}
		var phoneVal *string
		if req.Phone != "" {
			phoneVal = &req.Phone
		}

		_, err = s.db.Pool.Exec(ctx, 
			"INSERT INTO users (id, email, phone, role, status, preferred_language) VALUES ($1, $2, $3, $4, $5, $6)",
			user.ID, emailVal, phoneVal, user.Role, user.Status, user.PreferredLanguage)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create user: %v", err), http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Auto-promote test roles in Convex
	if user.Email == "reviewer@bimanyaya.in" && user.Role != "REVIEWER" {
		user.Role = "REVIEWER"
		s.db.Pool.Exec(ctx, "UPDATE users SET role = $1 WHERE email = $2", "REVIEWER", user.Email)
	}
	if user.Email == "admin@bimanyaya.in" && user.Role != "ADMIN" {
		user.Role = "ADMIN"
		s.db.Pool.Exec(ctx, "UPDATE users SET role = $1 WHERE email = $2", "ADMIN", user.Email)
	}

	// Generate JWT Access Token
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		http.Error(w, "Token generation failed", http.StatusInternalServerError)
		return
	}

	// Generate Refresh Token
	refreshToken := uuid.New().String()
	_, err = s.db.Pool.Exec(ctx,
		"INSERT INTO sessions (user_id, refresh_token, ip_address, user_agent, expires_at) VALUES ($1, $2, $3, $4, $5)",
		user.ID, refreshToken, r.RemoteAddr, r.UserAgent(), time.Now().Add(30*24*time.Hour))
	if err != nil {
		http.Error(w, "Session creation failed", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(TokenResponse{
		AccessToken:  tokenString,
		RefreshToken: refreshToken,
		User:         user,
	})
}

func (s *AuthService) Logout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Simple logout: in production we revoke the refresh token and optionally blacklist the access token
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully"})
}

func (s *AuthService) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	json.NewEncoder(w).Encode(user)
}

// Middleware to secure endpoints
func (s *AuthService) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return s.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Fetch current user details
		var user User
		err = s.db.Pool.QueryRow(r.Context(),
			"SELECT id, COALESCE(email, ''), COALESCE(phone, ''), role, status, preferred_language FROM users WHERE id = $1",
			claims.UserID).Scan(&user.ID, &user.Email, &user.Phone, &user.Role, &user.Status, &user.PreferredLanguage)

		if err != nil {
			http.Error(w, "User not found or database error", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole enforces specific user roles
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := r.Context().Value("user").(User)
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			slog.Info("Checking user role compatibility", "user_role", user.Role, "required_roles", roles)

			allowed := false
			for _, role := range roles {
				if user.Role == role {
					allowed = true
					break
				}
			}

			if !allowed {
				http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
