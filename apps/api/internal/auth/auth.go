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
	"net/http"
	"strings"
	"sync"
	"time"

	"bimanyaya/api/internal/db"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserKey contextKey = "user"

type ClerkClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	jwt.RegisteredClaims
}

type User struct {
	ID                string `json:"id"`
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

type AuthService struct {
	db             *db.DB
	clerkSecret    string
	clerkIssuer    string
	clerkJWKSURL   string
	convexURL      string
	jwksMu         sync.RWMutex
	jwksKeys       map[string]*rsa.PublicKey
	lastJWKSFetch  time.Time
}

func NewAuthService(database *db.DB, clerkSecret, clerkIssuer, clerkJWKSURL, convexURL string) *AuthService {
	return &AuthService{
		db:           database,
		clerkSecret:  clerkSecret,
		clerkIssuer:  clerkIssuer,
		clerkJWKSURL: clerkJWKSURL,
		convexURL:    convexURL,
		jwksKeys:     make(map[string]*rsa.PublicKey),
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
			// Validate algorithm is RS256
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
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

		// Resolve user profile via Convex database lookup or sync
		user, err := s.resolveUserProfile(r.Context(), claims)
		if err != nil {
			slog.Error("Failed to resolve user profile", "error", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to resolve user profile")
			return
		}

		ctx := context.WithValue(r.Context(), UserKey, user)
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
	if err != nil {
		// If query fails or returns null, we need to sync user profile in Convex
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
