package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Authorization header is required",
				"error":   "Missing authorization token",
			})
			c.Abort()
			return
		}

		// Check if the Authorization header has the correct format (Bearer {token})
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Invalid authorization header format",
				"error":   "Use format: Bearer {token}",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]
		jwtSecret := []byte(os.Getenv("JWT_SECRET_KEY"))
		if len(jwtSecret) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Server misconfiguration: JWT secret not configured",
			})
			c.Abort()
			return
		}

		// Parse and validate the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Invalid or expired token",
				"error":   err.Error(),
			})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Invalid token claims",
				"error":   "Token validation failed",
			})
			c.Abort()
			return
		}

		userID, ok := claims["user_id"].(float64)
		if !ok || userID <= 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Invalid token claims",
				"error":   "Missing or invalid user_id claim",
			})
			c.Abort()
			return
		}

		email, ok := claims["email"].(string)
		if !ok || email == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Invalid token claims",
				"error":   "Missing or invalid email claim",
			})
			c.Abort()
			return
		}

		c.Set("user_id", uint(userID))
		c.Set("email", email)
		validRoles := map[string]bool{
			"USER": true, "DATA_SCIENTIST": true, "MEDICAL_EXPERT": true, "ADMIN": true,
		}
		if role, ok := claims["role"].(string); ok && validRoles[role] {
			c.Set("role", role)
		}
		c.Next()
	}
}

// RoleMiddleware restricts access to users with one of the specified roles.
// Must be used after AuthMiddleware so that "role" is already set in context.
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		roleName, ok := role.(string)
		if !exists || !ok || !allowed[roleName] {
			c.JSON(http.StatusForbidden, gin.H{
				"status":  "error",
				"message": "Access forbidden: insufficient role",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// CORSMiddleware handles CORS headers for cross-origin requests
func CORSMiddleware() gin.HandlerFunc {
	allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
	if allowedOriginsEnv == "" {
		allowedOriginsEnv = "http://localhost:3000"
	}

	allowedOrigins := make(map[string]bool)
	for _, origin := range strings.Split(allowedOriginsEnv, ",") {
		allowedOrigins[strings.TrimSpace(origin)] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if allowedOrigins[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,HEAD,OPTIONS,POST,PUT,PATCH,DELETE")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
