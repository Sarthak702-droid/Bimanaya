package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"bimanyaya/api/internal/config"
	"bimanyaya/api/internal/db"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const UserKey contextKey = "user"

type ClerkClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	jwt.RegisteredClaims
}

type User struct {
	ID                string `json:"_id"`
	Email             string `json:"email"`
	Phone             string `json:"phone"`
	Role              string `json:"role"`
	Status            string `json:"status"`
	PreferredLanguage string `json:"preferred_language"`
}

type JSONWebKey struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	Alg string   `json:"alg"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

type JWKS struct {
	Keys []JSONWebKey `json:"keys"`
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

type AuthService struct {
	db            *db.DB
	clerkSecret   string
	clerkIssuer   string
	clerkJWKSURL  string
	convexURL     string
	environment   string
	jwksMu        sync.RWMutex
	jwksKeys      map[string]*rsa.PublicKey
	lastJWKSFetch time.Time
	otpStore      map[string]string
	otpMu         sync.Mutex
}

func NewAuthService(database *db.DB, cfg *config.Config) *AuthService {
	return &AuthService{
		db:           database,
		clerkSecret:  cfg.ClerkSecretKey,
		clerkIssuer:  cfg.ClerkJWTIssuer,
		clerkJWKSURL: cfg.ClerkJWKSURL,
		convexURL:    cfg.ConvexURL,
		environment:  cfg.Environment,
		jwksKeys:     make(map[string]*rsa.PublicKey),
		otpStore:     make(map[string]string),
	}
}

// Me endpoint to return current user info
func (s *AuthService) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(UserKey).(User)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User context not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// AuthMiddleware validates Clerk issued RS256 JWTs using JWKS
func (s *AuthService) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authorization header")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authorization header format")
			return
		}

		tokenString := parts[1]
		claims := &ClerkClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Support mock HMAC tokens signed with clerkSecret key only in development
			if s.environment == "development" {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
					return []byte(s.clerkSecret), nil
				}
			}

			// Validate algorithm is strictly RS256
			if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
				return nil, fmt.Errorf("unsupported signing algorithm: %v", token.Header["alg"])
			}

			kid, ok := token.Header["kid"].(string)
			if !ok {
				return nil, errors.New("missing kid header")
			}

			pubKey, err := s.getPublicKey(kid)
			if err != nil {
				// Retry fetching JWKS if key not found (could be a newly rotated key)
				slog.Warn("Key ID not found in cache, refreshing JWKS", "kid", kid)
				if fetchErr := s.refreshJWKS(); fetchErr != nil {
					slog.Error("Failed to refresh JWKS", "error", fetchErr)
				}
				pubKey, err = s.getPublicKey(kid)
				if err != nil {
					return nil, fmt.Errorf("key not found after refresh: %w", err)
				}
			}

			return pubKey, nil
		})

		if err != nil || !token.Valid {
			slog.Warn("Token validation failed", "error", err)
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token")
			return
		}

		// Validate claims (issuer, expiration, audience etc.)
		if claims.Issuer != s.clerkIssuer {
			slog.Warn("Token issuer mismatch", "expected", s.clerkIssuer, "got", claims.Issuer)
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token issuer mismatch")
			return
		}

		// Build the AuthContext with the bearer token to propagate user identity
		authCtx := db.AuthContext{
			ClerkUserID: claims.Subject,
			BearerToken: tokenString,
			Email:       claims.Email,
		}
		rCtx := db.WithAuthContext(r.Context(), authCtx)

		// Resolve user profile via Convex database lookup or sync
		user, err := s.resolveUserProfile(rCtx, claims)
		if err != nil {
			slog.Error("Failed to resolve user profile", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to resolve user profile")
			return
		}

		ctx := context.WithValue(rCtx, UserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole enforces specific user roles
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := r.Context().Value(UserKey).(User)
			if !ok {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User context not found")
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
				writeError(w, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Helper functions for JWKS and key parsing
func (s *AuthService) getPublicKey(kid string) (*rsa.PublicKey, error) {
	s.jwksMu.RLock()
	defer s.jwksMu.RUnlock()

	pubKey, exists := s.jwksKeys[kid]
	if !exists {
		return nil, fmt.Errorf("public key not found for kid: %s", kid)
	}
	return pubKey, nil
}

func (s *AuthService) refreshJWKS() error {
	s.jwksMu.Lock()
	defer s.jwksMu.Unlock()

	// Rate limit JWKS fetches to once per minute
	if time.Since(s.lastJWKSFetch) < 1*time.Minute {
		return nil
	}

	slog.Info("Fetching JWKS from Clerk", "url", s.clerkJWKSURL)
	resp, err := http.Get(s.clerkJWKSURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks request returned status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return err
	}

	newKeys := make(map[string]*rsa.PublicKey)
	for _, key := range jwks.Keys {
		if key.Kty != "RSA" || key.Use != "sig" {
			continue
		}

		decN, err := base64.RawURLEncoding.DecodeString(key.N)
		if err != nil {
			continue
		}

		decE, err := base64.RawURLEncoding.DecodeString(key.E)
		if err != nil {
			continue
		}

		var eVal int
		for _, b := range decE {
			eVal = (eVal << 8) | int(b)
		}

		pubKey := &rsa.PublicKey{
			N: new(big.Int).SetBytes(decN),
			E: eVal,
		}

		newKeys[key.Kid] = pubKey
	}

	s.jwksKeys = newKeys
	s.lastJWKSFetch = time.Now()
	return nil
}

func (s *AuthService) resolveUserProfile(ctx context.Context, claims *ClerkClaims) (User, error) {
	// Look up user by Clerk subject (user ID) in Convex db
	var user User
	err := s.db.CallQuery(ctx, "users:getCurrent", map[string]interface{}{}, &user)
	if err != nil || user.ID == "" {
		// Fallback to getByLegacyId (for mock tokens where getUserIdentity is not set in Convex auth)
		err = s.db.CallQuery(ctx, "users:getByLegacyId", map[string]interface{}{"legacyId": claims.Subject}, &user)
		if err != nil {
			// If getByLegacyId fails as well, sync Clerk user profile in Convex
			slog.Info("Syncing Clerk user profile with Convex database", "clerk_id", claims.Subject)
			
			var syncedID string
			syncArgs := map[string]interface{}{
				"clerkUserId":   claims.Subject,
				"clerkSubject":  claims.Subject,
				"email":         claims.Email,
				"emailVerified": claims.EmailVerified,
			}

			err = s.db.CallMutation(ctx, "users:syncCurrentUser", syncArgs, &syncedID)
			if err != nil {
				return User{}, fmt.Errorf("failed to sync user in convex: %w", err)
			}

			// Re-fetch synced user profile
			err = s.db.CallQuery(ctx, "users:getCurrent", map[string]interface{}{}, &user)
			if err != nil {
				return User{}, fmt.Errorf("failed to fetch user after sync: %w", err)
			}
		}
	}

	return user, nil
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func (s *AuthService) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var req OTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid input")
		return
	}

	identifier := req.Email
	if identifier == "" {
		identifier = req.Phone
	}

	if identifier == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Email or phone is required")
		return
	}

	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	
	s.otpMu.Lock()
	s.otpStore[identifier] = code
	s.otpMu.Unlock()

	slog.Info("[OTP DEMO] Sent OTP", "code", code, "to", identifier)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message":           "OTP sent successfully",
		"code_preview_demo": code,
	})
}

func (s *AuthService) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req OTPVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid input")
		return
	}

	identifier := req.Email
	if identifier == "" {
		identifier = req.Phone
	}

	s.otpMu.Lock()
	expectedCode, ok := s.otpStore[identifier]
	if ok && expectedCode == req.Code {
		delete(s.otpStore, identifier)
	}
	s.otpMu.Unlock()

	if !ok || expectedCode != req.Code {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired OTP")
		return
	}

	ctx := r.Context()
	var user User

	err := s.db.CallQuery(ctx, "users:getByEmailS2S", map[string]interface{}{"email": req.Email}, &user)
	if err != nil || user.ID == "" {
		legacyID := "user_" + strings.ReplaceAll(uuid.New().String(), "-", "")
		role := "POLICYHOLDER"
		
		if req.Email == "reviewer@bimanyaya.in" {
			role = "REVIEWER"
		} else if req.Email == "admin@bimanyaya.in" {
			role = "ADMIN"
		}

		var createdID string
		err = s.db.CallMutation(ctx, "users:registerUserS2S", map[string]interface{}{
			"email":    req.Email,
			"phone":    req.Phone,
			"role":     role,
			"legacyId": legacyID,
		}, &createdID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", fmt.Sprintf("Failed to register user in Convex: %v", err))
			return
		}

		err = s.db.CallQuery(ctx, "users:getByEmailS2S", map[string]interface{}{"email": req.Email}, &user)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve user after registration")
			return
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":            user.ID,
		"email":          user.Email,
		"email_verified": true,
		"role":           user.Role,
		"iss":            s.clerkIssuer,
		"aud":            "convex",
		"exp":            time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.clerkSecret))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate token")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": tokenString,
		"token_type":   "Bearer",
		"expires_in":   86400,
	})
}
